# Feature 07 — Outbox publisher

**Status:** ✅ shipped
**Owner:** reservation-service (pattern is reusable)

## Scope

Domain events emitted by `usecase.*` are written to `outbox_event` in the same
transaction as the state change. A background worker reads unpublished rows and
publishes them to RabbitMQ. Guarantees at-least-once delivery; consumers are
expected to be idempotent.

## Why outbox

Without outbox, the publish call has to live either:
- *Inside* the transaction → publish-on-rollback risk; couples DB to broker.
- *After commit* → crash between commit + publish loses the event.

The outbox table makes "the event happened" durable in the same atomic unit as
"the state changed", and decouples publish from the request path.

## Worker pseudocode

```go
ticker := time.NewTicker(time.Second)
for range ticker.C {
    db.InTx(ctx, func(tx) error {
        rows, _ := tx.Query(ctx, `
            SELECT id, event_type, payload FROM outbox_event
            WHERE published_at IS NULL
            ORDER BY id
            FOR UPDATE SKIP LOCKED
            LIMIT 200
        `)
        for rows.Next() {
            var id int64; var key string; var payload []byte
            rows.Scan(&id, &key, &payload)
            err := rabbit.Publish(ctx, "parkirpintar.events", key, payload)
            if err != nil { return err }                // tx aborts; retry next tick
            tx.Exec(ctx, `UPDATE outbox_event SET published_at=now() WHERE id=$1`, id)
        }
        return nil
    })
}
```

## Tasks

- [ ] `worker/outbox_publisher.go`
- [ ] RabbitMQ connection pool in `pkg/rabbit`
- [ ] Topic exchange declared on connect (`parkirpintar.events`, durable)
- [ ] Metric: `outbox_published_total{event_type=...}` and `outbox_lag_seconds`
- [ ] Test: insert outbox row, run one tick, assert RMQ delivery + published_at set

## Acceptance criteria

- Killing the publisher mid-loop and restarting does NOT lose any events.
- Killing the publisher mid-loop and restarting MAY republish in-flight events
  (consumers must be idempotent — documented in ADR-001 in `infra/`).
- Metric `outbox_lag_seconds` reflects oldest unpublished row's age.
