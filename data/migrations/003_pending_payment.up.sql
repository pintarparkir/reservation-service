-- 003: Add PENDING_PAYMENT state and payment_expires_at column.
--
-- The booking-fee flow requires:
--   PENDING → PENDING_PAYMENT (on confirm, awaiting payment)
--   PENDING_PAYMENT → CONFIRMED (on payment success)
--   PENDING_PAYMENT → CANCELLED (on payment failure/timeout)

BEGIN;

-- Add PENDING_PAYMENT to the reservation_state enum
ALTER TYPE reservation_state ADD VALUE IF NOT EXISTS 'PENDING_PAYMENT' AFTER 'PENDING';

COMMIT;

-- Must be outside the above transaction because ADD VALUE can't run inside
-- a multi-statement transaction in older PG versions, but PG 16 is fine.
-- We use a separate block for the DDL that references the new enum value.

BEGIN;

-- Add payment_expires_at column for payment timeout tracking
ALTER TABLE reservation ADD COLUMN IF NOT EXISTS payment_expires_at timestamptz;

-- Update the exclusion constraint to also hold the spot during PENDING_PAYMENT
ALTER TABLE reservation DROP CONSTRAINT IF EXISTS no_overlapping_reservation;

ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('PENDING','PENDING_PAYMENT','CONFIRMED','ACTIVE'));

-- Index for payment timeout worker
CREATE INDEX IF NOT EXISTS idx_reservation_payment_expires
  ON reservation(payment_expires_at)
  WHERE state = 'PENDING_PAYMENT';

COMMIT;
