package mpesa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"financial-gateway/backend/financial-payment/models"
)

// TriggerB2CPayout initiates a B2C disbursement from the corporate utility account to a recipient phone number
func (c *MpesaClient) TriggerB2CPayout(amount int, recipientPhone, remarks, occasion string) ([]byte, int, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to authenticate with Daraja: %w", err)
	}

	if recipientPhone == "" {
		recipientPhone = "254708374149"
	}
	if remarks == "" {
		remarks = "Automated Swaps Payout"
	}
	if occasion == "" {
		occasion = "LightningConversion"
	}

	payload := models.B2CPayoutRequest{
		InitiatorName:      c.cfg.B2CInitiator,
		SecurityCredential: c.cfg.B2CSecurityCred,
		CommandID:          "BusinessPayment",
		Amount:             amount,
		PartyA:             c.cfg.B2CShortcode,
		PartyB:             recipientPhone,
		Remarks:            remarks,
		QueueTimeOutURL:    c.cfg.B2CTimeoutURL,
		ResultURL:          c.cfg.B2CResultURL,
		Occasion:           occasion,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize B2C payload: %w", err)
	}

	b2cURL := fmt.Sprintf("%s/mpesa/b2c/v1/paymentrequest", c.GetBaseURL())
	req, err := http.NewRequest("POST", b2cURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to build B2C request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("transport failure: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
