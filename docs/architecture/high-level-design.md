# High-Level Design — reservation-service

The most complex service: spot inventory + reservation lifecycle + billing trigger
+ event publishing.

## Position in the system

```
                Mini App
                   │ HTTPS + JWT
                   ▼
         ┌─────────────────────┐
         │ reservation-service │
         │   REST :8081        │
         │   gRPC :9090        │
         └─┬───────┬─────┬─────┘
           │       │     │
   gRPC    │       │     │ AMQP (outbox publisher)
  ┌────────▼─┐   ┌─▼─────▼──┐
  │ billing  │   │ rabbitmq │
  │ service  │   │ exchange │
  └──────────┘   └─┬────────┘
                   │
                   ▼ consumed by
                notification-service
```

## Responsibilities

- **Spot availability** — Redis-cached count by floor + vehicle_type
- **Reservation creation** — `SYSTEM_ASSIGNED` (`SELECT ... FOR UPDATE SKIP LOCKED`)
  and `USER_SELECTED` (preferred spot) modes
- **State machine enforcement** — `PENDING → CONFIRMED → ACTIVE → COMPLETED / CANCELLED / EXPIRED`
- **Hold-time enforcement** — 1-hour PENDING grace; no-show worker flips to EXPIRED
- **Geofence on check-in** — Haversine; soft-fail if GPS off
- **Outbox pattern** — at-least-once event delivery to RabbitMQ
- **Billing trigger** — `OpenInvoice` on confirm, `CloseInvoice` on check-out (via gRPC)

## Sequence — happy path

```
Driver        reservation-svc   Redis     Postgres    billing-svc   RabbitMQ
  │── POST /v1/reservations ──▶│
  │                             │── SETNX lock:spot ──▶│
  │                             │◀── OK ────────────│
  │                             │── BEGIN ────────────────▶
  │                             │── INSERT reservation ────▶  (EXCLUDE constraint)
  │                             │── INSERT outbox_event ───▶
  │                             │── COMMIT ─────────────────▶
  │                             │── gRPC OpenInvoice() ────────────────▶
  │                             │◀───────── invoice ────────────────────
  │◀── 201 (state=PENDING) ─────│
  │                             │  outbox poller (loop)
  │                             │  ──── publish reservation.confirmed.v1 ─────────▶
```

## Communication patterns

| Direction               | Protocol  | Notes                                            |
|-------------------------|-----------|--------------------------------------------------|
| Mini app → this service | REST HTTP | JWT verified per-request                         |
| This service → billing  | gRPC      | Idempotency-Key propagated as gRPC metadata      |
| This service → user     | gRPC      | Optional MSISDN lookup if needed                 |
| This service → RabbitMQ | AMQP      | Outbox publisher goroutine reads `outbox_event`  |

## Events emitted

Exchange: `parkirpintar.events` (topic).

| Routing key                  | Trigger                                |
|------------------------------|----------------------------------------|
| `reservation.created.v1`     | Reservation row inserted               |
| `reservation.confirmed.v1`   | PENDING → CONFIRMED                    |
| `reservation.cancelled.v1`   | * → CANCELLED                          |
| `reservation.expired.v1`     | CONFIRMED → EXPIRED (no-show)          |
| `reservation.checked_in.v1`  | CONFIRMED → ACTIVE                     |
| `reservation.checked_out.v1` | ACTIVE → COMPLETED                     |

## Deployment

Cloud Run, **min 1 instance** — this service is on the critical path for booking,
so we accept the cost of one always-warm instance to avoid cold-start latency on
`POST /v1/reservations`.
