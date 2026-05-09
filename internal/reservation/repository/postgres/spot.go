package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
	apperror "github.com/farid/reservation-service/pkg/error"
)

type spotRepo struct{ db *sqlx.DB }

func NewSpotRepository(db *sqlx.DB) repository.SpotRepository { return &spotRepo{db: db} }

const assignSystemSQL = `
SELECT id FROM spot
WHERE vehicle_type = $1 AND status = 'AVAILABLE'
ORDER BY id
FOR UPDATE SKIP LOCKED
LIMIT 1
`

const assignUserSQL = `
SELECT id FROM spot
WHERE id = $1 AND vehicle_type = $2 AND status = 'AVAILABLE'
FOR UPDATE
`

// Assign returns the chosen spot id. If preferred is empty it picks any AVAILABLE
// spot of the requested vehicle_type using SKIP LOCKED. The returned row is
// row-locked under the caller's transaction — but here we use a short tx via
// sqlx; the EXCLUDE constraint on `reservation` is the authoritative guard so
// the row-lock is just defence-in-depth.
func (r *spotRepo) Assign(ctx context.Context, vt model.VehicleType, preferred string) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var id string
	if preferred != "" {
		err = tx.QueryRowxContext(ctx, assignUserSQL, preferred, vt).Scan(&id)
	} else {
		err = tx.QueryRowxContext(ctx, assignSystemSQL, vt).Scan(&id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", apperror.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

const availSQL = `
SELECT floor, count(*)::int AS count
FROM spot
WHERE vehicle_type = $1 AND status = 'AVAILABLE'
GROUP BY floor
ORDER BY floor
`

func (r *spotRepo) AvailabilityByFloor(ctx context.Context, vt model.VehicleType) ([]repository.FloorCount, int, error) {
	rows, err := r.db.QueryxContext(ctx, availSQL, vt)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []repository.FloorCount
	total := 0
	for rows.Next() {
		var fc repository.FloorCount
		if err := rows.StructScan(&fc); err != nil {
			return nil, 0, err
		}
		out = append(out, fc)
		total += fc.Count
	}
	return out, total, rows.Err()
}
