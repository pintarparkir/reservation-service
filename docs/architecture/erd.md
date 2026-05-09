# Entity Relationship Diagram — reservation-service

```mermaid
erDiagram
    SPOT ||--o{ RESERVATION : assigned_to
    RESERVATION ||--o{ OUTBOX_EVENT : emits

    SPOT {
        string id PK "F2-C-014"
        int floor "1..5"
        enum vehicle_type "CAR | MOTORCYCLE"
        enum status "AVAILABLE | OCCUPIED | OUT_OF_SERVICE"
        int version
    }
    RESERVATION {
        uuid id PK
        uuid driver_id "FK to user_service.user_profile (logical)"
        text spot_id FK
        enum vehicle_type
        enum state "PENDING|CONFIRMED|ACTIVE|COMPLETED|CANCELLED|EXPIRED"
        tstzrange hold_window "EXCLUDE constraint"
        timestamptz confirmed_at
        timestamptz checked_in_at
        timestamptz checked_out_at
        timestamptz expires_at
        text idempotency_key UK
        int version
        timestamptz created_at
        timestamptz updated_at
    }
    OUTBOX_EVENT {
        bigint id PK
        text aggregate_type
        text aggregate_id
        text event_type
        jsonb payload
        timestamptz created_at
        timestamptz published_at
    }
```

## Indexes & constraints

```sql
CREATE INDEX idx_reservation_driver_state ON reservation(driver_id, state);
CREATE INDEX idx_reservation_expires_at   ON reservation(expires_at) WHERE state='CONFIRMED';
CREATE UNIQUE INDEX uq_reservation_idem   ON reservation(idempotency_key) WHERE idempotency_key IS NOT NULL;

ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('CONFIRMED','ACTIVE'));

CREATE INDEX idx_outbox_unpublished ON outbox_event(created_at) WHERE published_at IS NULL;
```

## Cross-service references

`reservation.driver_id` is a *logical* foreign key to `user_service.user_profile.id`.
We do not enforce it with a real FK because the two services own distinct databases
in production. Integrity is enforced at the application boundary: reservation creation
calls `user-service.GetUserById` to validate the driver exists.
