# reservation-service

[![Security](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=security_rating&token=a282c8154601e67376f352be6bb9774cf89715f4)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Reliability](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=reliability_rating&token=a282c8154601e67376f352be6bb9774cf89715f4)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=sqale_rating&token=a282c8154601e67376f352be6bb9774cf89715f4)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)
[![Duplications](https://sonarcloud.io/api/project_badges/measure?project=pintarparkir_reservation-service&metric=duplicated_lines_density&token=a282c8154601e67376f352be6bb9774cf89715f4)](https://sonarcloud.io/summary/new_code?id=pintarparkir_reservation-service)

**Cloud Run:** `https://reservation-service-725nddkmwq-as.a.run.app`

## Architecture Overview

![Architecture](docs/PintarParkir.architecture.svg)

## E2E Flow

![Flow Diagram](docs/flow.diagram.svg)

## Sequence Diagrams

![E2E Sequence Diagram](docs/sequence-diagram.png)

---



> **Purpose:** Core domain — manages spot inventory, reservation lifecycle, geofence validation, and no-show expiration.  
> **Author:** Farid Triwicaksono · **Last Updated:** 2026-05-21

## Project Overview

**ParkirPintar** is a backend mini-app for smart parking within a super-app. It handles:
- Availability queries (spots per floor, per vehicle type)
- Reservation creation (system-assigned or user-selected spots)
- Reservation state transitions (confirm, cancel, check-in, check-out)
- Geofence validation (GPS-based check-in)
- No-show expiration (automatic after 1 hour hold)
- Event publishing (outbox pattern → RabbitMQ)

Five services: **user** (driver profile), **reservation** (this service), **billing** (invoices), **payment** (QRIS), **notification** (SMS).

## Service Scope

**Owns:**
- Spot inventory (150 cars + 250 motorcycles, pre-seeded)
- Reservation state machine (PENDING → CONFIRMED → ACTIVE → COMPLETED)
- Spot assignment logic (system-assigned or user-selected)
- Geofence validation (100-150m radius from building)
- Hold timer (1 hour default)
- No-show expiration worker
- Outbox event publishing (reservation.*.v1 events)

**Does NOT own:**
- Driver identity (delegated to super-app JWT)
- Invoice creation (billing-service owns)
- Payment processing (payment-service owns)
- SMS dispatch (notification-service owns)

**Key invariants:**
- One active reservation per driver (anti-hoarding)
- Double-book prevention via PostgreSQL EXCLUDE constraint + Redis lock
- Idempotency via `idempotency_key` table
- Strong consistency on spot inventory

## At a Glance

| Aspect | Details |
|--------|---------|
| **REST Port** | 8081 (mini-app) |
| **gRPC Port** | 9090 (s2s) |
| **Database** | PostgreSQL 16 (reservation, spot, outbox_event) |
| **Cache** | Redis 7 (availability cache, spot locks) |
| **Message Queue** | RabbitMQ 3.13 (event publishing) |
| **External APIs** | user-service (gRPC), billing-service (gRPC) |

## Tech Stack

- **Language:** Go 1.25
- **Web Framework:** Gin (REST) + gRPC
- **Database:** PostgreSQL 16 + sqlx
- **Cache:** Redis 7 (go-redis v8)
- **Message Queue:** RabbitMQ 3.13 (amqp091-go)
- **Logging:** Zap + Lumberjack
- **Observability:** OpenTelemetry (OTLP/gRPC)
- **Testing:** testify/mock, table-driven tests
- **Locking:** Redis SETNX + Lua release script

## Architecture

### High-Level Design
See [`docs/architecture/high-level-design/02-reservation-service.md`](../docs/architecture/high-level-design/02-reservation-service.md) for:
- Service responsibilities and boundaries
- Sync vs async communication patterns
- Failure modes and resilience strategies

### Low-Level Design
See [`docs/architecture/low-level-design/02-reservation-service-lld.md`](../docs/architecture/low-level-design/02-reservation-service-lld.md) for:
- Layer cake (model → usecase → repository → handler)
- Transaction boundaries and outbox pattern
- Key algorithms (spot assignment, geofence check)

### Entity Relationship Diagram
See [`docs/architecture/erd/02-reservation-service.md`](../docs/architecture/erd/02-reservation-service.md) for:
- Table schema (reservation, spot, outbox_event, idempotency_key)
- EXCLUDE constraint for double-book prevention
- Critical indexes

![ParkirPintar ERD](../user-service/ERD.jpg)

## API Reference

### REST Endpoints (mini-app, all require `Authorization: Bearer <jwt>`)

| Method | Path | Description | Idempotent |
|--------|------|-------------|-----------|
| GET | `/v1/availability?type=CAR` | List available spots by floor | Yes |
| POST | `/v1/reservations` | Create reservation (system/user mode) | Yes (via Idempotency-Key) |
| GET | `/v1/reservations/{id}` | Get reservation details | Yes |
| POST | `/v1/reservations/{id}/confirm` | PENDING → CONFIRMED | Yes |
| POST | `/v1/reservations/{id}/cancel` | Cancel (any state) | Yes |
| POST | `/v1/reservations/{id}/check-in` | CONFIRMED → ACTIVE (geofence check) | Yes |
| POST | `/v1/reservations/{id}/check-out` | ACTIVE → COMPLETED | Yes |

### gRPC Services (s2s, internal only)

| RPC | Input | Output | Purpose |
|-----|-------|--------|---------|
| CreateReservation | CreateReservationRequest | Reservation | Idempotent create |
| ConfirmReservation | ConfirmReservationRequest | Reservation | State transition |
| CancelReservation | CancelReservationRequest | CancelReservationResponse | State transition |
| CheckIn | CheckInRequest | Reservation | Geofence + state |
| CheckOut | CheckOutRequest | CheckOutResponse | Close session |
| GetReservation | GetReservationRequest | Reservation | Read-only |

### RabbitMQ Events (published via outbox)

| Event | Trigger | Payload |
|-------|---------|---------|
| `reservation.created.v1` | Create succeeds | reservation_id, driver_id, spot_id, vehicle_type, msisdn (optional) |
| `reservation.confirmed.v1` | Confirm succeeds | reservation_id, driver_id, confirmed_at, msisdn (optional) |
| `reservation.cancelled.v1` | Cancel succeeds | reservation_id, driver_id, reason, cancelled_at, msisdn (optional) |
| `reservation.expired.v1` | No-show worker fires | reservation_id, driver_id, expired_at |
| `reservation.checked_out.v1` | Check-out succeeds | reservation_id, driver_id, confirmed_at, checked_in_at, checked_out_at |

## Sample Environment

```bash
# ── App ─────────────────────────────────────────────────────────────────────
APP_NAME=reservation-service
APP_ENV=local
APP_PORT=8081        # REST port (mini app)
GRPC_PORT=9090       # gRPC port (s2s)

# ── Postgres ────────────────────────────────────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_NAME=reservation_service
DB_MAX_OPEN=25
DB_MAX_IDLE=10

# ── Redis (availability cache + spot lock) ──────────────────────────────────
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=2
REDIS_APP_CONFIG=reservation-service

# ── RabbitMQ (event publishing via outbox) ──────────────────────────────────
RABBIT_URL=amqp://guest:guest@localhost:5672/
RABBIT_EXCHANGE=parkirpintar.events

# ── s2s gRPC clients ────────────────────────────────────────────────────────
USER_GRPC_ADDR=localhost:9094
BILLING_GRPC_ADDR=localhost:9091

# ── Observability ────────────────────────────────────────────────────────────
OTLP_ENDPOINT=localhost:4317

# ── JWT verification ─────────────────────────────────────────────────────────
SUPER_APP_JWT_PUBLIC_KEY_PEM=<paste-public-key-here>

# ── Reservation behaviour knobs ─────────────────────────────────────────────
HOLD_DURATION_MINUTES=60
GEOFENCE_RADIUS_METERS=150
CHECK_IN_BUILDING_LAT=-6.2088
CHECK_IN_BUILDING_LNG=106.8456
```

See `configs/.env.example` for full reference.

## Getting Started

### Prerequisites
- Docker 24+ & Docker Compose v2
- Go 1.25+ (for local development)
- `buf` CLI (for proto regeneration)

### Local Development

```bash
# 1. Clone and setup
git clone <repo> && cd <repo>
cd reservation-service
cp configs/.env.example configs/.env

# 2. Start shared infra (see https://github.com/pintarparkir/infra)
cd ../infra && podman compose up -d

# 3. Run migrations
cd ../reservation-service
make migrate-up

# 4. Seed spots (150 cars + 250 motorcycles)
make seed

# 5. Run the service
make run

# 6. Verify health
curl http://localhost:8081/healthz
```

## Testing

### Unit Tests (no infra)
```bash
make test-unit
```
Covers: usecase logic, pricing engine, state transitions, geofence math.

### Integration Tests (requires postgres/redis/rabbitmq)
```bash
make test-integration
```
Covers: repository layer, outbox pattern, event publishing, double-book prevention.

### All Tests
```bash
make test
```

### Coverage
```bash
go test -coverprofile=cov.out ./...
go tool cover -html=cov.out
```
Target: usecase ≥80%, repository ≥60%.

## Debugging

### Logs
```bash
LOG_LEVEL=debug make run
```
Logs are JSON-formatted with trace_id, span_id, request_id.

### Database
```bash
psql postgresql://postgres:postgres@localhost:5432/reservation_service

# View schema
\dt reservation.*

# Check EXCLUDE constraint
\d reservation
```

### Redis
```bash
redis-cli

# Inspect spot locks
KEYS lock:spot:*

# Check availability cache
KEYS availability:*
```

### RabbitMQ
- **Management UI:** http://localhost:15672 (guest/guest)
- **View exchange:** parkirpintar.events
- **View queues:** reservation.* queues

### gRPC
```bash
# Test gRPC health
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## Operations

### Health Checks
```bash
# REST
curl http://localhost:8081/healthz

# gRPC
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

### Migrations
```bash
make migrate-up      # Apply all pending migrations
make migrate-down    # Rollback one migration
```

### Outbox Publisher
Background worker publishes unsent outbox events to RabbitMQ every 5 seconds. Check logs for `outbox: published` messages.

### No-Show Expirer
Background worker expires reservations older than 1 hour (HOLD_DURATION_MINUTES) every 5 minutes. Emits `reservation.expired.v1` event.

## Security Notes

- **Secrets:** Never commit `.env` files. Use Secret Manager in production.
- **PII:** Phone numbers encrypted at rest via pgcrypto.
- **JWT:** Verified by gateway; service trusts `X-Driver-Id` header.
- **SQL:** All queries parameterized (sqlx prevents injection).
- **Idempotency:** Scoped per (driver_id, idempotency_key) to prevent cross-driver replay.

## Business Flow Logic

### 1. Reservation Flow (Create & Hold Spot)

```mermaid
sequenceDiagram
    autonumber
    actor Driver as 👤 Driver
    participant MiniApp as 📱 Mini-App
    participant ResSvc as Reservation Service
    participant Redis as Redis Lock
    participant DB as Postgres DB
    participant RMQ as RabbitMQ
    participant Billing as Billing Service
    
    Note over Driver,Billing: Scenario: User selects spot or requests system assignment
    
    Driver->>MiniApp: Tap "Reserve Spot" (F2-C-014 or auto-assign)
    MiniApp->>ResSvc: POST /v1/reservations<br/>{spot_id, vehicle_type}
    Note right of MiniApp: Headers: Authorization, Idempotency-Key
    
    activate ResSvc
    ResSvc->>ResSvc: Validate JWT (extract driver_id)
    
    %% Idempotency check
    ResSvc->>DB: SELECT * FROM idempotency_key WHERE key = ?
    alt Key exists (replay)
        DB-->>ResSvc: Found
        ResSvc-->>MiniApp: 200 OK (cached response)
        deactivate ResSvc
        return
    end
    
    %% Double-book prevention: Redis optimistic lock
    ResSvc->>Redis: SETNX lock:spot:F2-C-014 30
    alt Lock acquired
        Redis-->>ResSvc: true
        
        ResSvc->>DB: BEGIN TRANSACTION
        
        %% EXCLUDE constraint ensures no overlapping reservations
        ResSvc->>DB: INSERT INTO reservation (<br/>state='PENDING',<br/>hold_window=[now, now+60min])
        Note right of DB: EXCLUDE USING gist<br/>(spot_id WITH =,<br/>hold_window WITH &&)<br/>WHERE state IN (PENDING,CONFIRMED,ACTIVE)
        
        alt Success
            DB-->>ResSvc: reservation_id = uuid
            
            %% Outbox pattern
            ResSvc->>DB: INSERT INTO outbox_event(<br/>topic='reservation.created.v1',<br/>payload={...})
            
            ResSvc->>DB: INSERT INTO idempotency_key (key, response)
            ResSvc->>DB: COMMIT
            
            ResSvc->>Redis: DEL lock:spot:F2-C-014
            ResSvc-->>MiniApp: 201 Created {reservation_id, spot_id, hold_until}
            
            %% Background: Outbox Publisher
            par Outbox Worker (async)
                ResSvc->>DB: SELECT * FROM outbox_event WHERE published_at IS NULL
                ResSvc->>RMQ: PUBLISH reservation.created.v1
                ResSvc->>DB: UPDATE outbox_event SET published_at = NOW()
                
                RMQ-->Billing: CONSUME reservation.created.v1
                Billing->>Billing: OpenInvoice(idem_key=reservation_id)
            end
            
        else Constraint violation (double-book)
            DB-->>ResSvc: ERROR: conflicting key
            ResSvc->>DB: ROLLBACK
            ResSvc->>Redis: DEL lock:spot:F2-C-014
            ResSvc-->>MiniApp: 409 Conflict "Spot already reserved"
        end
        
    else Lock held (concurrent request)
        Redis-->>ResSvc: false
        ResSvc-->>MiniApp: 409 Conflict "Try again"
    end
    
    deactivate ResSvc
```

### 2. Check-in Flow (Geofence Validation)

```mermaid
sequenceDiagram
    autonumber
    actor Driver as 👤 Driver
    participant MiniApp as 📱 Mini-App
    participant ResSvc as Reservation Service
    participant DB as Postgres DB
    
    Note over Driver,DB: Scenario: Driver arrives at building
    
    Driver->>MiniApp: Tap "Check-In"
    MiniApp->>MiniApp: Get GPS coordinates
    MiniApp->>ResSvc: POST /v1/reservations/{id}/check-in<br/>{lat: -6.2001, lon: 106.8170}
    
    activate ResSvc
    ResSvc->>DB: SELECT * FROM reservation WHERE id = ? FOR UPDATE
    DB-->>ResSvc: state=CONFIRMED
    
    alt State != CONFIRMED
        ResSvc-->>MiniApp: 400 Bad Request "Confirm reservation first"
        deactivate ResSvc
        return
    end
    
    %% Geofence calculation: Haversine formula
    ResSvc->>ResSvc: Haversine distance(<br/>user_lat, user_lon,<br/>building_lat, building_lon)
    Note right of ResSvc: Distance = 45m
    
    alt distance <= 150m (configurable)
        ResSvc->>DB: BEGIN
        ResSvc->>DB: UPDATE reservation SET<br/>state='ACTIVE',<br/>checked_in_at=NOW()
        ResSvc->>DB: COMMIT
        
        ResSvc-->>MiniApp: 200 OK {state: ACTIVE, checked_in_at}
        deactivate ResSvc
        
    else distance > 150m
        ResSvc-->>MiniApp: 409 Conflict<br/>"Too far from building ({distance}m)"
        MiniApp->>MiniApp: Show map with building location
    end
```

### 3. Check-out Flow (Trigger Billing)

```mermaid
sequenceDiagram
    autonumber
    actor Driver as 👤 Driver
    participant MiniApp as 📱 Mini-App
    participant ResSvc as Reservation Service
    participant DB as Postgres DB
    participant RMQ as RabbitMQ
    participant Billing as Billing Service
    participant Pricing as Pricing Engine
    participant Notif as Notification Service
    
    Note over Driver,Notif: Scenario: Driver leaves building, triggers invoice
    
    Driver->>MiniApp: Tap "Check-Out"
    MiniApp->>ResSvc: POST /v1/reservations/{id}/check-out
    
    activate ResSvc
    ResSvc->>DB: BEGIN
    ResSvc->>DB: UPDATE reservation SET<br/>state='COMPLETED',<br/>checked_out_at=NOW()<br/>WHERE state='ACTIVE'
    ResSvc->>DB: INSERT INTO outbox_event(<br/>topic='reservation.checked_out.v1')
    ResSvc->>DB: COMMIT
    
    ResSvc-->>MiniApp: 200 OK {state: COMPLETED}
    deactivate ResSvc
    
    %% Async: Billing consumer
    par Billing Event Consumer
        ResSvc->>RMQ: PUBLISH reservation.checked_out.v1
        RMQ-->Billing: CONSUME
        
        activate Billing
        Billing->>DB: SELECT * FROM invoice WHERE reservation_id = ?
        
        %% Pricing engine
        Billing->>Pricing: CalculatePricing({<br/>checked_in: 08:00,<br/>checked_out: 10:30,<br/>vehicle_type: CAR})
        activate Pricing
        Pricing->>Pricing: BOOKING: 2,000<br/>HOURLY (2.5h): 12,500<br/>TOTAL: 14,500
        Pricing-->>Billing: {lines, total: 14500}
        deactivate Pricing
        
        Billing->>DB: BEGIN
        Billing->>DB: UPDATE invoice SET total=14500, status='CLOSED'
        Billing->>DB: INSERT INTO invoice_line (kind, amount)
        Billing->>DB: INSERT INTO outbox_event('billing.invoice.closed.v1')
        Billing->>DB: COMMIT
        
        Billing->>RMQ: PUBLISH billing.invoice.closed.v1
        RMQ-->Notif: CONSUME → Send SMS receipt
        deactivate Billing
    end
```

### 4. Cancellation Flow

```mermaid
sequenceDiagram
    autonumber
    actor Driver as 👤 Driver
    participant MiniApp as 📱 Mini-App
    participant ResSvc as Reservation Service
    participant DB as Postgres DB
    participant RMQ as RabbitMQ
    participant Billing as Billing Service
    
    Note over Driver,Billing: Scenario: User cancels before check-in
    
    Driver->>MiniApp: Tap "Cancel Reservation"
    MiniApp->>ResSvc: POST /v1/reservations/{id}/cancel<br/>{reason: "schedule_change"}
    
    activate ResSvc
    ResSvc->>DB: BEGIN
    ResSvc->>DB: SELECT state FROM reservation WHERE id = ? FOR UPDATE
    DB-->>ResSvc: state=CONFIRMED
    
    alt state IN (PENDING, CONFIRMED)
        ResSvc->>DB: UPDATE reservation SET<br/>state='CANCELLED',<br/>cancelled_at=NOW()
        ResSvc->>DB: INSERT INTO outbox_event('reservation.cancelled.v1')
        ResSvc->>DB: COMMIT
        
        ResSvc-->>MiniApp: 200 OK {state: CANCELLED}
        deactivate ResSvc
        
        %% Async: Billing applies cancellation fee
        par Billing Consumer
            ResSvc->>RMQ: PUBLISH reservation.cancelled.v1
            RMQ-->Billing: CONSUME
            
            activate Billing
            Billing->>Billing: CalculateCancelFee(<br/>cancelled_at vs hold_expiry)
            
            alt Cancelled within 30min of expiry
                Billing->>DB: INSERT invoice_line(kind='CANCELLATION', amount=2000)
                Note right of Billing: Booking fee retained
            else Early cancellation
                Billing->>DB: No fee applied
            end
            
            Billing->>DB: UPDATE invoice SET status='CLOSED'
            Billing->>DB: COMMIT
            deactivate Billing
        end
        
    else state NOT IN (PENDING, CONFIRMED)
        ResSvc->>DB: ROLLBACK
        ResSvc-->>MiniApp: 400 Bad Request "Cannot cancel from current state"
    end
```

### Reservation State Machine

```
        ┌────────────┐
        │  PENDING   │ ← Create reservation
        └─────┬──────┘
              │ confirm (within hold time)
              ▼
        ┌────────────┐
        │ CONFIRMED  │
        └─────┬──────┘
              │         ┌────────────┐
              │────────▶│  EXPIRED   │ (no-show worker)
              │         └────────────┘
              │ check-in (geofence valid)
              ▼
        ┌────────────┐
        │   ACTIVE   │
        └─────┬──────┘
              │ check-out
              ▼
        ┌────────────┐
        │ COMPLETED  │
        └────────────┘

(any state) ──▶ CANCELLED (user/system cancel before COMPLETED)
```

---

## Related Documentation

- **Architecture Overview:** [`../docs/README.md`](../docs/README.md)
- **Shared Infra Docs:** [`infra`](https://github.com/pintarparkir/infra)
- **API Documentation:** [`../docs/api-documentation/00-overview.md`](../docs/api-documentation/00-overview.md)
- **Implementation Backlog:** [`../docs/implementation-todo/00-backlog.md`](../docs/implementation-todo/00-backlog.md)

---

_For questions or issues, refer to the troubleshooting section in the main README or open an issue on the repo._
