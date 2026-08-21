# Bitcoin / Lightning Integration & Custody Model

## 1. Provider Choice: Blink GraphQL API

GreenTech selected **Blink** (Galoy-based backend) for its Bitcoin & Lightning rail.

### Rationale:
- **Zero Node-Ops Overhead**: No need to manage Lightning channels, rebalancing, liquidity, or routing infrastructure for the hackathon MVP.
- **Native Lightning Addresses**: Native support for micro-payouts to addresses like `collector@blink.sv`.
- **Live Real-time Price Oracle**: High-precision `realtimePrice(currency: "KES")` query for dynamic fiat-to-Sats conversion.

---

## 2. Custody Trade-off & Risk Profile

- **Hosted Custody**: Blink holds the private keys; GreenTech holds an API Key.
- **Counterparty Exposure**: Sats balances held in the connected Blink wallet are subject to Blink's custodial service.
- **Dust Limit**: Payouts enforce a minimum limit (`MIN_PAYOUT_SATS = 10`) to prevent sub-dust failure modes.
- **Invoice Expiry**: All BOLT11 invoices carry an explicit expiration timestamp (1 hour) and are automatically swept by an in-process background worker.
