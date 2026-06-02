-- 003 down: Revert PENDING_PAYMENT additions.
-- Note: PostgreSQL does not support removing enum values directly.
-- This rollback removes the column and reverts the constraint.

BEGIN;

DROP INDEX IF EXISTS idx_reservation_payment_expires;

ALTER TABLE reservation DROP CONSTRAINT IF EXISTS no_overlapping_reservation;

ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('PENDING','CONFIRMED','ACTIVE'));

ALTER TABLE reservation DROP COLUMN IF EXISTS payment_expires_at;

COMMIT;
