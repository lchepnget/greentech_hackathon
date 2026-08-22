package actions

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"backend/models"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

var frontStore = struct {
	sync.RWMutex
	listings, orders, invoices []map[string]interface{}
}{}

func frontJSON(c buffalo.Context, v interface{}) error { return c.Render(http.StatusOK, r.JSON(v)) }
func FrontendListings(c buffalo.Context) error {
	frontStore.RLock()
	defer frontStore.RUnlock()
	return frontJSON(c, map[string]interface{}{"items": frontStore.listings, "page": 1, "pages": 1, "total": len(frontStore.listings)})
}
func FrontendListingSummary(c buffalo.Context) error {
	frontStore.RLock()
	defer frontStore.RUnlock()
	return frontJSON(c, map[string]interface{}{"totalSatsMoved": 0, "newListings": len(frontStore.listings), "regions": []string{}})
}
func FrontendListingByID(c buffalo.Context) error {
	id := c.Param("id")
	frontStore.RLock()
	defer frontStore.RUnlock()
	for _, x := range frontStore.listings {
		if x["id"] == id {
			return frontJSON(c, x)
		}
	}
	return c.Render(http.StatusNotFound, r.JSON(map[string]string{"error": "listing not found"}))
}
func FrontendCreateListing(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "authentication required"}))
	}
	var user models.User
	if err := models.DB.Find(&user, userID); err != nil || strings.ToUpper(user.Role) != "PRODUCER" {
		return c.Render(http.StatusForbidden, r.JSON(map[string]string{"error": "only producers may create listings"}))
	}
	x := map[string]interface{}{}
	if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(10 << 20); err != nil {
			return c.Render(400, r.JSON(map[string]string{"error": "invalid multipart payload"}))
		}
		for _, key := range []string{"title", "wasteType", "quantity", "unit", "priceSats", "location", "description"} {
			x[key] = c.Request().FormValue(key)
		}
	} else if err := json.NewDecoder(c.Request().Body).Decode(&x); err != nil {
		return c.Render(400, r.JSON(map[string]string{"error": "invalid payload"}))
	}
	x["id"] = fmt.Sprintf("listing_%d", time.Now().UnixNano())
	if userID, ok := c.Value("user_id").(uuid.UUID); ok {
		x["ownerId"] = userID.String()
		if tx, ok := c.Value("tx").(*pop.Connection); ok {
			user := &models.User{}
			if err := tx.Find(user, userID); err == nil {
				x["producerName"] = strings.TrimSpace(user.FirstName + " " + user.LastName)
				if x["location"] == "" {
					x["location"] = user.Location
				}
			}
		}
	}
	frontStore.Lock()
	frontStore.listings = append(frontStore.listings, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}

func HandleCreateListingInvoice(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(401, r.JSON(map[string]string{"error": "authentication required"}))
	}
	var user models.User
	if err := models.DB.Find(&user, userID); err != nil || strings.ToUpper(user.Role) != "PRODUCER" {
		return c.Render(403, r.JSON(map[string]string{"error": "only producers may create listings"}))
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		PriceSats   int64  `json:"priceSats"`
		WasteType   string `json:"wasteType"`
		Quantity    string `json:"quantity"`
		Unit        string `json:"unit"`
		Location    string `json:"location"`
	}
	var imageData string
	if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(6 << 20); err != nil {
			return c.Render(400, r.JSON(map[string]string{"error": "invalid multipart payload"}))
		}
		req.Name, req.Description, req.WasteType, req.Quantity, req.Unit, req.Location = c.Request().FormValue("title"), c.Request().FormValue("description"), c.Request().FormValue("wasteType"), c.Request().FormValue("quantity"), c.Request().FormValue("unit"), c.Request().FormValue("location")
		if _, err := fmt.Sscan(c.Request().FormValue("priceSats"), &req.PriceSats); err != nil {
			req.PriceSats = 0
		}
		if file, header, err := c.Request().FormFile("photos"); err == nil {
			defer file.Close()
			if header.Size > 5<<20 {
				return c.Render(413, r.JSON(map[string]string{"error": "image must be 5 MB or smaller"}))
			}
			content, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
			if err != nil || len(content) > 5<<20 {
				return c.Render(400, r.JSON(map[string]string{"error": "unable to read image"}))
			}
			imageData = "data:" + header.Header.Get("Content-Type") + ";base64," + base64.StdEncoding.EncodeToString(content)
		}
	} else if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.Render(400, r.JSON(map[string]string{"error": "invalid payload"}))
	}
	if strings.TrimSpace(req.Name) == "" || req.PriceSats < 1 {
		return c.Render(400, r.JSON(map[string]string{"error": "name and positive priceSats are required"}))
	}
	inv, err := createBlinkDepositInvoice(c.Request().Context(), req.PriceSats)
	if err != nil {
		return c.Render(502, r.JSON(map[string]string{"error": "unable to create Blink invoice"}))
	}
	if strings.TrimSpace(inv.PaymentRequest) == "" {
		return c.Render(502, r.JSON(map[string]string{"error": "Blink returned an empty invoice"}))
	}
	x := map[string]interface{}{"id": fmt.Sprintf("listing_%d", time.Now().UnixNano()), "ownerId": userID.String(), "producerName": strings.TrimSpace(user.FirstName + " " + user.LastName), "title": req.Name, "description": req.Description, "wasteType": req.WasteType, "quantity": req.Quantity, "unit": req.Unit, "location": req.Location, "priceSats": req.PriceSats, "bolt11": inv.PaymentRequest, "invoice": inv.PaymentRequest, "status": "active"}
	if imageData != "" {
		x["image"] = imageData
	}
	frontStore.Lock()
	frontStore.listings = append(frontStore.listings, x)
	frontStore.Unlock()
	return c.Render(201, r.JSON(x))
}

func FrontendDeleteListing(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(401, r.JSON(map[string]string{"error": "authentication required"}))
	}
	id := c.Param("id")
	frontStore.Lock()
	defer frontStore.Unlock()
	for i, listing := range frontStore.listings {
		if listing["id"] == id {
			if listing["ownerId"] != userID.String() {
				return c.Render(403, r.JSON(map[string]string{"error": "you can only delete your own listings"}))
			}
			frontStore.listings = append(frontStore.listings[:i], frontStore.listings[i+1:]...)
			return c.Render(204, r.String(""))
		}
	}
	return c.Render(404, r.JSON(map[string]string{"error": "listing not found"}))
}

func HandleVerifyPayment(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(401, r.JSON(map[string]string{"error": "authentication required"}))
	}
	var user models.User
	if err := models.DB.Find(&user, userID); err != nil || strings.ToUpper(user.Role) != "FARMER" {
		return c.Render(403, r.JSON(map[string]string{"error": "only farmers may pay invoices"}))
	}
	var req struct {
		Bolt11 string `json:"bolt11"`
	}
	_ = json.NewDecoder(c.Request().Body).Decode(&req)
	frontStore.RLock()
	defer frontStore.RUnlock()
	for _, x := range frontStore.listings {
		if x["bolt11"] == req.Bolt11 {
			return frontJSON(c, map[string]interface{}{"bolt11": req.Bolt11, "status": "pending", "settled": false})
		}
	}
	return c.Render(404, r.JSON(map[string]string{"error": "invoice not found"}))
}
func FrontendOrders(c buffalo.Context) error {
	userID, _ := c.Value("user_id").(uuid.UUID)
	frontStore.RLock()
	defer frontStore.RUnlock()
	items := make([]map[string]interface{}, 0)
	for _, order := range frontStore.orders {
		if order["ownerId"] == userID.String() {
			items = append(items, order)
		}
	}
	return frontJSON(c, items)
}
func FrontendCreateOrder(c buffalo.Context) error {
	userID, _ := c.Value("user_id").(uuid.UUID)
	var farmer models.User
	if err := models.DB.Find(&farmer, userID); err != nil || strings.ToUpper(farmer.Role) != "FARMER" {
		return c.Render(http.StatusForbidden, r.JSON(map[string]string{"error": "only farmers may create orders"}))
	}
	var q struct {
		ListingID string `json:"listingId"`
		Quantity  int    `json:"quantity"`
	}
	_ = c.Bind(&q)
	if q.Quantity < 1 {
		q.Quantity = 1
	}
	frontStore.RLock()
	var listing map[string]interface{}
	for _, candidate := range frontStore.listings {
		if candidate["id"] == q.ListingID {
			listing = candidate
			break
		}
	}
	frontStore.RUnlock()
	if listing == nil {
		return c.Render(http.StatusNotFound, r.JSON(map[string]string{"error": "listing not found"}))
	}
	if listing["ownerId"] == userID.String() {
		return c.Render(http.StatusForbidden, r.JSON(map[string]string{"error": "cannot order your own listing"}))
	}
	x := map[string]interface{}{"id": fmt.Sprintf("order_%d", time.Now().UnixNano()), "ownerId": userID.String(), "listingId": q.ListingID, "quantity": q.Quantity, "amountSats": listing["priceSats"], "bolt11": listing["bolt11"], "status": "pending_payment", "createdAt": time.Now().UTC().Format(time.RFC3339)}
	frontStore.Lock()
	frontStore.orders = append(frontStore.orders, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}
func FrontendCreateInvoice(c buffalo.Context) error {
	var q map[string]interface{}
	_ = c.Bind(&q)
	id := fmt.Sprintf("inv_%d", time.Now().UnixNano())
	x := map[string]interface{}{"id": id, "orderId": q["orderId"], "amountSats": q["amountSats"], "bolt11": "lnbc1placeholder", "status": "pending"}
	frontStore.Lock()
	frontStore.invoices = append(frontStore.invoices, x)
	frontStore.Unlock()
	return c.Render(http.StatusCreated, r.JSON(x))
}
func FrontendInvoiceStatus(c buffalo.Context) error {
	id := c.Param("id")
	frontStore.RLock()
	defer frontStore.RUnlock()
	for _, x := range frontStore.invoices {
		if x["id"] == id {
			return frontJSON(c, x)
		}
	}
	return c.Render(404, r.JSON(map[string]string{"error": "invoice not found"}))
}
func FrontendWallet(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "authentication required"}))
	}

	// LEDGER MAPPING POINT: users.balance_sats is the current authoritative
	// materialized balance. Replace this query with your ledger repository's
	// BalanceForUser implementation if balances are derived from ledger entries.
	var balance struct {
		BalanceSats int64 `db:"balance_sats"`
	}
	if err := models.DB.RawQuery(
		"SELECT balance_sats FROM users WHERE id = ?", userID,
	).First(&balance); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to read wallet balance"}))
	}

	return frontJSON(c, map[string]interface{}{"balanceSats": balance.BalanceSats})
}
func FrontendWalletTransactions(c buffalo.Context) error {
	userID, _ := c.Value("user_id").(uuid.UUID)
	frontStore.RLock()
	defer frontStore.RUnlock()
	items := make([]map[string]interface{}, 0)
	for _, order := range frontStore.orders {
		if order["ownerId"] == userID.String() {
			items = append(items, map[string]interface{}{"id": order["id"], "type": "order_payment", "amountSats": order["amountSats"], "status": order["status"], "createdAt": order["createdAt"]})
		}
	}
	return frontJSON(c, items)
}
func FrontendWalletDeposit(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "authentication required"}))
	}
	var req struct {
		AmountSats int64 `json:"amountSats"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil || req.AmountSats < 1 {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": "a positive amountSats is required"}))
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 20*time.Second)
	defer cancel()
	invoice, err := createBlinkDepositInvoice(ctx, req.AmountSats)
	if err != nil {
		code := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			code = http.StatusGatewayTimeout
		}
		return c.Render(code, r.JSON(map[string]string{"error": "unable to create Blink deposit invoice"}))
	}

	expiresAt := time.Now().UTC().Add(time.Hour)
	var saved struct {
		ID uuid.UUID `db:"id"`
	}
	if err := models.DB.RawQuery(`
		INSERT INTO invoices (user_id, payment_request, payment_hash, payment_secret, amount_sats, memo, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'Marketplace Blink wallet deposit', 'pending', ?)
		RETURNING id`, userID, invoice.PaymentRequest, invoice.PaymentHash, invoice.PaymentSecret, req.AmountSats, expiresAt,
	).First(&saved); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to save deposit invoice"}))
	}

	return c.Render(http.StatusCreated, r.JSON(map[string]interface{}{
		"id": saved.ID, "bolt11": invoice.PaymentRequest, "amountSats": req.AmountSats,
		"expiresAt": expiresAt.Format(time.RFC3339), "status": "pending",
	}))
}
func FrontendWalletWithdraw(c buffalo.Context) error {
	userID, ok := c.Value("user_id").(uuid.UUID)
	if !ok {
		return c.Render(http.StatusUnauthorized, r.JSON(map[string]string{"error": "authentication required"}))
	}

	var req struct {
		LnAddress  string `json:"lnAddress"`
		AmountSats int64  `json:"amountSats"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": "invalid withdrawal payload"}))
	}
	req.LnAddress = strings.TrimSpace(req.LnAddress)
	if req.AmountSats < 1 || req.LnAddress == "" {
		return c.Render(http.StatusBadRequest, r.JSON(map[string]string{"error": "lnAddress and a positive amountSats are required"}))
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 20*time.Second)
	defer cancel()

	// This is an explicit, independent transaction rather than the request
	// middleware transaction: the row lock and debit must remain together until
	// Blink has definitively accepted or rejected the payment.
	tx, err := models.DB.NewTransactionContextOptions(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to start wallet transaction"}))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.TX.Rollback()
		}
	}()

	// LEDGER MAPPING POINT: this row lock is the authoritative balance read.
	// Map it to LedgerRepository.LockBalanceForUser(tx, userID) if applicable.
	var wallet struct {
		BalanceSats int64 `db:"balance_sats"`
	}
	if err := tx.RawQuery(
		"SELECT balance_sats FROM users WHERE id = ? FOR UPDATE",
		userID,
	).First(&wallet); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to lock wallet balance"}))
	}
	if wallet.BalanceSats < req.AmountSats {
		return c.Render(http.StatusConflict, r.JSON(map[string]interface{}{
			"error": "insufficient funds", "balanceSats": wallet.BalanceSats,
		}))
	}

	var pending struct {
		ID uuid.UUID `db:"id"`
	}
	// LEDGER MAPPING POINT: this creates the pending debit and reserves funds
	// before any external side effect. Map these statements to
	// LedgerRepository.CreatePendingDebit and ReserveBalance.
	if err := tx.RawQuery(`
		INSERT INTO wallet_transactions
			(user_id, transaction_type, direction, settlement_method, status, amount_sats, payee_msisdn, memo)
		VALUES (?, 'withdrawal', 'OUTBOUND', 'LIGHTNING', 'PAYOUT_PENDING', ?, ?, 'Marketplace wallet withdrawal')
		RETURNING id`, userID, req.AmountSats, req.LnAddress).First(&pending); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to create pending debit"}))
	}
	if err := tx.RawQuery(
		"UPDATE users SET balance_sats = balance_sats - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND balance_sats >= ?",
		req.AmountSats, userID, req.AmountSats,
	).Exec(); err != nil {
		return c.Render(http.StatusConflict, r.JSON(map[string]string{"error": "wallet balance changed; retry withdrawal"}))
	}

	status, providerErr := sendBlinkWithdrawal(ctx, req.LnAddress, req.AmountSats)
	if providerErr != nil || status != "SUCCESS" {
		code := http.StatusBadGateway
		message := "Blink payment failed"
		if errors.Is(providerErr, context.DeadlineExceeded) || errors.Is(providerErr, context.Canceled) {
			code, message = http.StatusGatewayTimeout, "Blink payment timed out"
		}
		return c.Render(code, r.JSON(map[string]string{"error": message}))
	}

	if err := tx.RawQuery(`
		UPDATE wallet_transactions
		SET status = 'SETTLED', settled_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'PAYOUT_PENDING'`, pending.ID).Exec(); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to settle wallet debit"}))
	}
	if err := tx.TX.Commit(); err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(map[string]string{"error": "unable to commit wallet debit"}))
	}
	committed = true

	return frontJSON(c, map[string]interface{}{
		"id": pending.ID, "type": "withdraw", "status": "SUCCESS", "amountSats": req.AmountSats,
	})
}

func sendBlinkWithdrawal(ctx context.Context, lnAddress string, amountSats int64) (string, error) {
	apiToken := strings.TrimSpace(os.Getenv("BLINK_API_KEY"))
	walletID := strings.TrimSpace(os.Getenv("BLINK_WALLET_ID"))
	if apiToken == "" || walletID == "" {
		return "", errors.New("Blink credentials are not configured")
	}
	endpoint := strings.TrimSpace(os.Getenv("BLINK_API_URL"))
	if endpoint == "" {
		endpoint = "https://api.blink.sv/graphql"
	}

	payload := map[string]interface{}{
		"query": `mutation LnAddressPaymentSend($input: LnAddressPaymentSendInput!) {
			lnAddressPaymentSend(input: $input) { status errors { message } }
		}`,
		"variables": map[string]interface{}{"input": map[string]interface{}{
			"walletId": walletID, "lnAddress": lnAddress, "amount": amountSats,
			"memo": "GreenTech marketplace withdrawal",
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-KEY", apiToken)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Blink returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Payment struct {
				Status string `json:"status"`
				Errors []struct {
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"lnAddressPaymentSend"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Errors) > 0 || len(result.Data.Payment.Errors) > 0 {
		return "", errors.New("Blink returned payment errors")
	}
	return result.Data.Payment.Status, nil
}

type blinkDepositInvoice struct {
	PaymentRequest string
	PaymentHash    string
	PaymentSecret  string
}

func createBlinkDepositInvoice(ctx context.Context, amountSats int64) (*blinkDepositInvoice, error) {
	apiToken := strings.TrimSpace(os.Getenv("BLINK_API_KEY"))
	walletID := strings.TrimSpace(os.Getenv("BLINK_WALLET_ID"))
	if apiToken == "" || walletID == "" {
		return nil, errors.New("Blink credentials are not configured")
	}
	endpoint := strings.TrimSpace(os.Getenv("BLINK_API_URL"))
	if endpoint == "" {
		endpoint = "https://api.blink.sv/graphql"
	}
	payload := map[string]interface{}{
		"query": `mutation LnInvoiceCreate($input: LnInvoiceCreateInput!) {
			lnInvoiceCreate(input: $input) {
				invoice { paymentRequest paymentHash paymentSecret satoshis }
				errors { message }
			}
		}`,
		"variables": map[string]interface{}{"input": map[string]interface{}{
			"walletId": walletID, "amount": amountSats, "memo": "GreenTech marketplace wallet deposit",
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-KEY", apiToken)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Blink returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Data struct {
			Create struct {
				Invoice struct {
					PaymentRequest string `json:"paymentRequest"`
					PaymentHash    string `json:"paymentHash"`
					PaymentSecret  string `json:"paymentSecret"`
				} `json:"invoice"`
				Errors []struct {
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"lnInvoiceCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Errors) > 0 || len(result.Data.Create.Errors) > 0 || result.Data.Create.Invoice.PaymentRequest == "" {
		return nil, errors.New("Blink returned invoice errors")
	}
	return &blinkDepositInvoice{
		PaymentRequest: result.Data.Create.Invoice.PaymentRequest,
		PaymentHash:    result.Data.Create.Invoice.PaymentHash,
		PaymentSecret:  result.Data.Create.Invoice.PaymentSecret,
	}, nil
}
func FrontendUpdateUser(c buffalo.Context) error {
	var x map[string]interface{}
	_ = c.Bind(&x)
	return frontJSON(c, x)
}
func FrontendForgotPassword(c buffalo.Context) error { return c.Render(http.StatusNoContent, nil) }
func FrontendResetPassword(c buffalo.Context) error  { return c.Render(http.StatusNoContent, nil) }
