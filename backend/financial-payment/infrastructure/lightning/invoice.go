package lightning

import (
	"fmt"

	"financial-gateway/backend/financial-payment/models"
)

// CreateInvoice generates a real BOLT11 Lightning payment request through Blink
func (c *BlinkClient) CreateInvoice(amountSats int64, memo string) (*models.LnInvoice, error) {
	if amountSats <= 0 {
		amountSats = 1
	}

	if memo == "" {
		memo = fmt.Sprintf("GreenTech BSF Payment (%d Sats)", amountSats)
	}

	gqlMutation := `
		mutation LnInvoiceCreate($input: LnInvoiceCreateInput!) {
			lnInvoiceCreate(input: $input) {
				invoice {
					paymentRequest
					paymentHash
					paymentSecret
					satoshis
				}
				errors {
					message
				}
			}
		}
	`
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"walletId": c.cfg.BlinkWalletID,
			"amount":   amountSats,
			"memo":     memo,
		},
	}

	var invResp models.LnInvoiceCreateResponse
	err := c.Query(gqlMutation, variables, &invResp)
	if err != nil {
		return nil, fmt.Errorf("blink api connection failed: %w", err)
	}

	if len(invResp.Errors) > 0 {
		return nil, fmt.Errorf("blink error: %s", invResp.Errors[0].Message)
	}

	if len(invResp.Data.LnInvoiceCreate.Errors) > 0 {
		return nil, fmt.Errorf("invoice creation error: %s", invResp.Data.LnInvoiceCreate.Errors[0].Message)
	}

	return &invResp.Data.LnInvoiceCreate.Invoice, nil
}
