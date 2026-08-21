# 🌿 RegenFeed — GreenTech Circular Economy & Dual-Rail Financial Gateway

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![SvelteKit](https://img.shields.io/badge/SvelteKit-5.x-FF3E00?style=flat&logo=svelte)](https://kit.svelte.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org)
[![Vite](https://img.shields.io/badge/Vite-8.x-646CFF?style=flat&logo=vite)](https://vitejs.dev)
[![M-Pesa Daraja](https://img.shields.io/badge/M--Pesa-Daraja_2.0-00A651?style=flat)](https://developer.safaricom.co.ke)
[![Bitcoin Lightning](https://img.shields.io/badge/Bitcoin-Lightning_Network-F7931A?style=flat&logo=bitcoin)](https://blink.sv)

> **Empowering Circular Agriculture**: Transforming hotel and municipal food waste into high-protein Black Soldier Fly (BSF) animal feed and organic bio-fertilizer — powered by a dual-rail financial gateway bridging **Kenyan Shillings (KES)** mobile money with the **Bitcoin Lightning Network (Satoshis)**.

---

## 📌 Table of Contents

- [Overview & Circular Model](#-overview--circular-model)
- [System Architecture](#-system-architecture)
- [Key Features](#-key-features)
- [Directory Layout](#-directory-layout)
- [Dual-Rail Financial Engine](#-dual-rail-financial-engine)
- [API Reference](#-api-reference)
- [Getting Started](#-getting-started)
  - [Prerequisites](#prerequisites)
  - [Environment Configuration](#environment-configuration)
  - [Running the Backend](#running-the-backend)
  - [Running the Frontend](#running-the-frontend)
- [Testing & Validation](#-testing--validation)
- [Security & Idempotency](#-security--idempotency)
- [Documentation Index](#-documentation-index)

---

## 🌿 Overview & Circular Model

**RegenFeed** solves two intertwined environmental and economic problems in emerging markets:
1. **Urban Organic Waste**: Massive quantities of food waste from hotels and restaurants end up in landfills, producing methane and polluting waterways.
2. **Protein Scarcity & High Fertilizer Costs**: Smallholder farmers struggle with skyrocketing prices for commercial livestock feed and synthetic fertilizers.

### The RegenFeed Solution:
- **Waste Collection**: Hotels and restaurants log food waste pickups. Waste collectors deliver raw organic substrate to localized BSF bioconversion facilities.
- **Bioconversion**: Black Soldier Fly larvae consume the organic waste, yielding:
  - **High-Protein Larvae Feed**: Sustainable, nutrient-rich feed for poultry, pigs, and aquaculture.
  - **Organic Frass Fertilizer**: Natural soil-enriching bio-fertilizer for regenerative crop cultivation.
- **Dual-Rail Micro-Incentives**:
  - **Hotels / Buyers** pay in **Kenyan Shillings (KES)** via **Safaricom M-Pesa C2B**.
  - **Waste Collectors / Suppliers** receive automated disbursements in either **Satoshis (Bitcoin Lightning)** or **KES (M-Pesa B2C)**.

---

## 🏗️ System Architecture

```
┌──────────────────────────┐                      ┌───────────────────────────┐
│     Hotel / Customer     │                      │ Waste Collector / Supplier│
│ (Purchases BSF Products) │                      │  (Supplies Organic Waste) │
└─────────────┬────────────┘                      └─────────────▲─────────────┘
              │ (Pays KES via M-Pesa)                           │ (Receives Sats / KES)
              ▼                                                 │
┌───────────────────────────────────────────────────────────────┴─────────────┐
│                       RegenFeed Core Backend Engine                         │
│                           (Go 1.22 + net/http)                              │
│                                                                             │
│  ┌──────────────────────┐  ┌─────────────────────┐  ┌────────────────────┐  │
│  │ M-Pesa C2B/B2C Hooks │  │ Asynchronous Engine │  │ In-Memory Ledger   │  │
│  │ (Daraja OAuth 2.0)   │  │ (Goroutine Pipeline)│  │ & Audit Trail Repo │  │
│  └──────────┬───────────┘  └──────────┬──────────┘  └─────────▲──────────┘  │
│             │                         │                       │             │
│             ▼                         ▼                       │             │
│  ┌──────────────────────┐  ┌─────────────────────┐            │             │
│  │ Blink GraphQL Client │  │ Dual-Rail FSM       │────────────┘             │
│  │ (Oracle & Invoicing) │  │ (State Machine)     │                          │
│  └──────────────────────┘  └─────────────────────┘                          │
└──────────────┬────────────────────────────────────────────────┬─────────────┘
               │                                                │
               ▼                                                ▼
┌─────────────────────────────┐                  ┌────────────────────────────┐
│   Safaricom Daraja API      │                  │   Blink GraphQL API        │
│   (C2B & B2C Gateways)      │                  │   (Lightning & Sats Oracle)│
└─────────────────────────────┘                  └────────────────────────────┘
```

---

## ✨ Key Features

- **⚡ Dual-Rail Payment Engine**:
  - **M-Pesa C2B**: Instant payment notifications, validation, and confirmation webhooks.
  - **M-Pesa B2C**: Automated payout disbursements to collector mobile numbers.
  - **Bitcoin Lightning (Blink)**: Zero-node-ops BOLT11 invoice generation and Lightning Address (`collector@blink.sv`) payouts.
- **🔄 Real-Time Currency Conversion**:
  - High-precision live fiat-to-Sats price oracle (`KES ⇄ Sats`).
  - Asynchronous conversion pipelines using Go goroutines with non-blocking confirmation callbacks.
- **🛡️ Enterprise Reliability & Ledger**:
  - Thread-safe in-memory repository with idempotency deduplication (`TransID` & `PaymentHash`).
  - Immutable audit logs for compliance, dispute resolution, and security tracing.
  - In-process background sweeper for expiring old invoices and cleaning up in-flight states.
  - Dual-rail reconciliation engine identifying anomalies, stuck transactions, and volume discrepancies.
- **💻 Modern Reactive Frontend**:
  - Built with **SvelteKit 5** (Runes reactivity) and **TypeScript**.
  - Fast asset compilation with **Vite**.
  - Responsive marketplace, wallet overview, listing management, and authentication flows.

---

## 📁 Directory Layout

```
greentech_hackathon/
├── backend/
│   └── financial-payment/          # Modular Go backend services
│       ├── config/                 # Environment variables & credentials loader
│       │   └── config.go
│       ├── handlers/               # HTTP webhooks, controllers & endpoints
│       │   ├── lightning_handler.go
│       │   ├── mpesa_handler.go
│       │   └── payment_handler.go
│       ├── infrastructure/         # External integrations & API clients
│       │   ├── lightning/          # Blink GraphQL client, invoices & payouts
│       │   │   ├── client.go
│       │   │   ├── interface.go
│       │   │   ├── invoice.go
│       │   │   └── payment.go
│       │   └── mpesa/              # Safaricom Daraja REST clients (OAuth, C2B, B2C)
│       │       ├── b2c.go
│       │       ├── c2b.go
│       │       ├── client.go
│       │       └── interface.go
│       ├── models/                 # Domain schemas, state machine & audit entities
│       │   └── payment.go
│       ├── repository/             # Concurrency-safe ledger & audit trail
│       │   └── payment_repository.go
│       ├── service/                # Core business logic & conversion pipelines
│       │   └── payment_service.go
│       └── tests/                  # Comprehensive unit & integration test suite
│           ├── lightning_test.go
│           └── payment_test.go
├── frontend/                       # SvelteKit 5 + TypeScript + Vite UI
│   ├── src/
│   │   ├── lib/                    # Shared components, stores, and API clients
│   │   │   ├── api/                # Typed REST client & endpoints
│   │   │   ├── components/         # Navigation, layout & UI components
│   │   │   ├── stores/             # Session & wallet state
│   │   │   └── types/              # Domain models & interfaces
│   │   └── routes/                 # SvelteKit application routes
│   │       ├── auth/               # Login, register & password recovery
│   │       ├── dashboard/          # Role-based dashboard & new listing form
│   │       ├── marketplace/        # BSF product catalog & detail views
│   │       ├── settings/           # Profile & account configuration
│   │       └── wallet/             # Dual-rail wallet balance & transactions
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/                           # Architecture & Payment Documentation
│   └── payments/
│       ├── architecture.md         # Full payment system design
│       ├── lightning-provider.md   # Blink integration & custody model
│       ├── mpesa-integration.md    # Daraja OAuth, C2B & B2C flows
│       ├── reconciliation.md       # Discrepancy detection & audit reports
│       ├── refunds.md              # Authorized dispute & reversal policies
│       └── security.md             # Idempotency, validation & secrets rules
├── main.go                         # Backend server entrypoint & router
├── test_queries.sh                 # End-to-end payment testing bash script
├── go.mod                          # Go module dependencies
└── README.md
```

---

## 💳 Dual-Rail Financial Engine

### State Machine Lifecycle
Every payment transitions strictly through a finite state machine (FSM):

$$\text{RECEIVED} \longrightarrow \text{CONVERTING} \longrightarrow \text{PAYOUT\_PENDING} \longrightarrow \text{SETTLED}$$

- **`EXPIRED`**: Triggered when a BOLT11 invoice exceeds its 1-hour time-to-live.
- **`FAILED` / `TIMED_OUT`**: Recorded if Daraja or Blink upstream calls fail or time out.
- **`REFUNDED`**: Admin-authorized reversal via programmatic M-Pesa B2C.

---

## 📡 API Reference

### 1. Safaricom Daraja M-Pesa Endpoints
| Method | Path | Description |
|---|---|---|
| `POST` | `/api/mpesa/register-urls` | Register Daraja C2B validation and confirmation webhook URLs |
| `POST` | `/api/mpesa/c2b-validation` | M-Pesa pre-clearance validation webhook |
| `POST` | `/api/mpesa/c2b-confirmation` | Authoritative M-Pesa payment confirmation webhook |
| `POST` | `/api/mpesa/trigger-b2c` | Initiate automated B2C supplier payout |
| `POST` | `/api/mpesa/b2c-result` | Daraja B2C payout result webhook |
| `POST` | `/api/mpesa/b2c-timeout` | Daraja B2C payout timeout callback |

### 2. Bitcoin Lightning & Blink Endpoints
| Method | Path | Description |
|---|---|---|
| `GET` | `/api/blink/wallet` | Query connected Blink wallet balance (Sats & USD) |
| `GET` | `/api/blink/price?amount=500` | Fetch real-time KES ⇄ Satoshis conversion rate |
| `POST` | `/api/lightning/create-invoice` | Generate BOLT11 invoice (by Satoshis or KES amount) |
| `POST` | `/api/lightning/pay-address` | Disburse Sats to a Lightning Address (`name@domain.com`) |

### 3. Application & Marketplace Endpoints
| Method | Path | Description |
|---|---|---|
| `GET` | `/api/health` | Service health status check |
| `GET` | `/api/payments` | List payment ledger transactions |
| `GET` | `/api/admin/reconciliation` | Run automated dual-rail audit reconciliation |
| `POST` | `/api/admin/refund` | Issue an authorized reversal for a settled payment |
| `GET` | `/api/listings` | Fetch marketplace waste & BSF product listings |
| `GET` | `/api/wallet` | Fetch authenticated user wallet balances |
| `POST` | `/api/wallet/deposit` | Create a deposit invoice or initiate STK push |
| `POST` | `/api/wallet/withdraw` | Withdraw funds via Lightning or M-Pesa |

---

## 🚀 Getting Started

### Prerequisites
- **Go**: `1.22.x` or higher ([Download](https://golang.org/dl/))
- **Node.js**: `18.x` or higher & **npm** `9.x+` ([Download](https://nodejs.org/))

---

### Environment Configuration

Create a `.env` file in the root directory:

```bash
# Server Port
PORT=8080

# Safaricom Daraja Sandbox Credentials
MPESA_ENV="sandbox"
MPESA_CONSUMER_KEY="your_daraja_consumer_key"
MPESA_CONSUMER_SECRET="your_daraja_consumer_secret"
MPESA_C2B_SHORTCODE="600990"
MPESA_C2B_CONFIRMATION_URL="https://your-domain.ngrok.app/api/mpesa/c2b-confirmation"
MPESA_C2B_VALIDATION_URL="https://your-domain.ngrok.app/api/mpesa/c2b-validation"

# M-Pesa B2C Payout Settings
MPESA_B2C_SHORTCODE="600000"
MPESA_INITIATOR_NAME="testapi"
MPESA_SECURITY_CREDENTIAL="your_b2c_security_credential"
MPESA_B2C_RESULT_URL="https://your-domain.ngrok.app/api/mpesa/b2c-result"
MPESA_B2C_TIMEOUT_URL="https://your-domain.ngrok.app/api/mpesa/b2c-timeout"

# Blink Bitcoin & Lightning Credentials
BLINK_API_URL="https://api.blink.sv/graphql"
BLINK_API_KEY="your_blink_api_key"
BLINK_WALLET_ID="your_blink_wallet_id"
```

Configure the frontend API URL in `frontend/.env`:

```bash
PUBLIC_API_BASE_URL=http://localhost:8080/api
```

---

### Running the Backend

```bash
# From project root
go run main.go
```
The Go financial engine will start at `http://localhost:8080`.

---

### Running the Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies (if not already installed)
npm install

# Start Vite development server
npm run dev -- --port 5173
```
Access the web application at `http://localhost:5173`.

---

## 🧪 Testing & Validation

### 1. Run the Go Test Suite
The backend includes extensive unit and integration tests covering the Sats conversion pipeline, invoice lifecycle, idempotency guards, and state transitions:

```bash
go test -v ./...
```

### 2. Frontend Type & Syntax Verification
```bash
cd frontend && npm run check
```

### 3. Interactive Payment Pipeline Simulation
You can simulate live M-Pesa C2B payments, live exchange rate queries, and Lightning invoice generation using the provided test script:

```bash
chmod +x test_queries.sh
./test_queries.sh
```

---

## 🔒 Security & Idempotency

- **Zero Hardcoded Secrets**: All sensitive keys, secrets, and URLs are loaded via environment variables.
- **Deduplication Guards**: Webhook endpoints enforce idempotency on `TransID` and `PaymentHash` to prevent double-crediting or duplicate disbursements.
- **Strict Data Validation**: Payloads are validated before entering the pipeline; non-conforming requests are rejected with explicit HTTP errors.
- **Authoritative Amounts**: The backend strictly relies on confirmed gateway callback amounts, preventing client-side price tampering.

---

## 📚 Documentation Index

For in-depth specifications, review the docs in `docs/payments/`:
- [Architecture & Circular Economy Pipeline](docs/payments/architecture.md)
- [Blink & Lightning Provider Architecture](docs/payments/lightning-provider.md)
- [Safaricom Daraja M-Pesa Integration](docs/payments/mpesa-integration.md)
- [Dual-Rail Reconciliation Engine](docs/payments/reconciliation.md)
- [Refunds & Reversals Policy](docs/payments/refunds.md)
- [Security, Idempotency & Secrets Management](docs/payments/security.md)

---

## 👥 Contributors
Developed for the **GreenTech Hackathon 2026**.
