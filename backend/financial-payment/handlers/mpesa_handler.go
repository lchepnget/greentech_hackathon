package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"financial-gateway/backend/financial-payment/models"
	"financial-gateway/backend/financial-payment/service"
)

// MpesaHandler manages HTTP webhooks and controllers for Safaricom Daraja
type MpesaHandler struct {
	svc *service.PaymentService
}

// NewMpesaHandler creates a new MpesaHandler instance
func NewMpesaHandler(svc *service.PaymentService) *MpesaHandler {
	return &MpesaHandler{svc: svc}
}

// HandleRegisterUrls registers C2B validation and confirmation URLs with Daraja
func (h *MpesaHandler) HandleRegisterUrls(w http.ResponseWriter, r *http.Request) {
	respBody, statusCode, err := h.svc.RegisterC2BUrls()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBody)
}

// HandleValidation is Safaricom's validation hook to approve/reject incoming funds
func (h *MpesaHandler) HandleValidation(w http.ResponseWriter, r *http.Request) {
	var payload models.C2BCallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ResultCode":1,"ResultDesc":"Rejected: Invalid payload format"}`))
		return
	}

	log.Printf("📥 Validation hook triggered for KES %s from phone %s (TransID: %s)\n",
		payload.TransAmount, payload.MSISDN, payload.TransID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
}

// HandleConfirmation is triggered once real funds clear into the Paybill/Till
func (h *MpesaHandler) HandleConfirmation(w http.ResponseWriter, r *http.Request) {
	var payload models.C2BCallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"Invalid payload"}`, http.StatusBadRequest)
		return
	}

	log.Printf("✅ Payment Confirmed! KES %s received from user %s. Mpesa Ref: %s\n",
		payload.TransAmount, payload.MSISDN, payload.TransID)

	// Process payment with idempotency & async conversion
	_, err := h.svc.ProcessC2BConfirmation(payload)
	if err != nil {
		log.Printf("❌ Failed to register C2B confirmation: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Acknowledged"}`))
}

// HandleTriggerB2C executes a programmatic B2C payout to a recipient phone
func (h *MpesaHandler) HandleTriggerB2C(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount   int    `json:"amount"`
		Phone    string `json:"phone"`
		Remarks  string `json:"remarks"`
		Occasion string `json:"occasion"`
	}

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.Amount <= 0 {
		req.Amount = 10
	}

	respBody, statusCode, err := h.svc.TriggerB2CPayout(req.Amount, req.Phone, req.Remarks, req.Occasion)
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(respBody)
}

// HandleB2CResult receives callback results for B2C payouts
func (h *MpesaHandler) HandleB2CResult(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("📩 B2C Result received: %s\n", string(body))

	var payload models.B2CResultPayload
	_ = json.Unmarshal(body, &payload)
	_ = h.svc.HandleB2CCallbackResult(payload)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Acknowledged"}`))
}

// HandleB2CTimeout receives timeout notifications for B2C payouts
func (h *MpesaHandler) HandleB2CTimeout(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("⚠️ B2C Timeout notification received: %s\n", string(body))

	h.svc.HandleB2CTimeoutNotification(body)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Acknowledged"}`))
}
