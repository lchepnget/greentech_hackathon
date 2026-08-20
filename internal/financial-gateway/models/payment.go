package models

import "time"

// Transaction Status Constants (State Machine)
const (
	StatusReceived      = "RECEIVED"
	StatusConverting    = "CONVERTING"
	StatusPayoutPending = "PAYOUT_PENDING"
	StatusSettled       = "SETTLED"
	StatusExpired       = "EXPIRED"
	StatusFailed        = "FAILED"
	StatusTimedOut      = "TIMED_OUT"
)

// Transaction Direction Constants
const (
	DirectionInbound  = "INBOUND"  // Hotel/Customer -> GreenTech
	DirectionOutbound = "OUTBOUND" // GreenTech -> Waste Collector/Supplier
)

// Settlement Method Constants
const (
	SettlementLightning = "LIGHTNING"
	SettlementMpesaB2C  = "MPESA_B2C"
)

// Audit Action Constants
const (
	ActionC2BValidated          = "C2B_VALIDATED"
	ActionC2BConfirmed          = "C2B_CONFIRMED"
	ActionConversionStarted     = "CONVERSION_STARTED"
	ActionConversionCompleted   = "CONVERSION_COMPLETED"
	ActionConversionFailed      = "CONVERSION_FAILED"
	ActionInvoiceCreated        = "INVOICE_CREATED"
	ActionInvoiceExpired        = "INVOICE_EXPIRED"
	ActionLightningPayoutSent   = "LIGHTNING_PAYOUT_SENT"
	ActionLightningPayoutFailed = "LIGHTNING_PAYOUT_FAILED"
	ActionB2CTriggered          = "B2C_TRIGGERED"
	ActionB2CResultReceived     = "B2C_RESULT_RECEIVED"
	ActionB2CTimeout            = "B2C_TIMEOUT"
	ActionRefundIssued          = "REFUND_ISSUED"
	ActionDuplicateIgnored      = "DUPLICATE_IGNORED"
)

// Minimum Satoshis limit (supports micro-payments down to 1 Sat)
const MinPayoutSats int64 = 1

// Transaction represents a canonical record across both Fiat (M-Pesa) and Lightning rails
type Transaction struct {
	ID                    string     `json:"id"`
	MpesaTransID          string     `json:"mpesaTransId,omitempty"` // Safaricom TransID (C2B) or ConversationID (B2C)
	MpesaType             string     `json:"mpesaType,omitempty"`    // C2B | B2C
	PayerMSISDN           string     `json:"payerMsisdn,omitempty"`  // Sender phone number
	PayeeMSISDN           string     `json:"payeeMsisdn,omitempty"`  // Recipient phone number
	FiatAmountKES         float64    `json:"fiatAmountKes"`          // Authoritative KES amount
	PaymentHash           string     `json:"paymentHash,omitempty"`  // Blink invoice/payment hash
	AmountSats            int64      `json:"amountSats"`             // Integer satoshis
	ExchangeRate          float64    `json:"exchangeRate,omitempty"` // KES per Sat at time of conversion
	ExchangeRateSource    string     `json:"exchangeRateSource,omitempty"` // e.g. "blink.realtimePrice"
	ExchangeRateTimestamp *time.Time `json:"exchangeRateTimestamp,omitempty"`
	Direction             string     `json:"direction"`              // INBOUND | OUTBOUND
	SettlementMethod      string     `json:"settlementMethod"`       // LIGHTNING | MPESA_B2C
	Status                string     `json:"status"`                 // State machine state
	Memo                  string     `json:"memo,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	ExpiresAt             *time.Time `json:"expiresAt,omitempty"`
	SettledAt             *time.Time `json:"settledAt,omitempty"`
	UpdatedAt             time.Time  `json:"updatedAt"`

	// Legacy field aliases for frontend / backwards-compatibility
	Type           string  `json:"type,omitempty"`
	Amount         float64 `json:"amount,omitempty"`
	Currency       string  `json:"currency,omitempty"`
	SatsEquivalent float64 `json:"satsEquivalent,omitempty"`
	Sender         string  `json:"sender,omitempty"`
	Recipient      string  `json:"recipient,omitempty"`
	TransactionID  string  `json:"transactionId,omitempty"`
}

// Payment is an alias for Transaction to preserve backwards compatibility
type Payment = Transaction

// AuditEvent records tamper-evident actions throughout the payment lifecycle
type AuditEvent struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transactionId"`
	Action        string    `json:"action"`
	Actor         string    `json:"actor"` // "system", "daraja_webhook", "blink_webhook", "admin"
	ProviderRef   string    `json:"providerRef,omitempty"`
	Details       string    `json:"details"`
	Timestamp     time.Time `json:"timestamp"`
}

// ReconciliationReport provides auditing and financial integrity metrics
type ReconciliationReport struct {
	UnmatchedInboundCount int           `json:"unmatchedInboundCount"`
	UnmatchedInbound      []Transaction `json:"unmatchedInbound"`
	StuckPendingCount     int           `json:"stuckPendingCount"`
	StuckPending          []Transaction `json:"stuckPending"`
	DuplicateTransIDs     []string      `json:"duplicateTransIds"`
	TotalInboundKES       float64       `json:"totalInboundKes"`
	TotalOutboundSats     int64         `json:"totalOutboundSats"`
	GeneratedAt           time.Time     `json:"generatedAt"`
}

// --- DARAJA (M-PESA) MODELS ---

type MpesaAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type C2BRegistrationPayload struct {
	ShortCode       string `json:"ShortCode"`
	ResponseType    string `json:"ResponseType"`
	ConfirmationURL string `json:"ConfirmationURL"`
	ValidationURL   string `json:"ValidationURL"`
}

type C2BCallbackPayload struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
}

type B2CPayoutRequest struct {
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             int    `json:"Amount"`
	PartyA             string `json:"PartyA"`
	PartyB             string `json:"PartyB"`
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occasion           string `json:"Occasion"`
}

type B2CResultPayload struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
	} `json:"Result"`
}

// --- BLINK GRAPHQL MODELS ---

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type BlinkRealtimePriceResponse struct {
	Data struct {
		RealtimePrice struct {
			Timestamp   int64 `json:"timestamp"`
			BtcSatPrice struct {
				Base   float64 `json:"base"`
				Offset int     `json:"offset"`
			} `json:"btcSatPrice"`
			UsdCentPrice struct {
				Base   float64 `json:"base"`
				Offset int     `json:"offset"`
			} `json:"usdCentPrice"`
			DenominatorCurrency string `json:"denominatorCurrency"`
		} `json:"realtimePrice"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type BlinkWalletMe struct {
	ID             string  `json:"id"`
	Username       *string `json:"username"`
	DefaultAccount struct {
		DefaultWalletId string `json:"defaultWalletId"`
		Wallets         []struct {
			ID             string  `json:"id"`
			WalletCurrency string  `json:"walletCurrency"`
			Balance        float64 `json:"balance"`
		} `json:"wallets"`
	} `json:"defaultAccount"`
}

type BlinkWalletQueryResponse struct {
	Data struct {
		Me BlinkWalletMe `json:"me"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// --- LIGHTNING INVOICES & PAYMENTS ---

type CreateInvoiceReq struct {
	Amount    int64   `json:"amount"`    // In Satoshis
	KesAmount float64 `json:"kesAmount"` // Optional: If provided, will auto-convert KES to Sats
	Memo      string  `json:"memo"`
}

type LnInvoice struct {
	PaymentRequest string `json:"paymentRequest"`
	PaymentHash    string `json:"paymentHash"`
	PaymentSecret  string `json:"paymentSecret"`
	Satoshis       int64  `json:"satoshis"`
}

type LnInvoiceCreateResponse struct {
	Data struct {
		LnInvoiceCreate struct {
			Invoice LnInvoice `json:"invoice"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"lnInvoiceCreate"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type PayAddressReq struct {
	LnAddress string `json:"lnAddress"`
	Amount    int64  `json:"amount"` // in Sats
	Memo      string `json:"memo"`
}

type LnAddressPaymentSendResponse struct {
	Data struct {
		LnAddressPaymentSend struct {
			Status string `json:"status"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"lnAddressPaymentSend"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}
