package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime application configurations
type Config struct {
	Port              string
	MpesaEnv          string
	MpesaKey          string
	MpesaSecret       string
	C2BShortcode      string
	C2BConfirmURL     string
	C2BValidateURL    string
	B2CShortcode      string
	B2CInitiator      string
	B2CSecurityCred   string
	B2CResultURL      string
	B2CTimeoutURL     string
	BlinkURL          string
	BlinkKey          string
	BlinkWalletID     string
}

// getEnv retrieves an environment variable or returns a fallback value
func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return fallback
	}
	return val
}

// LoadConfig initializes and parses all configuration values
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: No .env file found. Falling back to system environment variables.")
	}

	return &Config{
		Port:            getEnv("PORT", "3000"),
		MpesaEnv:        getEnv("MPESA_ENV", "sandbox"),
		MpesaKey:        getEnv("MPESA_CONSUMER_KEY", ""),
		MpesaSecret:     getEnv("MPESA_CONSUMER_SECRET", ""),
		C2BShortcode:    getEnv("MPESA_C2B_SHORTCODE", "600990"),
		C2BConfirmURL:   getEnv("MPESA_C2B_CONFIRMATION_URL", ""),
		C2BValidateURL:  getEnv("MPESA_C2B_VALIDATION_URL", ""),
		B2CShortcode:    getEnv("MPESA_B2C_SHORTCODE", "600000"),
		B2CInitiator:    getEnv("MPESA_INITIATOR_NAME", "testapi"),
		B2CSecurityCred: getEnv("MPESA_SECURITY_CREDENTIAL", ""),
		B2CResultURL:    getEnv("MPESA_B2C_RESULT_URL", ""),
		B2CTimeoutURL:   getEnv("MPESA_B2C_TIMEOUT_URL", ""),
		BlinkURL:        getEnv("BLINK_API_URL", "https://api.blink.sv/graphql"),
		BlinkKey:        getEnv("BLINK_API_KEY", ""),
		BlinkWalletID:   getEnv("BLINK_WALLET_ID", "9006a937-ebc8-4f53-9dac-00ae79e9f216"),
	}
}
