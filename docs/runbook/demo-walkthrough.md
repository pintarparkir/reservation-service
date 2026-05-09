# Demo Walkthrough — reservation-service

End-to-end runbook for the assessment review.

## Setup (~30 s)

```bash
# 1. Bring up shared infra (postgres, redis, rabbitmq, otel)
cd ../infra && docker compose up -d

# 2. Apply schema + seed 250 spots
cd ../reservation-service
cp configs/.env.example configs/.env
make migrate-up
psql "postgres://postgres:postgres@localhost:5432/reservation_service?sslmode=disable" \
  -f data/seed.sql

# 3. Run
make run
```

Health:
```bash
curl -s http://localhost:8081/healthz
# → {"status":"ok"}
```

## Dev JWT helper

`SUPER_APP_JWT_PUBLIC_KEY_PEM` is empty by default → signature verification is skipped.
Payload is still parsed.

```bash
PAYLOAD=$(printf '{"sub":"super-user-001","phone":"+628123456789","exp":9999999999}' | base64)
TOKEN="eyJhbGciOiJSUzI1NiJ9.${PAYLOAD}.devsig"
```

## Scenario 1 — Happy path (60 s)

```bash
# 1. Check availability
curl -s "http://localhost:8081/v1/availability?type=CAR" \
  -H "Authorization: Bearer $TOKEN" | jq .
# → {"available_count":150,"by_floor":[{"floor":1,"count":30},...]}

# 2. Create reservation (system-assigned)
IDEM=$(uuidgen)
RESV=$(curl -s -X POST http://localhost:8081/v1/reservations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Idempotency-Key: $IDEM" \
  -H "Content-Type: application/json" \
  -d '{"vehicle_type":"CAR","mode":"SYSTEM_ASSIGNED"}')
echo "$RESV" | jq .
RESV_ID=$(echo "$RESV" | jq -r .id)

# 3. Confirm — flips PENDING → CONFIRMED
curl -s -X POST "http://localhost:8081/v1/reservations/$RESV_ID/confirm" \
  -H "Authorization: Bearer $TOKEN" | jq .

# 4. Check-in (Monas-ish coords, inside default geofence)
curl -s -X POST "http://localhost:8081/v1/reservations/$RESV_ID/check-in" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"latitude":-6.2088,"longitude":106.8456}' | jq .

# 5. Check-out
curl -s -X POST "http://localhost:8081/v1/reservations/$RESV_ID/check-out" \
  -H "Authorization: Bearer $TOKEN" | jq .

# 6. Inspect final state
curl -s "http://localhost:8081/v1/reservations/$RESV_ID" -H "Authorization: Bearer $TOKEN" | jq .
# → state=COMPLETED, checked_out_at set, version=4
```

## Scenario 2 — Double-book prevention (45 s)

```bash
SPOT="F1-C-001"

# Driver A grabs the spot
RA=$(curl -s -X POST http://localhost:8081/v1/reservations \
  -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $(uuidgen)" \
  -d "{\"vehicle_type\":\"CAR\",\"mode\":\"USER_SELECTED\",\"preferred_spot_id\":\"$SPOT\"}")
RID=$(echo "$RA" | jq -r .id)
curl -s -X POST "http://localhost:8081/v1/reservations/$RID/confirm" \
  -H "Authorization: Bearer $TOKEN" > /dev/null

# Driver B tries the same spot — must fail with 404 (spot is no longer AVAILABLE)
HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8081/v1/reservations \
  -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $(uuidgen)" \
  -d "{\"vehicle_type\":\"CAR\",\"mode\":\"USER_SELECTED\",\"preferred_spot_id\":\"$SPOT\"}")
echo "Driver B got HTTP $HTTP"
```

**Talking points:**
- `Assign` runs `SELECT ... FOR UPDATE` on the spot row first — the lock prevents two
  drivers from each thinking the spot is theirs.
- The EXCLUDE constraint (`tstzrange` overlap on `spot_id`) is the authoritative
  guard if app-level locks somehow leak.
- Check `data/migrations/001_init.up.sql` line 41-44.

## Scenario 3 — Idempotency replay (30 s)

```bash
IDEM=$(uuidgen)

R1=$(curl -s -X POST http://localhost:8081/v1/reservations \
  -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $IDEM" \
  -d '{"vehicle_type":"CAR","mode":"SYSTEM_ASSIGNED"}')
echo "First:  $(echo $R1 | jq -r .id)"

R2=$(curl -s -X POST http://localhost:8081/v1/reservations \
  -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $IDEM" \
  -d '{"vehicle_type":"CAR","mode":"SYSTEM_ASSIGNED"}')
echo "Replay: $(echo $R2 | jq -r .id)"
# → identical id, no second row created
```

The replay is caught at the row level (`uq_reservation_idem` UNIQUE on `idempotency_key`).
The gRPC interceptor caches at the wire level too — both layers protect against double-booking on retry.

## Scenario 4 — Geofence violation (30 s)

```bash
# Coords ~5 km away from default building centre
curl -s -X POST "http://localhost:8081/v1/reservations/$RESV_ID/check-in" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"latitude":-6.3000,"longitude":106.9000}' -w "\nHTTP:%{http_code}\n"
# → HTTP 400 with {"error":"GEOFENCE_VIOLATION", ...}

# Soft-fail with gps_unavailable=true
curl -s -X POST "http://localhost:8081/v1/reservations/$RESV_ID/check-in" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"latitude":0,"longitude":0,"gps_unavailable":true}' | jq .state
# → "ACTIVE" (soft-passed; check the audit logs for geofence_skipped=true)
```

## Scenario 5 — No-show expiry (set hold to 30 s for demo)

```bash
HOLD_DURATION=30s make run    # restart with short hold

# Confirm but don't check-in; wait > 30 s + 30 s (worker tick) = ~60 s
RID=$(... create + confirm flow ...)
sleep 65
curl -s "http://localhost:8081/v1/reservations/$RID" -H "Authorization: Bearer $TOKEN" | jq .state
# → "EXPIRED"
```

Watch the worker logs:
```
[noshow expirer: flipped reservations] count=1
```

The corresponding `reservation.expired.v1` event lands in RabbitMQ exchange
`parkirpintar.events`. Verify in the management UI at http://localhost:15672
(guest/guest).

## Scenario 6 — Outbox → RabbitMQ (visual)

After any state change, the publisher worker drains `outbox_event` rows and
publishes to RabbitMQ within 1 s. To watch:

```bash
# In another terminal: bind a temporary queue to the exchange
rabbitmqadmin declare queue name=demo.tap durable=false auto_delete=true
rabbitmqadmin declare binding source=parkirpintar.events destination=demo.tap routing_key="#"
rabbitmqadmin get queue=demo.tap requeue=true
```

You'll see one message per state change with the routing key
(`reservation.created.v1`, `.confirmed.v1`, `.checked_in.v1`, `.checked_out.v1`).

## Cleanup

```bash
# Stop the service (Ctrl-C)
cd ../infra && docker compose down -v
```

Total demo time: ~5 min for all six scenarios.
