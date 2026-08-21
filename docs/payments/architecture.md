# GreenTech Financial Gateway — Architecture

## 1. Overview & Circular Economy Model

**GreenTech** transforms hotel organic food waste into high-value Black Soldier Fly (BSF) products (protein animal feed and organic bio-fertilizer), bridging Kenyan Shillings (KES) mobile money with the Bitcoin Lightning Network (Satoshis) ecosystem.

```
[ Hotel / Customer ]                    [ Waste Collector / Supplier ]
         │ (Pays KES via M-Pesa)                       ▲ (Receives Sats or KES)
         ▼                                             │
┌─────────────────────────────────────────────────────────────┐
│                GreenTech Go Backend Engine                  │
│                                                             │
│  • M-Pesa C2B Webhooks (capture hotel payments)             │
│  • Real-time KES ⇄ Sats conversion pipeline (goroutines)    │
│  • Lightning BOLT11 invoicing & Lightning Address payouts   │
│  • M-Pesa B2C automated supplier disbursements               │
└─────────────────────────────────────────────────────────────┘
         │                                             │
         ▼                                             ▼
[ Safaricom Daraja API ]                     [ Blink GraphQL API ]
```

---

## 2. Core Principles & Transaction Authority

1. **Backend Authority**: Neither the frontend nor external unverified callers decide transaction state, pricing, or settled amounts.
2. **Authoritative Amounts**: The authoritative payment amount is strictly whatever Safaricom reports in the C2B `TransAmount` or Blink reports for an invoice.
3. **No Recalculation**: Historical conversions are immutable. Once KES is converted to Satoshis at confirmation time, the stored rate and Satoshis value remain permanent.
4. **State Machine Integrity**:
   `RECEIVED` ➔ `CONVERTING` ➔ `PAYOUT_PENDING` ➔ `SETTLED` (or `EXPIRED` / `FAILED` / `TIMED_OUT`).

---

## 3. Directory Layout

```
backend/
└── financial-payment/
    ├── config/                 # Environment variables & credentials loader
    │   └── config.go
    ├── infrastructure/         # Low-level third-party integration pipelines
    │   ├── lightning/          # Blink GraphQL client, invoices, and payouts
    │   │   ├── interface.go
    │   │   ├── client.go
    │   │   ├── invoice.go
    │   │   └── payment.go
    │   └── mpesa/              # Safaricom Daraja REST clients and authentication
    │       ├── interface.go
    │       ├── client.go
    │       ├── c2b.go
    │       └── b2c.go
    ├── service/                # Core KES ⇄ Satoshis business logic & pipeline
    │   └── payment_service.go
    ├── models/                 # Canonical schemas, state machine & audit models
    │   └── payment.go
    ├── repository/             # In-memory/Postgres persistence & reconciliation
    │   └── payment_repository.go
    ├── handlers/               # HTTP webhooks & admin controllers
    │   ├── mpesa_handler.go
    │   ├── lightning_handler.go
    │   └── payment_handler.go
    └── tests/                  # Mocked unit & integration test suite
        ├── payment_test.go
        └── lightning_test.go
```
