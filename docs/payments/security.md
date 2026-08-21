# Webhook Security, Idempotency & Secrets Management

## 1. Secrets Management

- **Zero Hardcoded Credentials**: API Keys, Consumer Secrets, and Initiator Passwords are read solely from environment variables (`.env`).
- **No Credential Logging**: All audit logs, errors, and JSON responses scrub credentials and sensitive payload tokens.

---

## 2. Idempotency Guarantees

Safaricom and Lightning webhook callbacks can be re-delivered multiple times.

- **Unique Constraints**: `UNIQUE(mpesa_trans_id)` and `UNIQUE(payment_hash)`.
- **Deduplication**: When an incoming callback has a known `TransID`, the backend marks the attempt `DUPLICATE_IGNORED` in audit logs and returns the existing transaction without re-executing conversions or disbursements.

---

## 3. Webhook Callback Validation

- Incoming payloads are strictly validated for required fields (`TransID`, `TransAmount`, `MSISDN`).
- State transitions are strictly validated against the finite state machine (`RECEIVED` ➔ `CONVERTING` ➔ `PAYOUT_PENDING` ➔ `SETTLED`).
