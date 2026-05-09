# Feature 02 — Create reservation

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

Driver creates a reservation in one of two modes. The DB is the source of truth
for assignment correctness; Redis lock is defence-in-depth.

**In:**
- `mode = SYSTEM_ASSIGNED` — pick any AVAILABLE spot for the requested type
- `mode = USER_SELECTED` — driver passes `preferred_spot_id`; spot must be AVAILABLE
- Idempotent on `Idempotency-Key` header
- On success, opens an invoice via gRPC → billing-service
- On success, appends `reservation.created.v1` to outbox

**Out:**
- Pre-booking (T+24 h) — Beyond MVP
- Bulk fleet reservations — Beyond MVP

## API contract

### REST
```
POST /v1/reservations
Headers: Authorization, Idempotency-Key
Body: { "vehicle_type": "CAR", "mode": "SYSTEM_ASSIGNED" }
   or: { "vehicle_type": "CAR", "mode": "USER_SELECTED", "preferred_spot_id": "F2-C-014" }
→ 201 { id, driver_id, spot_id, state:"PENDING", hold_end, version:1 }
```

## Algorithm

```
1. idempotency replay check
2. spot assignment:
   - SYSTEM_ASSIGNED:
       SELECT id FROM spot
       WHERE vehicle_type=$1 AND status='AVAILABLE'
       ORDER BY id LIMIT 1
       FOR UPDATE SKIP LOCKED
   - USER_SELECTED:
       SELECT id FROM spot
       WHERE id=$preferred AND status='AVAILABLE'
       FOR UPDATE
3. Redis: SETNX lock:spot:<id> ttl=30s   (advisory lock)
4. BEGIN tx:
   - INSERT reservation (PENDING, hold_window=[now, now+1h])
     → EXCLUDE constraint catches concurrent insert
   - INSERT outbox_event 'reservation.created.v1'
5. gRPC OpenInvoice() with Idempotency-Key
6. COMMIT
7. release Redis lock
8. cache idempotency response
9. return Reservation
```

## Tasks

- [ ] `usecase.Create` orchestration
- [ ] `SpotRepository.Assign(vt, preferred)` — system + user mode
- [ ] `ReservationRepository.Insert` returning ALREADY_EXISTS on EXCLUDE
- [ ] Redis lock helper in `pkg/lock`
- [ ] Outbox append inside the same tx
- [ ] gRPC + REST handlers
- [ ] Integration test for double-book race (two goroutines, same spot)

## Acceptance criteria

- Two concurrent USER_SELECTED requests for the same spot: one 201, one 409.
- 100 concurrent SYSTEM_ASSIGNED requests for CAR (assume 100 free CAR spots):
  100 distinct `spot_id`s, 100 reservation rows, 0 collisions.
- If `OpenInvoice` returns an error, the reservation tx is rolled back; no orphan row.
- Replaying with the same `Idempotency-Key` returns the original response.
