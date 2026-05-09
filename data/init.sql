-- reservation-service Postgres schema.
-- Mounted into the postgres container's /docker-entrypoint-initdb.d/ for fresh dev.
-- Production: applied via golang-migrate from data/migrations/.

BEGIN;

CREATE EXTENSION IF NOT EXISTS btree_gist;  -- needed for the EXCLUDE constraint below

-- ── Enums ─────────────────────────────────────────────────────────────────────
DO $$ BEGIN
  CREATE TYPE vehicle_type AS ENUM ('CAR','MOTORCYCLE');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE reservation_state AS ENUM
    ('PENDING','CONFIRMED','ACTIVE','COMPLETED','CANCELLED','EXPIRED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  CREATE TYPE spot_status AS ENUM ('AVAILABLE','OCCUPIED','OUT_OF_SERVICE');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ── Spot (parking spot inventory) ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS spot (
  id            text PRIMARY KEY,                       -- e.g. F2-C-014
  floor         int  NOT NULL CHECK (floor BETWEEN 1 AND 5),
  vehicle_type  vehicle_type NOT NULL,
  status        spot_status  NOT NULL DEFAULT 'AVAILABLE',
  version       int          NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_spot_avail ON spot(vehicle_type, status) WHERE status='AVAILABLE';

-- ── Reservation ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reservation (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  driver_id       uuid NOT NULL,
  spot_id         text NOT NULL REFERENCES spot(id),
  vehicle_type    vehicle_type NOT NULL,
  state           reservation_state NOT NULL DEFAULT 'PENDING',
  hold_window     tstzrange NOT NULL,
  confirmed_at    timestamptz,
  checked_in_at   timestamptz,
  checked_out_at  timestamptz,
  expires_at      timestamptz,
  idempotency_key text,
  version         int NOT NULL DEFAULT 1,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  -- DB-level double-book prevention. See ADR-002 in this service.
  CONSTRAINT no_overlapping_reservation EXCLUDE USING gist
    (spot_id WITH =, hold_window WITH &&)
    WHERE (state IN ('CONFIRMED','ACTIVE'))
);
CREATE INDEX IF NOT EXISTS idx_reservation_driver_state ON reservation(driver_id, state);
CREATE INDEX IF NOT EXISTS idx_reservation_expires_at   ON reservation(expires_at) WHERE state = 'CONFIRMED';
CREATE UNIQUE INDEX IF NOT EXISTS uq_reservation_idem   ON reservation(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- ── Outbox ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS outbox_event (
  id              bigserial PRIMARY KEY,
  aggregate_type  text NOT NULL,
  aggregate_id    text NOT NULL,
  event_type      text NOT NULL,
  payload         jsonb NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  published_at    timestamptz
);
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox_event(created_at) WHERE published_at IS NULL;

COMMIT;
