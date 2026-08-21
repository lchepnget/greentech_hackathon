package lightning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"financial-gateway/backend/financial-payment/config"
	"financial-gateway/backend/financial-payment/models"
)

// BlinkClient communicates with the remote Blink Bitcoin backend GraphQL Gateway
type BlinkClient struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewBlinkClient creates a new Blink GraphQL client instance
func NewBlinkClient(cfg *config.Config) *BlinkClient {
	return &BlinkClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Query executes a GraphQL query or mutation against Blink API
func (c *BlinkClient) Query(query string, variables map[string]interface{}, target interface{}) error {
	payload := models.GraphQLRequest{Query: query, Variables: variables}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal graphql payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.cfg.BlinkURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	if c.cfg.BlinkKey != "" {
		req.Header.Set("X-API-KEY", c.cfg.BlinkKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("blink api request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode blink response: %w", err)
	}

	return nil
}

// GetWalletInfo queries the authenticated Blink user profile and balances
func (c *BlinkClient) GetWalletInfo() (*models.BlinkWalletMe, error) {
	gqlQuery := `
		query Me {
			me {
				id
				username
				defaultAccount {
					defaultWalletId
					wallets {
						id
						walletCurrency
						balance
					}
				}
			}
		}
	`
	var walletResp models.BlinkWalletQueryResponse
	err := c.Query(gqlQuery, nil, &walletResp)
	if err != nil {
		return nil, err
	}
	if len(walletResp.Errors) > 0 {
		return nil, fmt.Errorf("blink graphql error: %s", walletResp.Errors[0].Message)
	}

	return &walletResp.Data.Me, nil
}

// ConvertFiatToSats converts standard fiat currency amount values straight into Satoshis using live Blink rates
func (c *BlinkClient) ConvertFiatToSats(amount float64, currency string) (float64, float64, error) {
	if currency == "" {
		currency = "KES"
	}

	gqlQuery := `
		query RealTimePrice($currency: DisplayCurrency!) {
			realtimePrice(currency: $currency) {
				timestamp
				btcSatPrice {
					base
					offset
				}
				usdCentPrice {
					base
					offset
				}
				denominatorCurrency
			}
		}
	`
	variables := map[string]interface{}{
		"currency": currency,
	}

	var priceResp models.BlinkRealtimePriceResponse
	err := c.Query(gqlQuery, variables, &priceResp)
	if err != nil {
		return 0, 0, err
	}
	if len(priceResp.Errors) > 0 {
		return 0, 0, fmt.Errorf("blink graphql error: %s", priceResp.Errors[0].Message)
	}

	btcPrice := priceResp.Data.RealtimePrice.BtcSatPrice
	if btcPrice.Base == 0 {
		return 0, 0, fmt.Errorf("invalid price data returned from Blink")
	}

	// In Blink schema, btcSatPrice is price of 1 Sat in minor units (cents): base / 10^offset
	// 1 KES = 100 cents
	// Price of 1 Sat in KES = (base / 10^offset) / 100
	priceOfOneSatInCurrency := (btcPrice.Base / math.Pow10(btcPrice.Offset)) / 100.0
	if priceOfOneSatInCurrency <= 0 {
		return 0, 0, fmt.Errorf("calculated price per sat is zero or negative")
	}

	totalSats := amount / priceOfOneSatInCurrency
	return math.Round(totalSats), priceOfOneSatInCurrency, nil
}
