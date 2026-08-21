package mpesa

// MpesaProvider defines the abstract contract for interacting with Safaricom Daraja
type MpesaProvider interface {
	GetToken() (string, error)
	RegisterC2BUrls() ([]byte, int, error)
	TriggerB2CPayout(amount int, recipientPhone, remarks, occasion string) ([]byte, int, error)
}
