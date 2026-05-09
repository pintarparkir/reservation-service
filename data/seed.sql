-- Seed: 5 floors × 30 CAR spots + 5 floors × 20 MOTORCYCLE spots = 250 spots.

BEGIN;

INSERT INTO spot (id, floor, vehicle_type, status)
SELECT
  format('F%s-C-%03s', floor_num, spot_num) AS id,
  floor_num,
  'CAR'::vehicle_type,
  'AVAILABLE'::spot_status
FROM generate_series(1, 5) AS floor_num
CROSS JOIN generate_series(1, 30) AS spot_num
ON CONFLICT (id) DO NOTHING;

INSERT INTO spot (id, floor, vehicle_type, status)
SELECT
  format('F%s-M-%03s', floor_num, spot_num) AS id,
  floor_num,
  'MOTORCYCLE'::vehicle_type,
  'AVAILABLE'::spot_status
FROM generate_series(1, 5) AS floor_num
CROSS JOIN generate_series(1, 20) AS spot_num
ON CONFLICT (id) DO NOTHING;

COMMIT;
