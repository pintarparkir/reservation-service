BEGIN;

ALTER TABLE reservation DROP CONSTRAINT IF EXISTS no_overlapping_reservation;

ALTER TABLE reservation ADD CONSTRAINT no_overlapping_reservation
  EXCLUDE USING gist (spot_id WITH =, hold_window WITH &&)
  WHERE (state IN ('CONFIRMED','ACTIVE'));

COMMIT;
