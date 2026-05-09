# Feature 09 — REST surface for mini app

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

Tencent Mini Program calls this service directly over HTTPS (no gateway).
JWT verified per-request via the same stdlib middleware pattern as user-service.

## Routes

| Method | Path                                | Maps to gRPC RPC      |
|--------|-------------------------------------|-----------------------|
| GET    | /v1/availability?type=CAR           | GetAvailability       |
| POST   | /v1/reservations                    | CreateReservation     |
| GET    | /v1/reservations/{id}               | GetReservation        |
| POST   | /v1/reservations/{id}/confirm       | ConfirmReservation    |
| POST   | /v1/reservations/{id}/cancel        | CancelReservation     |
| POST   | /v1/reservations/{id}/check-in      | CheckIn               |
| POST   | /v1/reservations/{id}/check-out     | CheckOut              |

All except `GET /v1/availability` require `Idempotency-Key` on writes.

## Middleware stack

```
1. logger middleware (request_id injection)
2. otelgin (trace propagation)
3. jwt middleware (extract sub + phone from RS256 token)
4. driver-resolver middleware (gRPC → user-service.GetUserById on sub) → sets driver_id
5. handler
```

The driver-resolver caches `external_user_id → driver_id` for 5 min in Redis to
avoid hammering user-service on the hot path.

## Tasks

- [ ] `internal/reservation/handler/http/middleware.go` (mirror user-service pattern)
- [ ] `pkg/jwt` — re-use stdlib RS256 verifier (or factor up to a shared `pkg/`)
- [ ] Driver-resolver cache wired in
- [ ] All seven handlers, each ~30 LOC
- [ ] Error mapper from gRPC code → HTTP status

## Acceptance criteria

- Each REST call lands in the same usecase as the gRPC equivalent.
- A request without a JWT → 401.
- A request with a JWT but unknown `sub` → 401 with body explaining "driver not registered"
  (the user-service should have created the row already on first /v1/me call;
  if it hasn't, the mini app's flow is broken).
