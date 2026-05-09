# ADR-004: PostgreSQL EXCLUDE constraint as authoritative double-book prevention

**Status:** Accepted  
**Date:** 2026-04-27

## Context
Reservation must guarantee a spot is never assigned to two drivers for overlapping time windows. We have Redis distributed locks, but Redis is not the system of record.

## Decision
The **PostgreSQL `EXCLUDE USING gist` constraint** on `(spot_id, hold_window)` is the authoritative defense. Redis lock is a performance optimization to fail fast and reduce DB write contention, not a correctness mechanism.

```sql
ALTER TABLE reservation
  ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('CONFIRMED','ACTIVE'));
```

## Alternatives Considered
- **Redis lock only**: Vulnerable to clock drift, network partition, lock holder crash. Used in many real outages.
- **App-level SELECT-then-INSERT**: Race condition window between read and write.
- **Serializable isolation level**: Works, but introduces high abort rate under contention; harder to reason about.

## Consequences
- **+** Storage layer enforces invariant — impossible to bypass.
- **+** Conflict surfaces as `pgerrcode.ExclusionViolation` (23P01) cleanly.
- **−** Postgres-specific; if we ever migrate DB engine, must reimplement.
- **−** Range type + GIST has slight write cost (~5-10% vs no constraint), acceptable.

## Trigger to revisit
- Migrating off PostgreSQL (unlikely, high friction).
- Need for cross-region writes (then we partition by spot, single writer per partition).
