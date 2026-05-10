# Features — reservation-service

| File                              | Status | Summary                                       |
|-----------------------------------|--------|-----------------------------------------------|
| `01-spot-availability.md`         | ✅      | Availability by floor + vehicle_type          |
| `02-create-reservation.md`        | ✅      | SYSTEM_ASSIGNED + USER_SELECTED modes         |
| `03-state-machine.md`             | ✅      | PENDING → CONFIRMED → ACTIVE → COMPLETED      |
| `04-double-book-prevention.md`    | ✅      | DB-level EXCLUDE constraint, Redis lock       |
| `05-no-show-expiry.md`            | ✅      | 1 h hold; worker auto-expires + emits event   |
| `06-checkin-geofence.md`          | ✅      | Haversine, soft-fail on missing GPS           |
| `07-outbox-publisher.md`          | ✅      | At-least-once event delivery to RabbitMQ      |
| `08-billing-trigger.md`           | ✅      | Real gRPC client to billing on Create; async close via outbox |
| `09-rest-surface.md`              | ✅      | `/v1/availability`, `/v1/reservations*`       |

Legend: 📋 planned · ⏳ in progress · ✅ shipped · 🚫 deferred
