# Feature 06 — Check-in geofence

**Status:** ✅ shipped
**Owner:** reservation-service

## Scope

When the driver POSTs `/v1/reservations/{id}/check-in`, they include their GPS
coordinates. We refuse the check-in if they're more than `GEOFENCE_RADIUS_METERS`
from the building centre.

**In:**
- Haversine distance from `(lat, lng)` to `(CHECK_IN_BUILDING_LAT, CHECK_IN_BUILDING_LNG)`.
- Default radius 150 m.
- Soft-fail mode: client passes `gps_unavailable=true` → we accept and log
  `geofence_skipped=true` for audit.

**Out:**
- Multi-building (one geofence per building) — Beyond MVP.
- Bluetooth-beacon presence verification — Beyond MVP.

## Tasks

- [ ] `pkg/geo.Haversine(lat1,lng1,lat2,lng2) float64`
- [ ] `usecase.CheckIn` validates distance before flipping state
- [ ] Soft-fail flag wired through API (header or body field)
- [ ] Metric: `checkin_geofence_failures_total{outcome=outside|gps_off}`

## Acceptance criteria

- Coords inside the radius → check-in succeeds.
- Coords outside the radius → 422 with `error: GEOFENCE_VIOLATION`.
- `gps_unavailable=true` with garbage coords → 200 + audit log entry.
