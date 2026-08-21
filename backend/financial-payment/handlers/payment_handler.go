package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"financial-gateway/backend/financial-payment/models"
	"financial-gateway/backend/financial-payment/service"
)

// In-memory mock store for marketplace listings, orders and demo user
type mockStore struct {
	mu       sync.RWMutex
	listings []map[string]interface{}
	orders   []map[string]interface{}
}

var store = &mockStore{
	listings: []map[string]interface{}{
		{
			"id":           "bsf-001",
			"title":        "High-Grade BSF Larvae Protein Feed (50kg)",
			"producerName": "GreenTech Bio-Refinery",
			"wasteType":    "BSF Protein Animal Feed",
			"quantity":     25,
			"unit":         "bags",
			"priceSats":    15000,
			"location":     "Nairobi Industrial Area",
			"description":  "Sustainable, high-protein insect meal converted from hotel organic food waste. Ideal for poultry and aquaculture.",
		},
		{
			"id":           "bsf-002",
			"title":        "Organic BSF Frass Fertilizer (25kg)",
			"producerName": "EcoLoop AgriSolutions",
			"wasteType":    "Organic Bio-Fertilizer",
			"quantity":     40,
			"unit":         "bags",
			"priceSats":    8500,
			"location":     "Naivasha Farming Hub",
			"description":  "Nutrient-dense organic soil conditioner and natural bio-fertilizer produced during BSF larvae bioconversion.",
		},
		{
			"id":           "bsf-003",
			"title":        "Hotel Food Waste Feedstock (100kg)",
			"producerName": "Sarova Hotel Green Team",
			"wasteType":    "Raw Organic Kitchen Waste",
			"quantity":     10,
			"unit":         "crates",
			"priceSats":    3200,
			"location":     "Nairobi CBD",
			"description":  "Pre-sorted, non-hazardous organic hotel food scraps ready for insect farm substrate processing.",
		},
	},
	orders: []map[string]interface{}{},
}

// PaymentHandler handles general payment queries, auditing, reconciliation, and health status
type PaymentHandler struct {
	svc *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler instance
func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// HandleRoot provides a landing summary when navigating to the root URL
func (h *PaymentHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/api" && r.URL.Path != "/api/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"system":      "GreenTech Financial Gateway",
		"status":      "online",
		"version":     "1.0.0",
		"description": "Unified fiat-to-Lightning payment router bridging Safaricom M-Pesa Daraja and Bitcoin Lightning Network",
		"endpoints": map[string]string{
			"health":               "GET /api/health",
			"payments_ledger":      "GET /api/payments",
			"admin_transactions":   "GET /api/admin/transactions",
			"admin_reconciliation": "GET /api/admin/reconciliation",
			"admin_audit_logs":     "GET /api/admin/audit-logs",
			"admin_refund":         "POST /api/admin/refund",
			"wallet_balance":       "GET /api/wallet",
			"wallet_txs":           "GET /api/wallet/transactions",
			"wallet_deposit":       "POST /api/wallet/deposit",
			"wallet_withdraw":      "POST /api/wallet/withdraw",
			"blink_wallet":         "GET /api/blink/wallet",
			"blink_price":          "GET /api/blink/price?amount=500",
			"lightning_invoice":    "POST /api/lightning/create-invoice",
			"lightning_payout":     "POST /api/lightning/pay-address",
			"mpesa_c2b_confirm":    "POST /api/mpesa/c2b-confirmation",
			"mpesa_b2c_payout":     "POST /api/mpesa/trigger-b2c",
			"marketplace":          "GET /api/listings",
			"orders":               "GET /api/orders",
		},
	})
}

// HandleHealth returns the system health status
func (h *PaymentHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"online","service":"Financial Gateway - GreenTech Engine"}`))
}

// HandleListPayments retrieves recorded transaction history across fiat and lightning with filters
func (h *PaymentHandler) HandleListPayments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	direction := r.URL.Query().Get("direction")

	payments, err := h.svc.ListPayments(status, direction)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    len(payments),
		"payments": payments,
	})
}

// HandleAdminTransactions lists all transactions for administrative monitoring
func (h *PaymentHandler) HandleAdminTransactions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	direction := r.URL.Query().Get("direction")

	txs, err := h.svc.ListPayments(status, direction)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":        len(txs),
		"transactions": txs,
	})
}

// HandleAdminTransactionByID retrieves details of a single transaction
func (h *PaymentHandler) HandleAdminTransactionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/transactions/")
	if id == "" {
		http.Error(w, `{"error":"transaction id is required"}`, http.StatusBadRequest)
		return
	}

	tx, err := h.svc.GetTransactionByID(id)
	if err != nil {
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	audits, _ := h.svc.ListAuditLogs(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction": tx,
		"auditEvents": audits,
	})
}

// HandleAdminReconciliation generates automated dual-rail audit reconciliation reports
func (h *PaymentHandler) HandleAdminReconciliation(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.GetReconciliationReport()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// HandleAdminAuditLogs returns the global or transaction-specific audit trail
func (h *PaymentHandler) HandleAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	txID := r.URL.Query().Get("transactionId")
	logs, err := h.svc.ListAuditLogs(txID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(logs),
		"logs":  logs,
	})
}

// HandleAdminRefund executes an authorized M-Pesa reversal linked to an original settled transaction
func (h *PaymentHandler) HandleAdminRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed. Use POST."}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OriginalTransactionID string `json:"originalTransactionId"`
		Reason                string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OriginalTransactionID == "" {
		http.Error(w, `{"error":"originalTransactionId is required"}`, http.StatusBadRequest)
		return
	}

	refundTx, err := h.svc.IssueRefund(req.OriginalTransactionID, req.Reason)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "refund_initiated",
		"refund": refundTx,
	})
}

// HandleWallet returns current balance in Sats and fiat KES for the frontend
func (h *PaymentHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	balanceSats := 100000.0 // Default demo wallet pool
	var fiatKes float64

	walletMe, err := h.svc.GetWalletBalances()
	if err == nil && walletMe != nil {
		for _, w := range walletMe.DefaultAccount.Wallets {
			if w.WalletCurrency == "BTC" && w.Balance > 0 {
				balanceSats = w.Balance
			}
		}
	}

	// Calculate fiat conversion
	_, pricePerSat, err := h.svc.ConvertKESAmountToSats(100)
	if err == nil && pricePerSat > 0 {
		fiatKes = balanceSats * pricePerSat
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"balanceSats": int64(balanceSats),
		"fiatKes":     fiatKes,
	})
}

// HandleWalletTransactions returns recent transactions formatted for the frontend
func (h *PaymentHandler) HandleWalletTransactions(w http.ResponseWriter, r *http.Request) {
	payments, err := h.svc.ListPayments("", "")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	type FrontTx struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		AmountSats   int64  `json:"amountSats"`
		Status       string `json:"status"`
		CreatedAt    string `json:"createdAt"`
		Counterparty string `json:"counterparty,omitempty"`
	}

	txs := make([]FrontTx, 0, len(payments))
	for _, p := range payments {
		txType := "deposit"
		if strings.Contains(p.Direction, "OUTBOUND") || strings.Contains(p.SettlementMethod, "B2C") {
			txType = "withdraw"
		} else if strings.Contains(p.Memo, "Order") || strings.Contains(p.Memo, "BSF") {
			txType = "order_payment"
		}

		sats := p.AmountSats
		if sats <= 0 && p.SatsEquivalent > 0 {
			sats = int64(p.SatsEquivalent)
		}

		txs = append(txs, FrontTx{
			ID:           p.ID,
			Type:         txType,
			AmountSats:   sats,
			Status:       p.Status,
			CreatedAt:    p.CreatedAt.Format(time.RFC3339),
			Counterparty: p.PayerMSISDN,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

// HandleWalletDeposit creates a deposit invoice
func (h *PaymentHandler) HandleWalletDeposit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AmountSats int64 `json:"amountSats"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.AmountSats < 1 {
		req.AmountSats = 1
	}

	inv, tx, err := h.svc.CreateLightningInvoice(models.CreateInvoiceReq{
		Amount: req.AmountSats,
		Memo:   fmt.Sprintf("GreenTech Wallet Deposit (%d Sats)", req.AmountSats),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         inv.PaymentHash,
		"bolt11":     inv.PaymentRequest,
		"amountSats": req.AmountSats,
		"expiresAt":  tx.ExpiresAt.Format(time.RFC3339),
		"status":     "pending",
	})
}

// HandleWalletWithdraw processes withdrawal via Lightning or M-Pesa
func (h *PaymentHandler) HandleWalletWithdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Bolt11    string `json:"bolt11"`
		LnAddress string `json:"lnAddress"`
		Amount    int64  `json:"amountSats"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	targetAddress := req.LnAddress
	if targetAddress == "" {
		targetAddress = "greentech@blink.sv"
	}
	if req.Amount < 1 {
		req.Amount = 1
	}

	status, p, err := h.svc.PayLightningAddress(models.PayAddressReq{
		LnAddress: targetAddress,
		Amount:    req.Amount,
		Memo:      "GreenTech Wallet Withdrawal",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         p.ID,
		"type":       "withdraw",
		"amountSats": req.Amount,
		"status":     status,
		"createdAt":  time.Now().Format(time.RFC3339),
	})
}

// HandleListings handles GET /api/listings and POST /api/listings
func (h *PaymentHandler) HandleListings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var item map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, `{"error":"Invalid payload"}`, http.StatusBadRequest)
			return
		}
		item["id"] = fmt.Sprintf("bsf-%d", time.Now().Unix())
		store.mu.Lock()
		store.listings = append(store.listings, item)
		store.mu.Unlock()
		json.NewEncoder(w).Encode(item)
		return
	}

	store.mu.RLock()
	items := store.listings
	store.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
		"page":  1,
		"pages": 1,
		"total": len(items),
	})
}

// HandleListingByID handles GET /api/listings/{id}
func (h *PaymentHandler) HandleListingByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/listings/")
	store.mu.RLock()
	defer store.mu.RUnlock()

	for _, item := range store.listings {
		if item["id"] == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(item)
			return
		}
	}

	http.NotFound(w, r)
}

// HandleOrders handles GET /api/orders and POST /api/orders
func (h *PaymentHandler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var req struct {
			ListingID string `json:"listingId"`
			Quantity  int    `json:"quantity"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		order := map[string]interface{}{
			"id":         fmt.Sprintf("ord_%d", time.Now().Unix()),
			"listingId":  req.ListingID,
			"quantity":   req.Quantity,
			"amountSats": req.Quantity * 5000,
			"status":     "confirmed",
			"createdAt":  time.Now().Format(time.RFC3339),
		}

		store.mu.Lock()
		store.orders = append(store.orders, order)
		store.mu.Unlock()

		json.NewEncoder(w).Encode(order)
		return
	}

	store.mu.RLock()
	orders := store.orders
	store.mu.RUnlock()

	json.NewEncoder(w).Encode(orders)
}

// HandleAuth handles /api/auth/login, /api/auth/register, and /api/auth/me
func (h *PaymentHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := map[string]interface{}{
		"id":           "usr_greentech_01",
		"name":         "Kimutai (GreenTech)",
		"email":        "kimutai@greentech.org",
		"role":         "farmer",
		"location":     "Nairobi, Kenya",
		"businessName": "GreenTech Bio-Conversions Ltd",
	}

	if strings.HasSuffix(r.URL.Path, "/me") {
		json.NewEncoder(w).Encode(user)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}
