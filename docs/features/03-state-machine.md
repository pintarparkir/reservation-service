# Feature 03 — Reservation state machine

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

Enforce the legal state transitions. Every state change increments `version`
(optimistic lock). Invalid transitions return `FAILED_PRECONDITION` / 422.

```
   PENDING ──confirm──▶ CONFIRMED ──check-in──▶ ACTIVE ──check-out──▶ COMPLETED
      │                    │                      │
      └──cancel──┐     ┌───┴──────┐                └──cancel──┐
                ▼     ▼          ▼                            ▼
             CANCELLED        EXPIRED                      CANCELLED
                          (no-show, by worker)
```

## Allowed transitions

| From       | Action     | To         | Side effects                            |
|------------|------------|------------|-----------------------------------------|
| PENDING    | confirm    | CONFIRMED  | gRPC `OpenInvoice` already called on create; emit `reservation.confirmed.v1` |
| PENDING    | cancel     | CANCELLED  | emit `reservation.cancelled.v1` (no fee)|
| CONFIRMED  | check-in   | ACTIVE     | geofence; emit `reservation.checked_in.v1` |
| CONFIRMED  | cancel     | CANCELLED  | apply cancel-fee policy; emit cancel event |
| CONFIRMED  | (worker)   | EXPIRED    | no-show; emit `reservation.expired.v1` |
| ACTIVE     | check-out  | COMPLETED  | gRPC `CloseInvoice`; emit `reservation.checked_out.v1` |
| ACTIVE     | cancel     | CANCELLED  | charge full; emit cancel event          |

Anything else → 422 / `FAILED_PRECONDITION`.

## Tasks

- [ ] `model.allowedTransitions` map keyed on `(from, action)`
- [ ] `usecase` action methods all share `applyTransition(from, action)` helper
- [ ] Optimistic lock: `UPDATE … WHERE version = $expected RETURNING …`
- [ ] Each transition emits exactly one outbox event in the same tx

## Acceptance criteria

- A `confirm` on a CANCELLED reservation returns `FAILED_PRECONDITION`.
- Two concurrent confirms on the same PENDING row: one succeeds (version=2), one
  returns `FAILED_PRECONDITION` (stale version).
- All state changes have an `outbox_event` row written in the same tx.
