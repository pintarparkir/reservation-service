# Feature 01 — Spot availability

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

Return current available spot count, broken down by floor, for a given vehicle type.
Read-heavy; cached in Redis with short TTL.

**In:**
- `GET /v1/availability?type=CAR` — REST for mini app
- `GetAvailability(VehicleType)` — gRPC for s2s
- Redis cache, 30 s TTL, key `avail:<vehicle_type>`
- Cache invalidation on any spot status change (admin tools)

**Out:**
- Floor-map heatmap (Beyond MVP)
- Real-time push updates to mini app (Beyond MVP)

## API contract

### REST
```
GET /v1/availability?type=CAR
→ 200 { "available_count": 142, "by_floor": [{"floor":1,"count":30}, ...] }
```

### gRPC
```proto
rpc GetAvailability(GetAvailabilityRequest) returns (GetAvailabilityResponse);
```

## Tasks

- [ ] `repository.SpotRepository.Available(vt)` — SELECT count + GROUP BY floor
- [ ] `usecase.Availability` with cache-aside
- [ ] gRPC handler
- [ ] REST handler `GET /v1/availability`
- [ ] Cache invalidation hook (manual admin path)
- [ ] Unit + integration tests

## Acceptance criteria

- Cold call hits DB; subsequent calls within 30 s hit Redis (verify via MONITOR).
- Response groups by floor in ascending order.
- Empty result returns `{ "available_count": 0, "by_floor": [] }` (not 404).
