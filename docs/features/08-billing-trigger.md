# Feature 08 — Billing trigger

**Status:** 📋 planned
**Owner:** reservation-service (caller); billing-service owns the implementation

## Scope

reservation-service calls billing-service over gRPC at two lifecycle points:

| Trigger             | Billing RPC      | Note                                 |
|---------------------|------------------|--------------------------------------|
| `CreateReservation` | `OpenInvoice`    | inside the same DB tx, idempotency-keyed |
| `CheckOut`          | `CloseInvoice`   | applies pricing engine, emits closed event |

`Cancel` does NOT call billing directly — instead, the `reservation.cancelled.v1`
event drives billing-service to apply the cancel-fee policy. This keeps the
cancel path fast and decoupled.

## gRPC contract (excerpt — full proto lives in billing-service)

```proto
rpc OpenInvoice (OpenInvoiceRequest) returns (Invoice);
rpc CloseInvoice(CloseInvoiceRequest) returns (Invoice);

message OpenInvoiceRequest {
  string reservation_id = 1;
  string driver_id = 2;
}
```

## Idempotency

The `Idempotency-Key` from the inbound request (mini app) is propagated as gRPC
metadata to billing. Billing's own idempotency interceptor de-dupes on its side.

## Failure modes

| Failure                          | reservation-service behaviour              |
|----------------------------------|--------------------------------------------|
| billing returns `UNAVAILABLE`    | tx rolls back; client retries              |
| billing returns `ALREADY_EXISTS` | treat as success (idempotent replay)       |
| billing exceeds 5 s deadline     | tx rolls back; client retries              |
| circuit breaker open             | fail-fast 503; client retries with backoff |

## Tasks

- [ ] `pkg/grpcclient/billing` with circuit breaker + timeout
- [ ] `usecase.Create` calls `billing.OpenInvoice` inside the tx
- [ ] `usecase.CheckOut` calls `billing.CloseInvoice` after state transition
- [ ] OTel propagation across the gRPC call

## Acceptance criteria

- Killing billing-service while a reservation is being created → reservation
  insert is rolled back; no orphan row.
- Replaying the same `Idempotency-Key` → exactly one invoice on the billing side.
