package lightning

import "financial-gateway/backend/financial-payment/models"

// LightningProvider defines the abstract contract for interacting with Bitcoin Lightning
type LightningProvider interface {
	GetWalletInfo() (*models.BlinkWalletMe, error)
	ConvertFiatToSats(amount float64, currency string) (float64, float64, error)
	CreateInvoice(amountSats int64, memo string) (*models.LnInvoice, error)
	PayToAddress(lnAddress string, amountSats int64, memo string) (string, error)
}
