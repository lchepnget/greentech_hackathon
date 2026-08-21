package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"financial-gateway/backend/financial-payment/config"
	"financial-gateway/backend/financial-payment/handlers"
	"financial-gateway/backend/financial-payment/infrastructure/lightning"
	"financial-gateway/backend/financial-payment/infrastructure/mpesa"
	"financial-gateway/backend/financial-payment/repository"
	"financial-gateway/backend/financial-payment/service"
)

func main() {
	// 1. Initialize Configuration
	cfg := config.LoadConfig()

	// 2. Initialize Repositories (with Concurrency, Idempotency & Audit Trail)
	repo := repository.NewInMemoryPaymentRepository()

	// 3. Initialize Infrastructure Clients (Dual-Rail Abstraction)
	mpesaClient := mpesa.NewMpesaClient(cfg)
	blinkClient := lightning.NewBlinkClient(cfg)

	// 4. Initialize Domain Services
	paymentService := service.NewPaymentService(mpesaClient, blinkClient, repo)

	// 5. Start Background Sweeper for Expired Invoices & In-Flight Cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	paymentService.StartBackgroundSweeper(ctx, 30*time.Second)

	// 6. Initialize HTTP Handlers
	mpesaHandler := handlers.NewMpesaHandler(paymentService)
	lightningHandler := handlers.NewLightningHandler(paymentService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	// 7. Register Routes
	// Daraja (M-Pesa) Endpoints
	http.HandleFunc("/api/mpesa/register-urls", mpesaHandler.HandleRegisterUrls)
	http.HandleFunc("/api/mpesa/c2b-validation", mpesaHandler.HandleValidation)
	http.HandleFunc("/api/mpesa/c2b-confirmation", mpesaHandler.HandleConfirmation)
	http.HandleFunc("/api/mpesa/trigger-b2c", mpesaHandler.HandleTriggerB2C)
	http.HandleFunc("/api/mpesa/b2c-result", mpesaHandler.HandleB2CResult)
	http.HandleFunc("/api/mpesa/b2c-timeout", mpesaHandler.HandleB2CTimeout)

	// Blink (Bitcoin & Lightning) Endpoints
	http.HandleFunc("/api/blink/wallet", lightningHandler.HandleWalletQuery)
	http.HandleFunc("/api/blink/price", lightningHandler.HandlePriceQuery)
	http.HandleFunc("/api/lightning/create-invoice", lightningHandler.HandleCreateInvoice)
	http.HandleFunc("/api/lightning/pay-address", lightningHandler.HandlePayAddress)

	// Root / System Status & Documentation
	http.HandleFunc("/", paymentHandler.HandleRoot)
	http.HandleFunc("/api", paymentHandler.HandleRoot)
	http.HandleFunc("/api/health", paymentHandler.HandleHealth)

	// Payment Ledger & Transaction Tracking
	http.HandleFunc("/api/payments", paymentHandler.HandleListPayments)

	// Admin Monitoring & Dual-Rail Reconciliation Endpoints
	http.HandleFunc("/api/admin/transactions", paymentHandler.HandleAdminTransactions)
	http.HandleFunc("/api/admin/transactions/", paymentHandler.HandleAdminTransactionByID)
	http.HandleFunc("/api/admin/reconciliation", paymentHandler.HandleAdminReconciliation)
	http.HandleFunc("/api/admin/audit-logs", paymentHandler.HandleAdminAuditLogs)
	http.HandleFunc("/api/admin/refund", paymentHandler.HandleAdminRefund)

	// Frontend Wallet Endpoints (Blink & Fiat Bridge)
	http.HandleFunc("/api/wallet", paymentHandler.HandleWallet)
	http.HandleFunc("/api/wallet/transactions", paymentHandler.HandleWalletTransactions)
	http.HandleFunc("/api/wallet/deposit", paymentHandler.HandleWalletDeposit)
	http.HandleFunc("/api/wallet/withdraw", paymentHandler.HandleWalletWithdraw)

	// BSF Marketplace & Orders
	http.HandleFunc("/api/listings", paymentHandler.HandleListings)
	http.HandleFunc("/api/listings/", paymentHandler.HandleListingByID)
	http.HandleFunc("/api/orders", paymentHandler.HandleOrders)

	// Authentication & User Profile
	http.HandleFunc("/api/auth/login", paymentHandler.HandleAuth)
	http.HandleFunc("/api/auth/register", paymentHandler.HandleAuth)
	http.HandleFunc("/api/auth/me", paymentHandler.HandleAuth)
	http.HandleFunc("/api/users/me", paymentHandler.HandleAuth)

	// CORS wrapper
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-KEY")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	log.Printf("🚀 Financial Gateway Service running on port :%s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, corsHandler))
}
