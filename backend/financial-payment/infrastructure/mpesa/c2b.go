package mpesa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"financial-gateway/backend/financial-payment/models"
)

// RegisterC2BUrls registers validation and confirmation callback endpoints with Safaricom Daraja
func (c *MpesaClient) RegisterC2BUrls() ([]byte, int, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to authenticate with Daraja: %w", err)
	}

	payload := models.C2BRegistrationPayload{
		ShortCode:       c.cfg.C2BShortcode,
		ResponseType:    "Completed",
		ConfirmationURL: c.cfg.C2BConfirmURL,
		ValidationURL:   c.cfg.C2BValidateURL,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize registration payload: %w", err)
	}

	registerURL := fmt.Sprintf("%s/mpesa/c2b/v1/registerurl", c.GetBaseURL())
	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("network transport failure: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
