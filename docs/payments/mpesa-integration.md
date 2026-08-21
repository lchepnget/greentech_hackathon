# Safaricom Daraja M-Pesa Integration

## 1. Authentication (OAuth 2.0)

GreenTech exchanges `MPESA_CONSUMER_KEY` and `MPESA_CONSUMER_SECRET` for a dynamic Bearer Access Token:
- **Sandbox URL:** `https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials`
- **Production URL:** `https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials`

---

## 2. Customer-to-Business (C2B) Flow

Hotels purchase BSF products via Paybill / Till (`600990`).

1. **URL Registration (`POST /api/mpesa/register-urls`)**: Registers `ConfirmationURL` and `ValidationURL`.
2. **Validation Callback (`POST /api/mpesa/c2b-validation`)**: Validates transaction before funds clear. Returns `{"ResultCode":0,"ResultDesc":"Accepted"}`.
3. **Confirmation Callback (`POST /api/mpesa/c2b-confirmation`)**: Safaricom clears funds.
   - Idempotency guard checks `TransID`.
   - Records transaction in `RECEIVED` state.
   - Spawns asynchronous goroutine for live Blink conversion to Satoshis.
   - Records `C2B_CONFIRMED` and `CONVERSION_COMPLETED` audit events.

---

## 3. Business-to-Customer (B2C) Supplier Disbursements

Used for programmatic fiat disbursements directly to waste collectors' M-Pesa numbers.

1. **Trigger Disbursement (`POST /api/mpesa/trigger-b2c`)**: Dispatches B2C request to Daraja.
2. **Result Webhook (`POST /api/mpesa/b2c-result`)**: Authoritative payout completion hook (`ResultCode == 0` marks `SETTLED`).
3. **Timeout Webhook (`POST /api/mpesa/b2c-timeout`)**: Notification that payout timed out. Marks state `TIMED_OUT` without double-settling.
