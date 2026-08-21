# Dual-Rail Reconciliation Engine

## 1. Automated Discrepancy Detection

The reconciliation engine inspects transactions across both rails to detect anomalies:

- **Unmatched Inbound Payments**: C2B confirmations without corresponding outbound product/service fulfillment.
- **Stuck Pending Transactions**: Transactions in `PAYOUT_PENDING` or `CONVERTING` for longer than 15 minutes.
- **Duplicate References**: Any duplicate `TransID` occurrences in the ledger.
- **Totals Calculation**: Total KES received vs. Total Satoshis disbursed.

---

## 2. Admin Reconciliation Endpoint

```http
GET /api/admin/reconciliation
```

### Sample Response:
```json
{
  "unmatchedInboundCount": 0,
  "unmatchedInbound": [],
  "stuckPendingCount": 0,
  "stuckPending": [],
  "duplicateTransIds": [],
  "totalInboundKes": 15000.0,
  "totalOutboundSats": 159500,
  "generatedAt": "2026-08-20T20:40:00Z"
}
```
