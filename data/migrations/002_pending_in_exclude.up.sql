-- 002: Include PENDING in the no-overlap EXCLUDE constraint.
--
-- Without this, two drivers can both create PENDING reservations on the same
-- spot — only the second's confirm fails. This pushes the failure to the
-- earliest possible point: insert time.
--
-- Reservation states that "hold" the spot (in this order):
--   PENDING   — within hold_window grace, awaiting confirm/cancel
--   CONFIRMED — confirmed but not checked in
--   ACTIVE    — checked in, occupying
-- Released states: COMPLETED, CANCELLED, EXPIRED.

BEGIN;

ALTER TABLE reservation DROP CONSTRAINT IF EXISTS no_overlapping_reservation;

ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('PENDING','CONFIRMED','ACTIVE'));

COMMIT;
