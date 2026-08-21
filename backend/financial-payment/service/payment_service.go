package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"financial-gateway/backend/financial-payment/infrastructure/lightning"
	"financial-gateway/backend/financial-payment/infrastructure/mpesa"
	"financial-gateway/backend/financial-payment/models"
	"financial-gateway/backend/financial-payment/repository"
)

// PaymentService coordinates business logic between M-Pesa, Lightning, persistence, and audit logging
type PaymentService struct {
	mpesaClient mpesa.MpesaProvider
	blinkClient lightning.LightningProvider
	repo        repository.PaymentRepository
}

// NewPaymentService creates a new PaymentService instance with interface dependencies
func NewPaymentService(
	mpesaClient mpesa.MpesaProvider,
	blinkClient lightning.LightningProvider,
	repo repository.PaymentRepository,
) *PaymentService {
	return &PaymentService{
		mpesaClient: mpesaClient,
		blinkClient: blinkClient,
		repo:        repo,
	}
}

// parseDarajaTime converts Daraja's local Nairobi timestamp (YYYYMMDDHHMMSS) to UTC
func parseDarajaTime(transTime string) time.Time {
	if len(transTime) == 14 {
		loc, err := time.LoadLocation("Africa/Nairobi")
		if err == nil {
			t, err := time.ParseInLocation("20060102150405", transTime, loc)
			if err == nil {
				return t.UTC()
			}
		}
	}
	return time.Now().UTC()
}

// ProcessC2BConfirmation processes incoming M-Pesa payments with idempotency guards and async Sats conversion
func (s *PaymentService) ProcessC2BConfirmation(payload models.C2BCallbackPayload) (*models.Transaction, error) {
	// Check for idempotency: if this M-Pesa TransID has already been received, do not duplicate
	if payload.TransID != "" {
		existing, err := s.repo.GetByMpesaTransID(payload.TransID)
		if err == nil && existing != nil {
			log.Printf("⚠️ Idempotent C2B callback received for TransID: %s. Ignoring duplicate.", payload.TransID)
			_ = s.repo.RecordAuditEvent(&models.AuditEvent{
				TransactionID: existing.ID,
				Action:        models.ActionDuplicateIgnored,
				Actor:         "daraja_webhook",
				ProviderRef:   payload.TransID,
				Details:       fmt.Sprintf("Duplicate C2B callback received for TransID %s", payload.TransID),
				Timestamp:     time.Now().UTC(),
			})
			return existing, nil
		}
	}

	// Safaricom TransAmount is authoritative
	amt, err := strconv.ParseFloat(payload.TransAmount, 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid or non-positive authoritative transaction amount: %s", payload.TransAmount)
	}

	createdTime := parseDarajaTime(payload.TransTime)
	txID := fmt.Sprintf("tx_c2b_%d", time.Now().UnixNano())

	tx := &models.Transaction{
		ID:            txID,
		MpesaTransID:  payload.TransID,
		MpesaType:     "C2B",
		PayerMSISDN:   payload.MSISDN,
		FiatAmountKES: amt,
		Direction:     models.DirectionInbound,
		Status:        models.StatusReceived,
		Memo:          fmt.Sprintf("M-Pesa payment from %s (Ref: %s)", payload.MSISDN, payload.BillRefNumber),
		CreatedAt:     createdTime,
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.repo.Save(tx); err != nil {
		log.Printf("❌ Failed to persist initial C2B transaction: %v", err)
		return nil, err
	}

	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionC2BConfirmed,
		Actor:         "daraja_webhook",
		ProviderRef:   payload.TransID,
		Details:       fmt.Sprintf("Captured KES %.2f from %s (TransID: %s)", amt, payload.MSISDN, payload.TransID),
		Timestamp:     time.Now().UTC(),
	})

	// Asynchronously execute real-time KES -> Sats conversion pipeline
	go func(targetTx *models.Transaction) {
		s.executeConversionPipeline(targetTx)
	}(tx)

	return tx, nil
}

// executeConversionPipeline executes the fiat-to-satoshis calculation and state progression
func (s *PaymentService) executeConversionPipeline(tx *models.Transaction) {
	_ = s.repo.UpdateStatus(tx.ID, models.StatusConverting, nil)
	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionConversionStarted,
		Actor:         "system",
		Details:       fmt.Sprintf("Querying Blink live exchange rate for KES %.2f", tx.FiatAmountKES),
		Timestamp:     time.Now().UTC(),
	})

	sats, pricePerSat, err := s.blinkClient.ConvertFiatToSats(tx.FiatAmountKES, "KES")
	if err != nil {
		log.Printf("❌ Conversion failed for transaction %s: %v", tx.ID, err)
		_ = s.repo.UpdateStatus(tx.ID, models.StatusFailed, nil)
		_ = s.repo.RecordAuditEvent(&models.AuditEvent{
			TransactionID: tx.ID,
			Action:        models.ActionConversionFailed,
			Actor:         "system",
			Details:       fmt.Sprintf("Blink rate unavailable: %v", err),
			Timestamp:     time.Now().UTC(),
		})
		return
	}

	intSats := int64(sats)
	now := time.Now().UTC()

	tx.AmountSats = intSats
	tx.ExchangeRate = pricePerSat
	tx.ExchangeRateSource = "blink.realtimePrice"
	tx.ExchangeRateTimestamp = &now
	tx.Status = models.StatusSettled
	tx.SettledAt = &now
	tx.UpdatedAt = now

	if err := s.repo.Save(tx); err != nil {
		log.Printf("⚠️ Warning: Failed to save settled conversion state for %s: %v", tx.ID, err)
		return
	}

	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionConversionCompleted,
		Actor:         "system",
		Details:       fmt.Sprintf("Settled KES %.2f to %d Sats (Rate: %.6f KES/Sat)", tx.FiatAmountKES, intSats, pricePerSat),
		Timestamp:     now,
	})

	log.Printf("📊 Pipeline Conversion Complete: Tx %s | KES %.2f -> %d Sats (Rate: %.6f KES/Sat)",
		tx.ID, tx.FiatAmountKES, intSats, pricePerSat)
}

// ConvertKESAmountToSats calculates real-time Sats value for a given KES amount without saving
func (s *PaymentService) ConvertKESAmountToSats(kesAmount float64) (float64, float64, error) {
	if kesAmount <= 0 {
		return 0, 0, errors.New("fiat amount must be strictly positive")
	}
	return s.blinkClient.ConvertFiatToSats(kesAmount, "KES")
}

// CreateLightningInvoice generates a BOLT11 invoice with explicit expiry and server-side conversion
func (s *PaymentService) CreateLightningInvoice(req models.CreateInvoiceReq) (*models.LnInvoice, *models.Transaction, error) {
	targetSats := req.Amount
	var pricePerSat float64
	var exchangeTimestamp *time.Time

	if req.KesAmount > 0 {
		sats, rate, err := s.blinkClient.ConvertFiatToSats(req.KesAmount, "KES")
		if err != nil {
			return nil, nil, fmt.Errorf("conversion rate query failed: %w", err)
		}
		targetSats = int64(sats)
		pricePerSat = rate
		t := time.Now().UTC()
		exchangeTimestamp = &t
	}

	if targetSats < 1 {
		targetSats = 1
	}

	memo := req.Memo
	if memo == "" {
		memo = fmt.Sprintf("GreenTech BSF Payment (%d Sats)", targetSats)
	}

	invoice, err := s.blinkClient.CreateInvoice(targetSats, memo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create lightning invoice via Blink: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(1 * time.Hour) // 1 Hour BOLT11 explicit validity

	tx := &models.Transaction{
		ID:                    fmt.Sprintf("tx_ln_%d", now.UnixNano()),
		PaymentHash:           invoice.PaymentHash,
		FiatAmountKES:         req.KesAmount,
		AmountSats:            targetSats,
		ExchangeRate:          pricePerSat,
		ExchangeRateSource:    "blink.realtimePrice",
		ExchangeRateTimestamp: exchangeTimestamp,
		Direction:             models.DirectionInbound,
		SettlementMethod:      models.SettlementLightning,
		Status:                models.StatusPayoutPending,
		Memo:                  memo,
		CreatedAt:             now,
		ExpiresAt:             &expiresAt,
		UpdatedAt:             now,
	}

	_ = s.repo.Save(tx)
	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionInvoiceCreated,
		Actor:         "system",
		ProviderRef:   invoice.PaymentHash,
		Details:       fmt.Sprintf("Created invoice for %d Sats, expires at %s", targetSats, expiresAt.Format(time.RFC3339)),
		Timestamp:     now,
	})

	return invoice, tx, nil
}

// PayLightningAddress executes and verifies a payout to a Lightning Address
func (s *PaymentService) PayLightningAddress(req models.PayAddressReq) (string, *models.Transaction, error) {
	if req.LnAddress == "" {
		return "", nil, errors.New("lightning address is required")
	}
	if req.Amount < 1 {
		return "", nil, errors.New("amount must be at least 1 satoshi")
	}

	now := time.Now().UTC()
	txID := fmt.Sprintf("tx_payout_%d", now.UnixNano())

	status, err := s.blinkClient.PayToAddress(req.LnAddress, req.Amount, req.Memo)
	if err != nil {
		failTx := &models.Transaction{
			ID:               txID,
			PayeeMSISDN:      req.LnAddress,
			AmountSats:       req.Amount,
			Direction:        models.DirectionOutbound,
			SettlementMethod: models.SettlementLightning,
			Status:           models.StatusFailed,
			Memo:             req.Memo,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		_ = s.repo.Save(failTx)
		_ = s.repo.RecordAuditEvent(&models.AuditEvent{
			TransactionID: txID,
			Action:        models.ActionLightningPayoutFailed,
			Actor:         "system",
			Details:       fmt.Sprintf("Payout failed: %v", err),
			Timestamp:     now,
		})
		return "", failTx, err
	}

	settledStatus := models.StatusSettled
	var settledAt *time.Time
	if status == "SUCCESS" || status == "PAID" || status == "ALREADY_PAID" {
		t := time.Now().UTC()
		settledAt = &t
	} else {
		settledStatus = models.StatusPayoutPending
	}

	tx := &models.Transaction{
		ID:               txID,
		PayeeMSISDN:      req.LnAddress,
		AmountSats:       req.Amount,
		Direction:        models.DirectionOutbound,
		SettlementMethod: models.SettlementLightning,
		Status:           settledStatus,
		Memo:             req.Memo,
		CreatedAt:        now,
		SettledAt:        settledAt,
		UpdatedAt:        now,
	}
	_ = s.repo.Save(tx)

	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionLightningPayoutSent,
		Actor:         "system",
		Details:       fmt.Sprintf("Dispatched %d Sats to %s (Status: %s)", req.Amount, req.LnAddress, status),
		Timestamp:     now,
	})

	return status, tx, nil
}

// TriggerB2CPayout executes an M-Pesa B2C payout to a recipient phone
func (s *PaymentService) TriggerB2CPayout(amount int, recipientPhone, remarks, occasion string) ([]byte, int, error) {
	if amount <= 0 {
		return nil, 400, errors.New("disbursement amount must be strictly positive")
	}

	now := time.Now().UTC()
	txID := fmt.Sprintf("tx_b2c_%d", now.UnixNano())

	respBody, statusCode, err := s.mpesaClient.TriggerB2CPayout(amount, recipientPhone, remarks, occasion)
	if err != nil {
		return respBody, statusCode, err
	}

	tx := &models.Transaction{
		ID:               txID,
		PayeeMSISDN:      recipientPhone,
		FiatAmountKES:    float64(amount),
		Direction:        models.DirectionOutbound,
		SettlementMethod: models.SettlementMpesaB2C,
		Status:           models.StatusPayoutPending,
		Memo:             remarks,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_ = s.repo.Save(tx)

	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: tx.ID,
		Action:        models.ActionB2CTriggered,
		Actor:         "system",
		Details:       fmt.Sprintf("B2C payout triggered for KES %d to %s", amount, recipientPhone),
		Timestamp:     now,
	})

	return respBody, statusCode, nil
}

// HandleB2CCallbackResult processes the asynchronous B2C result callback from Daraja
func (s *PaymentService) HandleB2CCallbackResult(payload models.B2CResultPayload) error {
	res := payload.Result
	now := time.Now().UTC()

	var targetTx *models.Transaction
	var err error

	if res.OriginatorConversationID != "" {
		targetTx, err = s.repo.GetByMpesaTransID(res.OriginatorConversationID)
	}
	if targetTx == nil && res.ConversationID != "" {
		targetTx, err = s.repo.GetByMpesaTransID(res.ConversationID)
	}

	if targetTx != nil {
		if res.ResultCode == 0 {
			_ = s.repo.UpdateStatus(targetTx.ID, models.StatusSettled, &now)
			_ = s.repo.RecordAuditEvent(&models.AuditEvent{
				TransactionID: targetTx.ID,
				Action:        models.ActionB2CResultReceived,
				Actor:         "daraja_webhook",
				ProviderRef:   res.TransactionID,
				Details:       fmt.Sprintf("B2C Succeeded: %s", res.ResultDesc),
				Timestamp:     now,
			})
		} else {
			_ = s.repo.UpdateStatus(targetTx.ID, models.StatusFailed, nil)
			_ = s.repo.RecordAuditEvent(&models.AuditEvent{
				TransactionID: targetTx.ID,
				Action:        models.ActionB2CResultReceived,
				Actor:         "daraja_webhook",
				ProviderRef:   res.TransactionID,
				Details:       fmt.Sprintf("B2C Failed (Code %d): %s", res.ResultCode, res.ResultDesc),
				Timestamp:     now,
			})
		}
	}

	return err
}

// HandleB2CTimeoutNotification processes the asynchronous B2C timeout notification
func (s *PaymentService) HandleB2CTimeoutNotification(rawPayload []byte) {
	now := time.Now().UTC()
	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		Action:    models.ActionB2CTimeout,
		Actor:     "daraja_webhook",
		Details:   fmt.Sprintf("B2C Timeout reported: %s", string(rawPayload)),
		Timestamp: now,
	})
}

// IssueRefund executes an authorized M-Pesa reversal linked to an original settled transaction
func (s *PaymentService) IssueRefund(origTxID string, reason string) (*models.Transaction, error) {
	origTx, err := s.repo.GetByID(origTxID)
	if err != nil {
		return nil, fmt.Errorf("original transaction not found: %w", err)
	}

	if origTx.Status != models.StatusSettled {
		return nil, errors.New("cannot refund non-settled transaction")
	}
	if origTx.PayerMSISDN == "" || origTx.FiatAmountKES <= 0 {
		return nil, errors.New("missing original payer phone or invalid refund amount")
	}

	now := time.Now().UTC()
	refundID := fmt.Sprintf("tx_refund_%d", now.UnixNano())

	// Execute reversal via Daraja B2C
	_, _, err = s.mpesaClient.TriggerB2CPayout(int(origTx.FiatAmountKES), origTx.PayerMSISDN, reason, "Refund")
	if err != nil {
		return nil, fmt.Errorf("reversal dispatch failed: %w", err)
	}

	refundTx := &models.Transaction{
		ID:               refundID,
		PayerMSISDN:      origTx.PayerMSISDN,
		FiatAmountKES:    origTx.FiatAmountKES,
		Direction:        models.DirectionOutbound,
		SettlementMethod: models.SettlementMpesaB2C,
		Status:           models.StatusPayoutPending,
		Memo:             fmt.Sprintf("Refund for %s: %s", origTxID, reason),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	_ = s.repo.Save(refundTx)

	_ = s.repo.RecordAuditEvent(&models.AuditEvent{
		TransactionID: refundID,
		Action:        models.ActionRefundIssued,
		Actor:         "admin",
		ProviderRef:   origTx.MpesaTransID,
		Details:       fmt.Sprintf("Refund of KES %.2f issued for Tx %s (Reason: %s)", origTx.FiatAmountKES, origTxID, reason),
		Timestamp:     now,
	})

	return refundTx, nil
}

// RegisterC2BUrls registers callback webhooks with Daraja
func (s *PaymentService) RegisterC2BUrls() ([]byte, int, error) {
	return s.mpesaClient.RegisterC2BUrls()
}

// GetWalletBalances returns the connected Blink wallet profiles and balances
func (s *PaymentService) GetWalletBalances() (*models.BlinkWalletMe, error) {
	return s.blinkClient.GetWalletInfo()
}

// ListPayments returns recorded transactions with optional filtering
func (s *PaymentService) ListPayments(status, direction string) ([]*models.Transaction, error) {
	return s.repo.List(status, direction)
}

// GetTransactionByID retrieves single transaction detail
func (s *PaymentService) GetTransactionByID(id string) (*models.Transaction, error) {
	return s.repo.GetByID(id)
}

// GetReconciliationReport generates automated reconciliation metrics
func (s *PaymentService) GetReconciliationReport() (*models.ReconciliationReport, error) {
	return s.repo.GenerateReconciliationReport()
}

// ListAuditLogs returns all audit trail entries
func (s *PaymentService) ListAuditLogs(txID string) ([]*models.AuditEvent, error) {
	return s.repo.ListAuditEvents(txID)
}

// SweepExpiredInvoices sweeps pending invoices past expiry
func (s *PaymentService) SweepExpiredInvoices() (int, error) {
	return s.repo.SweepExpiredInvoices()
}

// StartBackgroundSweeper starts an in-process ticker that regularly sweeps expired invoices
func (s *PaymentService) StartBackgroundSweeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				count, _ := s.SweepExpiredInvoices()
				if count > 0 {
					log.Printf("🧹 Swept %d expired Lightning invoices.", count)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
