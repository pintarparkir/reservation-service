# Low-Level Design — reservation-service

> Status: scaffold. Sections fill in as each feature ships.
> Mirror of user-service's LLD format.

## 1. Package layout

```
reservation-service/
├── api/proto/reservation/v1/   ← gRPC contract
├── cmd/reservation/main.go     ← entry point
├── configs/                    ← .env.example
├── data/                       ← init.sql + golang-migrate migrations
├── internal/reservation/
│   ├── model/                  ← Reservation, Spot, state-machine types
│   ├── repository/             ← interfaces + postgres impl
│   ├── usecase/                ← business logic
│   ├── handler/
│   │   ├── grpc/               ← ReservationService gRPC adapters
│   │   └── http/               ← REST adapters for mini app
│   └── worker/                 ← no-show expirer, outbox publisher
├── pkg/                        ← cross-cutting (logger, otel, redis, jwt, …)
└── docs/                       ← features, runbook, architecture
```

## 2. Critical transaction (pseudo-code)

```go
func (s *Server) CreateReservation(ctx, req) (*pb.Reservation, error) {
    // 1. Idempotency replay
    if cached, ok := s.idem.Get(ctx, key); ok { return cached, nil }

    // 2. Spot assignment
    spotID, err := s.assigner.Assign(ctx, req.VehicleType, req.PreferredSpotID)
    // → SYSTEM_ASSIGNED: SELECT id FROM spot WHERE vehicle_type=$1 AND status='AVAILABLE'
    //                    FOR UPDATE SKIP LOCKED LIMIT 1
    // → USER_SELECTED:   verify status='AVAILABLE' for the requested spot

    // 3. Redis lock (defence-in-depth — actual race is caught by the DB EXCLUDE)
    s.lock.Acquire(ctx, "spot:"+spotID, 30*time.Second)
    defer s.lock.Release(...)

    // 4. Tx: insert reservation + outbox row + call billing
    err := s.db.InTx(ctx, func(tx) error {
        r := s.repo.Insert(ctx, tx, ...)
        s.billing.OpenInvoice(ctx, ...)        // gRPC, idempotent
        s.outbox.Append(ctx, tx, "reservation.confirmed.v1", r.ToEvent())
        return nil
    })

    // 5. Cache idem response
    s.idem.Put(...)
    return r.ToProto(), nil
}
```

## 3. Error mapping (PG / domain → gRPC code)

| Source error                          | gRPC code            | Notes                       |
|---------------------------------------|----------------------|-----------------------------|
| `pgerrcode.ExclusionViolation` (23P01)| `ALREADY_EXISTS`     | Double-book attempt         |
| Idem replay (UNIQUE on idem_key)      | (return cached)      | —                           |
| `redis.Nil` on lock                   | `RESOURCE_EXHAUSTED` | Lock contention; retry-after|
| `domain.ErrInvalidTransition`         | `FAILED_PRECONDITION`| Bad state                   |
| `circuitbreaker.ErrOpen`              | `UNAVAILABLE`        | Dependency down             |

## 4. Background workers

- **No-show expirer** (`worker/noshow_expirer.go`): every 30 s,
  `SELECT id FROM reservation WHERE state='CONFIRMED' AND expires_at < now()
   FOR UPDATE SKIP LOCKED LIMIT 100`. Per row → `EXPIRED` + emit
   `reservation.expired.v1`. `SKIP LOCKED` lets multiple replicas run safely.

- **Outbox publisher** (`worker/outbox_publisher.go`): every 1 s,
  `SELECT * FROM outbox_event WHERE published_at IS NULL ORDER BY id LIMIT 200
   FOR UPDATE SKIP LOCKED`. Publish to RabbitMQ, ack, then `UPDATE published_at`.

## 5. Geofence (check-in)

Haversine distance between request `(lat, lng)` and configured building centre
(`CHECK_IN_BUILDING_LAT`, `CHECK_IN_BUILDING_LNG`). Distance > `GEOFENCE_RADIUS_METERS`
returns 422 *unless* the request also sets `gps_unavailable=true`, in which case we
soft-pass and log `geofence_skipped=true` for audit.
