package tests

import (
	"errors"
	"sync"
	"testing"
	"time"

	"financial-gateway/backend/financial-payment/models"
	"financial-gateway/backend/financial-payment/repository"
	"financial-gateway/backend/financial-payment/service"
)

// MockMpesaProvider implements mpesa.MpesaProvider for testing
type MockMpesaProvider struct {
	mu           sync.Mutex
	Token        string
	B2CCalls     int
	RegisterUrlsCalls int
	TriggerErr   error
}

func (m *MockMpesaProvider) GetToken() (string, error) {
	return "mock_token_123", nil
}

func (m *MockMpesaProvider) RegisterC2BUrls() ([]byte, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RegisterUrlsCalls++
	return []byte(`{"OriginatorConversationID":"123","ResponseCode":"0","ResponseDescription":"Success"}`), 200, nil
}

func (m *MockMpesaProvider) TriggerB2CPayout(amount int, recipientPhone, remarks, occasion string) ([]byte, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.B2CCalls++
	if m.TriggerErr != nil {
		return nil, 500, m.TriggerErr
	}
	return []byte(`{"ConversationID":"b2c_conv_999","ResponseCode":"0"}`), 200, nil
}

// MockLightningProvider implements lightning.LightningProvider for testing
type MockLightningProvider struct {
	mu           sync.Mutex
	PricePerSat  float64
	PriceErr     error
	InvoiceErr   error
	PayoutStatus string
	PayoutErr    error
}

func (m *MockLightningProvider) GetWalletInfo() (*models.BlinkWalletMe, error) {
	me := &models.BlinkWalletMe{
		ID: "user_mock",
	}
	me.DefaultAccount.Wallets = []struct {
		ID             string  `json:"id"`
		WalletCurrency string  `json:"walletCurrency"`
		Balance        float64 `json:"balance"`
	}{
		{ID: "btc_wallet_1", WalletCurrency: "BTC", Balance: 50000},
	}
	return me, nil
}

func (m *MockLightningProvider) ConvertFiatToSats(amount float64, currency string) (float64, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PriceErr != nil {
		return 0, 0, m.PriceErr
	}
	rate := m.PricePerSat
	if rate <= 0 {
		rate = 0.10 // 1 Sat = 0.10 KES (i.e. 10 Sats per KES)
	}
	sats := amount / rate
	return sats, rate, nil
}

func (m *MockLightningProvider) CreateInvoice(amountSats int64, memo string) (*models.LnInvoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.InvoiceErr != nil {
		return nil, m.InvoiceErr
	}
	return &models.LnInvoice{
		PaymentRequest: "lnbc_mock_invoice",
		PaymentHash:    "hash_mock_12345",
		Satoshis:       amountSats,
	}, nil
}

func (m *MockLightningProvider) PayToAddress(lnAddress string, amountSats int64, memo string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.PayoutErr != nil {
		return "", m.PayoutErr
	}
	status := m.PayoutStatus
	if status == "" {
		status = "SUCCESS"
	}
	return status, nil
}

// 1. Test Happy Path: C2B Confirmation -> Conversion -> Settled
func TestC2BHappyPath(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{PricePerSat: 0.10}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	payload := models.C2BCallbackPayload{
		TransactionType: "Pay Bill",
		TransID:         "MPESA_TEST_001",
		TransTime:       "20260820120000",
		TransAmount:     "1000",
		MSISDN:          "254712345678",
		BillRefNumber:   "BSF-ORDER",
	}

	tx, err := svc.ProcessC2BConfirmation(payload)
	if err != nil {
		t.Fatalf("expected nil error on C2B confirmation, got: %v", err)
	}

	if tx.Status != models.StatusReceived && tx.Status != models.StatusSettled {
		t.Errorf("expected initial status RECEIVED or SETTLED, got %s", tx.Status)
	}

	// Wait for goroutine conversion pipeline
	time.Sleep(50 * time.Millisecond)

	settledTx, err := repo.GetByMpesaTransID("MPESA_TEST_001")
	if err != nil {
		t.Fatalf("expected to find transaction by M-Pesa TransID: %v", err)
	}

	if settledTx.Status != models.StatusSettled {
		t.Errorf("expected status SETTLED, got %s", settledTx.Status)
	}
	if settledTx.AmountSats != 10000 { // 1000 KES / 0.10 KES per Sat = 10,000 Sats
		t.Errorf("expected 10000 sats, got %d", settledTx.AmountSats)
	}
	if settledTx.ExchangeRateSource != "blink.realtimePrice" {
		t.Errorf("expected exchange rate source 'blink.realtimePrice', got %s", settledTx.ExchangeRateSource)
	}
}

// 2. Test Idempotency: Duplicate C2B Callbacks (3x Delivery)
func TestC2BIdempotencyDuplicateCallbacks(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{PricePerSat: 0.10}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	payload := models.C2BCallbackPayload{
		TransactionType: "Pay Bill",
		TransID:         "MPESA_DUP_999",
		TransAmount:     "500",
		MSISDN:          "254700000000",
	}

	// Send callback 3 times
	tx1, err1 := svc.ProcessC2BConfirmation(payload)
	tx2, err2 := svc.ProcessC2BConfirmation(payload)
	tx3, err3 := svc.ProcessC2BConfirmation(payload)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("unexpected error during duplicate callbacks: %v, %v, %v", err1, err2, err3)
	}

	if tx1.ID != tx2.ID || tx2.ID != tx3.ID {
		t.Errorf("expected identical transaction ID across duplicate deliveries, got %s, %s, %s", tx1.ID, tx2.ID, tx3.ID)
	}

	list, err := repo.List("", "")
	if err != nil || len(list) != 1 {
		t.Fatalf("expected exactly 1 transaction in repository, got %d", len(list))
	}

	audits, _ := repo.ListAuditEvents(tx1.ID)
	foundDupAudit := false
	for _, a := range audits {
		if a.Action == models.ActionDuplicateIgnored {
			foundDupAudit = true
			break
		}
	}
	if !foundDupAudit {
		t.Errorf("expected DUPLICATE_IGNORED audit event to be logged")
	}
}

// 3. Test Failure Mode: Blink Exchange Rate Unavailable
func TestConversionFailureWhenPriceFails(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{PriceErr: errors.New("blink api timeout")}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	payload := models.C2BCallbackPayload{
		TransactionType: "Pay Bill",
		TransID:         "MPESA_FAIL_01",
		TransAmount:     "250",
		MSISDN:          "254711111111",
	}

	tx, err := svc.ProcessC2BConfirmation(payload)
	if err != nil {
		t.Fatalf("initial receipt should succeed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	failedTx, err := repo.GetByID(tx.ID)
	if err != nil {
		t.Fatalf("failed to retrieve transaction: %v", err)
	}

	if failedTx.Status != models.StatusFailed {
		t.Errorf("expected transaction status FAILED on price error, got %s", failedTx.Status)
	}
}

// 4. Test State Machine Transition Enforcement
func TestInvalidStateTransitions(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()

	tx := &models.Transaction{
		ID:            "tx_state_01",
		FiatAmountKES: 100,
		Status:        models.StatusSettled,
		CreatedAt:     time.Now().UTC(),
	}
	_ = repo.Save(tx)

	// Attempting to transition from SETTLED to RECEIVED must error
	tx.Status = models.StatusReceived
	err := repo.Save(tx)
	if !errors.Is(err, repository.ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition, got %v", err)
	}
}

// 5. Test Invoice Creation & Background Expiration Sweeper
func TestInvoiceExpirationSweeper(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	_, tx, err := svc.CreateLightningInvoice(models.CreateInvoiceReq{
		Amount: 500,
		Memo:   "Expiring Invoice Test",
	})
	if err != nil {
		t.Fatalf("invoice creation failed: %v", err)
	}

	// Manually backdate ExpiresAt to simulate time passage
	past := time.Now().UTC().Add(-10 * time.Minute)
	tx.ExpiresAt = &past
	_ = repo.Save(tx)

	swept, err := svc.SweepExpiredInvoices()
	if err != nil {
		t.Fatalf("sweeper error: %v", err)
	}
	if swept != 1 {
		t.Errorf("expected 1 invoice swept, got %d", swept)
	}

	updatedTx, _ := repo.GetByID(tx.ID)
	if updatedTx.Status != models.StatusExpired {
		t.Errorf("expected status EXPIRED, got %s", updatedTx.Status)
	}
}

// 6. Test M-Pesa B2C Payout & Timeout Handling
func TestB2CPayoutAndTimeout(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	_, statusCode, err := svc.TriggerB2CPayout(500, "254708374149", "Collector Payout", "WasteCollection")
	if err != nil || statusCode != 200 {
		t.Fatalf("expected B2C payout trigger to succeed, got status %d, err: %v", statusCode, err)
	}

	txs, _ := repo.List(models.StatusPayoutPending, models.DirectionOutbound)
	if len(txs) != 1 {
		t.Fatalf("expected 1 PAYOUT_PENDING outbound transaction, got %d", len(txs))
	}

	// Simulate Daraja Timeout Notification
	svc.HandleB2CTimeoutNotification([]byte(`{"Result":{"ResultCode":1,"ResultDesc":"Request timed out"}}`))

	audits, _ := repo.ListAuditEvents("")
	foundTimeoutAudit := false
	for _, a := range audits {
		if a.Action == models.ActionB2CTimeout {
			foundTimeoutAudit = true
			break
		}
	}
	if !foundTimeoutAudit {
		t.Errorf("expected B2C_TIMEOUT audit event to be logged")
	}
}

// 7. Test Admin Reconciliation Report
func TestReconciliationReport(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	// Save a settled transaction
	now := time.Now().UTC()
	_ = repo.Save(&models.Transaction{
		ID:            "tx_rec_1",
		Direction:     models.DirectionInbound,
		FiatAmountKES: 3000,
		Status:        models.StatusSettled,
		CreatedAt:     now,
	})

	report, err := svc.GetReconciliationReport()
	if err != nil {
		t.Fatalf("reconciliation report failed: %v", err)
	}

	if report.TotalInboundKES != 3000 {
		t.Errorf("expected TotalInboundKES 3000, got %f", report.TotalInboundKES)
	}
}

// 8. Test Authorized Refund Flow
func TestAuthorizedRefundFlow(t *testing.T) {
	repo := repository.NewInMemoryPaymentRepository()
	mockMpesa := &MockMpesaProvider{}
	mockLightning := &MockLightningProvider{}
	svc := service.NewPaymentService(mockMpesa, mockLightning, repo)

	now := time.Now().UTC()
	origTx := &models.Transaction{
		ID:            "tx_orig_100",
		PayerMSISDN:   "254712345678",
		FiatAmountKES: 1500,
		Direction:     models.DirectionInbound,
		Status:        models.StatusSettled,
		CreatedAt:     now,
	}
	_ = repo.Save(origTx)

	refundTx, err := svc.IssueRefund("tx_orig_100", "Order Cancelled by Hotel")
	if err != nil {
		t.Fatalf("refund failed: %v", err)
	}

	if refundTx.FiatAmountKES != 1500 {
		t.Errorf("expected refund amount 1500, got %f", refundTx.FiatAmountKES)
	}
	if refundTx.Direction != models.DirectionOutbound {
		t.Errorf("expected OUTBOUND refund direction, got %s", refundTx.Direction)
	}
}
