# reservation-service

Core domain — manages spot inventory and the reservation lifecycle. Calls
billing-service over gRPC on confirm; emits domain events to RabbitMQ via the
outbox pattern.

## At a glance

| Surface | Port  | Used by                                  |
|---------|-------|------------------------------------------|
| REST    | 8081  | Mini app (Tencent Mini Program)          |
| gRPC    | 9090  | s2s callers — currently none, ready for future fan-out |

## REST API (mini app)

Base path: `/v1`. All endpoints require `Authorization: Bearer <jwt>`.

| Method | Path                                | Description                          |
|--------|-------------------------------------|--------------------------------------|
| GET    | /v1/availability?type=CAR           | Available spot count by floor        |
| POST   | /v1/reservations                    | Create reservation (SYSTEM/USER mode)|
| GET    | /v1/reservations/{id}               | Get one reservation                  |
| POST   | /v1/reservations/{id}/confirm       | PENDING → CONFIRMED, opens invoice   |
| POST   | /v1/reservations/{id}/cancel        | Cancel (with cancel-fee policy)      |
| POST   | /v1/reservations/{id}/check-in      | CONFIRMED → ACTIVE (geofence check)  |
| POST   | /v1/reservations/{id}/check-out     | ACTIVE → COMPLETED, closes invoice   |

## State machine

```
   PENDING ──confirm──▶ CONFIRMED ──check-in──▶ ACTIVE ──check-out──▶ COMPLETED
      │                    │                      │
      └──cancel──┐         └──cancel/no-show──┐   └──cancel──┐
                 ▼                            ▼              ▼
              CANCELLED                    EXPIRED        CANCELLED
```

## Service dependencies

| Dependency      | Protocol | Purpose                              |
|-----------------|----------|--------------------------------------|
| user-service    | gRPC     | Resolve driver_id from JWT (s2s call)|
| billing-service | gRPC     | `OpenInvoice` on confirm             |
| RabbitMQ        | AMQP     | Publish domain events (outbox)       |
| Redis           | TCP      | Availability cache + spot lock       |
| PostgreSQL      | TCP      | Reservation + spot storage           |

## Run

```bash
# 1. Bring up shared infra
cd ../infra && docker compose up -d

# 2. Run the service
cd ../reservation-service
cp configs/.env.example configs/.env
make migrate-up
make run
```

## Docs

- `docs/architecture/high-level-design.md` — service-scoped HLD
- `docs/architecture/low-level-design.md` — design (TBD as features ship)
- `docs/architecture/erd.md` — owned tables (`spot`, `reservation`, `outbox_event`, …)
- `docs/features/` — one md per feature, with status / scope / tasks / acceptance
- `docs/runbook/` — demo & ops walkthroughs
