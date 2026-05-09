# Feature 05 — No-show expiry

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

A CONFIRMED reservation that is not checked in within 1 hour auto-flips to EXPIRED.
The expiry triggers a cancellation fee (per ADR-005 in billing-service).

**In:**
- Background worker, runs every 30 s.
- Uses `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 100` so multiple replicas can run safely.
- Each expired row gets `state='EXPIRED'` + outbox `reservation.expired.v1`.

**Out:**
- Configurable per-driver hold time (Beyond MVP).
- "About to expire" warning notification (Beyond MVP — could be added by listening to expirer's outbox).

## Worker pseudocode

```go
ticker := time.NewTicker(30 * time.Second)
for range ticker.C {
    db.InTx(ctx, func(tx pgx.Tx) error {
        rows, _ := tx.Query(ctx, `
            SELECT id, version FROM reservation
            WHERE state = 'CONFIRMED' AND expires_at < now()
            FOR UPDATE SKIP LOCKED
            LIMIT 100
        `)
        for rows.Next() {
            var id string; var ver int
            rows.Scan(&id, &ver)
            tx.Exec(ctx, `
                UPDATE reservation SET state='EXPIRED', version=$2+1, updated_at=now()
                WHERE id=$1 AND version=$2
            `, id, ver)
            tx.Exec(ctx, `INSERT INTO outbox_event (...) VALUES (..., 'reservation.expired.v1', $1)`, id)
        }
        return nil
    })
}
```

## Tasks

- [ ] `worker/noshow_expirer.go`
- [ ] Wire into main.go behind a feature flag (so it can be disabled in unit tests)
- [ ] Metric: `reservation_expired_total{outcome=...}`
- [ ] Test: insert CONFIRMED row with `expires_at = now()-1h`, run worker, assert EXPIRED

## Acceptance criteria

- A CONFIRMED row with `expires_at < now()` gets transitioned to EXPIRED within 30 s
  of the worker starting.
- Two replicas of the worker do NOT double-process (one sees the row locked).
- An `outbox_event` row is appended for each expired reservation.
- A reservation that's already CANCELLED or COMPLETED is NEVER touched by the worker.
