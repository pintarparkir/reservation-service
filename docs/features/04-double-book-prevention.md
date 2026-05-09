# Feature 04 — Double-book prevention

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

A spot may not have two overlapping reservations in `(CONFIRMED, ACTIVE)` states.
Enforced at three layers, with the DB constraint as the authoritative one.

## Layered defence

1. **Application — Redis lock**
   - `SETNX lock:spot:<id> ttl=30s`
   - Fast contention signal; lets us return 429 quickly without a DB round-trip.
   - Lossy: clock skew or holder crash can leak the lock. That's fine — layer 3 catches it.

2. **DB — `FOR UPDATE SKIP LOCKED`**
   - Spot row is locked at assignment time.
   - Concurrent SELECTs skip the locked row → see "no spots available".

3. **DB — EXCLUDE constraint** *(authoritative)*
   ```sql
   ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
     EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
     WHERE (state IN ('CONFIRMED','ACTIVE'));
   ```
   - Postgres rejects any insert/update that would create overlap on the same spot.
   - Catches every race the application layers might miss.

See `adr/001-pg-exclusion-constraint.md` for the full rationale.

## Tasks

- [ ] `btree_gist` extension in init.sql
- [ ] EXCLUDE constraint applied via migration
- [ ] Repository maps `pgerrcode.ExclusionViolation` → `apperror.ConflictError`
- [ ] Redis lock helper with safe Lua-script release (compare-and-delete)
- [ ] Integration test: 50 goroutines hammering the same `preferred_spot_id`

## Acceptance criteria

- 50 concurrent USER_SELECTED requests for the same spot: exactly 1 success, 49 conflicts.
- Killing Redis mid-flight does not allow double-book (DB still rejects).
- The conflict surfaces to the client as HTTP 409 / gRPC `ALREADY_EXISTS`.
