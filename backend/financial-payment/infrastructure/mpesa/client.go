package mpesa

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"financial-gateway/backend/financial-payment/config"
	"financial-gateway/backend/financial-payment/models"
)

// MpesaClient handles authentication and communication with Safaricom Daraja API
type MpesaClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewMpesaClient creates a new instance of MpesaClient
func NewMpesaClient(cfg *config.Config) *MpesaClient {
	return &MpesaClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetBaseURL returns the Daraja gateway endpoint based on environment
func (c *MpesaClient) GetBaseURL() string {
	if c.cfg.MpesaEnv == "production" || c.cfg.MpesaEnv == "prod" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

// GetToken retrieves dynamic OAuth bearer authentication token from Safaricom API Gateway
func (c *MpesaClient) GetToken() (string, error) {
	authStr := fmt.Sprintf("%s:%s", c.cfg.MpesaKey, c.cfg.MpesaSecret)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authStr))

	authURL := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", c.GetBaseURL())
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+encodedAuth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("daraja auth error (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp models.MpesaAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}
	return authResp.AccessToken, nil
}
