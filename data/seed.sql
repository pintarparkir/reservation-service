-- Seed: 5 floors × 30 CAR spots + 5 floors × 20 MOTORCYCLE spots = 250 spots.
-- Spot id format: F<floor>-{C|M}-<3-digit zero-padded number>, e.g. F1-C-001, F5-M-020.

BEGIN;

INSERT INTO spot (id, floor, vehicle_type, status)
SELECT
  'F' || floor_num || '-C-' || lpad(spot_num::text, 3, '0') AS id,
  floor_num,
  'CAR'::vehicle_type,
  'AVAILABLE'::spot_status
FROM generate_series(1, 5) AS floor_num
CROSS JOIN generate_series(1, 30) AS spot_num
ON CONFLICT (id) DO NOTHING;

INSERT INTO spot (id, floor, vehicle_type, status)
SELECT
  'F' || floor_num || '-M-' || lpad(spot_num::text, 3, '0') AS id,
  floor_num,
  'MOTORCYCLE'::vehicle_type,
  'AVAILABLE'::spot_status
FROM generate_series(1, 5) AS floor_num
CROSS JOIN generate_series(1, 20) AS spot_num
ON CONFLICT (id) DO NOTHING;

COMMIT;
