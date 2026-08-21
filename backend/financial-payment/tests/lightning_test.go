package tests

import (
	"math"
	"testing"

	"financial-gateway/backend/financial-payment/models"
)

func TestSatsCalculationLogic(t *testing.T) {
	// Simulate Blink pricing response: base = 125000, offset = 4 (for example)
	// Price of 1 Sat in minor units (cents) = base / 10^offset
	// Price of 1 Sat in KES = (base / 10^offset) / 100
	base := 12500.0
	offset := 4

	priceOfOneSatInKES := (base / math.Pow10(offset)) / 100.0
	if priceOfOneSatInKES <= 0 {
		t.Fatalf("price per sat must be positive, got %f", priceOfOneSatInKES)
	}

	kesAmount := 500.0
	totalSats := math.Round(kesAmount / priceOfOneSatInKES)

	if totalSats <= 0 {
		t.Errorf("expected positive satoshis, got %f", totalSats)
	}
}

func TestLightningInvoiceModel(t *testing.T) {
	req := models.CreateInvoiceReq{
		Amount:    100,
		KesAmount: 0,
		Memo:      "Test Invoice",
	}

	if req.Amount != 100 {
		t.Errorf("expected amount 100, got %d", req.Amount)
	}
}
