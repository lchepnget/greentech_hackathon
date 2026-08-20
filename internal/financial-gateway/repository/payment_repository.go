package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"financial-gateway/internal/financial-gateway/models"
)

// Common repository errors
var (
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrDuplicateMpesaTransID = errors.New("duplicate mpesa transaction id")
	ErrDuplicatePaymentHash  = errors.New("duplicate payment hash")
	ErrInvalidStateTransition= errors.New("invalid state machine transition")
	ErrNilTransaction        = errors.New("cannot save nil transaction")
)

// PaymentRepository defines the data access contract for payment tracking & audit trails
type PaymentRepository interface {
	Save(tx *models.Transaction) error
	GetByID(id string) (*models.Transaction, error)
	GetByMpesaTransID(transID string) (*models.Transaction, error)
	GetByPaymentHash(hash string) (*models.Transaction, error)
	List(filterStatus, filterDirection string) ([]*models.Transaction, error)
	UpdateStatus(id string, newStatus string, settledAt *time.Time) error
	RecordAuditEvent(event *models.AuditEvent) error
	ListAuditEvents(txID string) ([]*models.AuditEvent, error)
	GenerateReconciliationReport() (*models.ReconciliationReport, error)
	SweepExpiredInvoices() (int, error)
}

// InMemoryPaymentRepository implements PaymentRepository with thread-safe in-memory storage & constraints
type InMemoryPaymentRepository struct {
	mu            sync.RWMutex
	transactions  map[string]*models.Transaction
	mpesaTransMap map[string]string // mpesaTransID -> txID
	paymentHashMap map[string]string // paymentHash -> txID
	order         []string
	auditEvents   []*models.AuditEvent
}

// NewInMemoryPaymentRepository creates a new repository instance
func NewInMemoryPaymentRepository() *InMemoryPaymentRepository {
	return &InMemoryPaymentRepository{
		transactions:   make(map[string]*models.Transaction),
		mpesaTransMap:  make(map[string]string),
		paymentHashMap: make(map[string]string),
		order:          make([]string, 0),
		auditEvents:    make([]*models.AuditEvent, 0),
	}
}

// isValidTransition validates transaction state transitions according to the state machine
func isValidTransition(currentStatus, newStatus string) bool {
	if currentStatus == newStatus {
		return true // Idempotent no-op
	}

	switch currentStatus {
	case models.StatusReceived:
		return newStatus == models.StatusConverting || newStatus == models.StatusSettled || newStatus == models.StatusFailed
	case models.StatusConverting:
		return newStatus == models.StatusPayoutPending || newStatus == models.StatusSettled || newStatus == models.StatusFailed
	case models.StatusPayoutPending:
		return newStatus == models.StatusSettled || newStatus == models.StatusExpired || newStatus == models.StatusFailed || newStatus == models.StatusTimedOut
	case models.StatusTimedOut:
		return newStatus == models.StatusSettled || newStatus == models.StatusFailed // Allow late B2C confirmation or permanent failure
	case models.StatusSettled:
		return false // Terminal state
	case models.StatusExpired:
		return false // Terminal state unless fresh invoice
	case models.StatusFailed:
		return false // Terminal state
	default:
		return true
	}
}

// Save inserts or updates a transaction record with uniqueness checks and transition guards
func (r *InMemoryPaymentRepository) Save(tx *models.Transaction) error {
	if tx == nil {
		return ErrNilTransaction
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Fill legacy fields for compatibility
	if tx.Amount == 0 && tx.FiatAmountKES > 0 {
		tx.Amount = tx.FiatAmountKES
		tx.Currency = "KES"
	}
	if tx.SatsEquivalent == 0 && tx.AmountSats > 0 {
		tx.SatsEquivalent = float64(tx.AmountSats)
	}
	if tx.TransactionID == "" {
		if tx.MpesaTransID != "" {
			tx.TransactionID = tx.MpesaTransID
		} else if tx.PaymentHash != "" {
			tx.TransactionID = tx.PaymentHash
		}
	}
	if tx.UpdatedAt.IsZero() {
		tx.UpdatedAt = time.Now().UTC()
	}

	existing, exists := r.transactions[tx.ID]
	if exists {
		if !isValidTransition(existing.Status, tx.Status) {
			return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, existing.Status, tx.Status)
		}
	} else {
		// Enforce unique M-Pesa TransID constraint
		if tx.MpesaTransID != "" {
			if existingID, dup := r.mpesaTransMap[tx.MpesaTransID]; dup && existingID != tx.ID {
				return fmt.Errorf("%w: %s already associated with transaction %s", ErrDuplicateMpesaTransID, tx.MpesaTransID, existingID)
			}
			r.mpesaTransMap[tx.MpesaTransID] = tx.ID
		}

		// Enforce unique PaymentHash constraint
		if tx.PaymentHash != "" {
			if existingID, dup := r.paymentHashMap[tx.PaymentHash]; dup && existingID != tx.ID {
				return fmt.Errorf("%w: %s already associated with transaction %s", ErrDuplicatePaymentHash, tx.PaymentHash, existingID)
			}
			r.paymentHashMap[tx.PaymentHash] = tx.ID
		}

		r.order = append(r.order, tx.ID)
	}

	copied := *tx
	r.transactions[tx.ID] = &copied
	return nil
}

// GetByID finds a transaction by internal unique ID
func (r *InMemoryPaymentRepository) GetByID(id string) (*models.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tx, exists := r.transactions[id]
	if !exists {
		return nil, ErrTransactionNotFound
	}
	res := *tx
	return &res, nil
}

// GetByMpesaTransID finds a transaction by Safaricom TransID (C2B) or ConversationID (B2C)
func (r *InMemoryPaymentRepository) GetByMpesaTransID(transID string) (*models.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	txID, exists := r.mpesaTransMap[transID]
	if exists {
		res := *r.transactions[txID]
		return &res, nil
	}

	for _, tx := range r.transactions {
		if tx.MpesaTransID == transID || tx.TransactionID == transID {
			res := *tx
			return &res, nil
		}
	}
	return nil, ErrTransactionNotFound
}

// GetByPaymentHash finds a transaction by Blink invoice/payment hash
func (r *InMemoryPaymentRepository) GetByPaymentHash(hash string) (*models.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	txID, exists := r.paymentHashMap[hash]
	if exists {
		res := *r.transactions[txID]
		return &res, nil
	}

	for _, tx := range r.transactions {
		if tx.PaymentHash == hash {
			res := *tx
			return &res, nil
		}
	}
	return nil, ErrTransactionNotFound
}

// List returns recorded transactions in reverse chronological order with optional filters
func (r *InMemoryPaymentRepository) List(filterStatus, filterDirection string) ([]*models.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*models.Transaction, 0, len(r.order))
	for i := len(r.order) - 1; i >= 0; i-- {
		id := r.order[i]
		tx := r.transactions[id]

		if filterStatus != "" && tx.Status != filterStatus {
			continue
		}
		if filterDirection != "" && tx.Direction != filterDirection {
			continue
		}

		res := *tx
		result = append(result, &res)
	}
	return result, nil
}

// UpdateStatus atomically updates status and timestamps with state-machine validation
func (r *InMemoryPaymentRepository) UpdateStatus(id string, newStatus string, settledAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, exists := r.transactions[id]
	if !exists {
		return ErrTransactionNotFound
	}

	if !isValidTransition(tx.Status, newStatus) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, tx.Status, newStatus)
	}

	tx.Status = newStatus
	tx.UpdatedAt = time.Now().UTC()
	if settledAt != nil {
		tx.SettledAt = settledAt
	}
	return nil
}

// RecordAuditEvent stores an immutable audit event entry
func (r *InMemoryPaymentRepository) RecordAuditEvent(event *models.AuditEvent) error {
	if event == nil {
		return errors.New("cannot record nil audit event")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	r.auditEvents = append(r.auditEvents, event)
	return nil
}

// ListAuditEvents retrieves all audit logs, optionally filtered by transaction ID
func (r *InMemoryPaymentRepository) ListAuditEvents(txID string) ([]*models.AuditEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*models.AuditEvent, 0)
	for i := len(r.auditEvents) - 1; i >= 0; i-- {
		evt := r.auditEvents[i]
		if txID != "" && evt.TransactionID != txID {
			continue
		}
		result = append(result, evt)
	}
	return result, nil
}

// SweepExpiredInvoices transitions expired pending invoices to EXPIRED
func (r *InMemoryPaymentRepository) SweepExpiredInvoices() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	swept := 0

	for _, tx := range r.transactions {
		if tx.Status == models.StatusPayoutPending && tx.ExpiresAt != nil && tx.ExpiresAt.Before(now) {
			tx.Status = models.StatusExpired
			tx.UpdatedAt = now
			swept++

			r.auditEvents = append(r.auditEvents, &models.AuditEvent{
				ID:            fmt.Sprintf("audit_sweep_%d", time.Now().UnixNano()),
				TransactionID: tx.ID,
				Action:        models.ActionInvoiceExpired,
				Actor:         "background_sweeper",
				ProviderRef:   tx.PaymentHash,
				Details:       "Invoice passed expiry timestamp and was marked EXPIRED",
				Timestamp:     now,
			})
		}
	}

	return swept, nil
}

// GenerateReconciliationReport runs integrity checks across inbound and outbound flows
func (r *InMemoryPaymentRepository) GenerateReconciliationReport() (*models.ReconciliationReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now().UTC()
	report := &models.ReconciliationReport{
		UnmatchedInbound:  make([]models.Transaction, 0),
		StuckPending:      make([]models.Transaction, 0),
		DuplicateTransIDs: make([]string, 0),
		GeneratedAt:       now,
	}

	seenMpesaIDs := make(map[string]int)

	for _, tx := range r.transactions {
		if tx.Direction == models.DirectionInbound {
			report.TotalInboundKES += tx.FiatAmountKES
		} else if tx.Direction == models.DirectionOutbound && tx.Status == models.StatusSettled {
			report.TotalOutboundSats += tx.AmountSats
		}

		if tx.MpesaTransID != "" {
			seenMpesaIDs[tx.MpesaTransID]++
		}

		// Check for stuck pending transactions (> 15 minutes)
		if tx.Status == models.StatusPayoutPending || tx.Status == models.StatusConverting {
			if now.Sub(tx.CreatedAt) > 15*time.Minute {
				report.StuckPending = append(report.StuckPending, *tx)
			}
		}
	}

	for id, count := range seenMpesaIDs {
		if count > 1 {
			report.DuplicateTransIDs = append(report.DuplicateTransIDs, id)
		}
	}

	report.UnmatchedInboundCount = len(report.UnmatchedInbound)
	report.StuckPendingCount = len(report.StuckPending)

	return report, nil
}
