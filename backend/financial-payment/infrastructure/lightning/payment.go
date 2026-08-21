package lightning

import (
	"fmt"

	"financial-gateway/backend/financial-payment/models"
)

// PayToAddress sends Satoshis to a Lightning Address (e.g., collector@blink.sv)
func (c *BlinkClient) PayToAddress(lnAddress string, amountSats int64, memo string) (string, error) {
	if lnAddress == "" || amountSats <= 0 {
		return "", fmt.Errorf("lnAddress and positive amount in Sats are required")
	}

	gqlMutation := `
		mutation LnAddressPaymentSend($input: LnAddressPaymentSendInput!) {
			lnAddressPaymentSend(input: $input) {
				status
				errors {
					message
				}
			}
		}
	`
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"walletId":  c.cfg.BlinkWalletID,
			"lnAddress": lnAddress,
			"amount":    amountSats,
			"memo":      memo,
		},
	}

	var payResp models.LnAddressPaymentSendResponse
	err := c.Query(gqlMutation, variables, &payResp)
	if err != nil {
		return "", fmt.Errorf("blink api connection error: %w", err)
	}

	if len(payResp.Errors) > 0 {
		return "", fmt.Errorf("blink error: %s", payResp.Errors[0].Message)
	}

	if len(payResp.Data.LnAddressPaymentSend.Errors) > 0 {
		return "", fmt.Errorf("payment send error: %s", payResp.Data.LnAddressPaymentSend.Errors[0].Message)
	}

	return payResp.Data.LnAddressPaymentSend.Status, nil
}
