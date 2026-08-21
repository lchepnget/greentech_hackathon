package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"financial-gateway/backend/financial-payment/models"
	"financial-gateway/backend/financial-payment/service"
)

// LightningHandler handles HTTP endpoints for Bitcoin Lightning operations
type LightningHandler struct {
	svc *service.PaymentService
}

// NewLightningHandler creates a new LightningHandler instance
func NewLightningHandler(svc *service.PaymentService) *LightningHandler {
	return &LightningHandler{svc: svc}
}

// HandleWalletQuery returns the authenticated Blink user info and wallet balances
func (h *LightningHandler) HandleWalletQuery(w http.ResponseWriter, r *http.Request) {
	walletMe, err := h.svc.GetWalletBalances()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(walletMe)
}

// HandlePriceQuery calculates real-time KES to Satoshis conversion
func (h *LightningHandler) HandlePriceQuery(w http.ResponseWriter, r *http.Request) {
	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		amountStr = "100"
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid amount parameter"}`, http.StatusBadRequest)
		return
	}

	sats, pricePerSat, err := h.svc.ConvertKESAmountToSats(amount)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"kesAmount":   amount,
		"satoshis":    sats,
		"kesPerSat":   pricePerSat,
		"denominator": "KES",
		"timestamp":   time.Now().Unix(),
	})
}

// HandleCreateInvoice creates a BOLT11 invoice denominated in Sats or auto-converted from KES
func (h *LightningHandler) HandleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed. Use POST."}`, http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateInvoiceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	invoice, _, err := h.svc.CreateLightningInvoice(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoice)
}

// HandlePayAddress dispatches Satoshis to a Lightning Address
func (h *LightningHandler) HandlePayAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed. Use POST."}`, http.StatusMethodNotAllowed)
		return
	}

	var req models.PayAddressReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.LnAddress == "" || req.Amount <= 0 {
		http.Error(w, `{"error":"lnAddress and positive amount in Sats are required"}`, http.StatusBadRequest)
		return
	}

	status, _, err := h.svc.PayLightningAddress(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"lnAddress": req.LnAddress,
		"satoshis":  req.Amount,
	})
}
