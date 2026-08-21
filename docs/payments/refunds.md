# Refunds & Reversals Policy

## 1. Non-Reversible Nature of Lightning & M-Pesa

Bitcoin Lightning payments are cryptographically final. M-Pesa C2B transactions cannot be automatically pulled back by a client.

---

## 2. Authorized Reversal Workflow

When a hotel cancellation or dispute requires a refund:

1. **Admin Authorization**: An authenticated administrator triggers `POST /api/admin/refund` with `{"originalTransactionId": "...", "reason": "..."}`.
2. **Original State Verification**: The backend verifies that the original transaction exists, is in `SETTLED` state, and belongs to an inbound M-Pesa payment.
3. **M-Pesa B2C Reversal**: A programmatic B2C disbursement is sent back to the hotel's original phone number (`PayerMSISDN`).
4. **Audit Trail**: The action is recorded as `REFUND_ISSUED` with links to both the original `transaction_id` and the reversal `mpesa_trans_id`.
