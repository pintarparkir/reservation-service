package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
	apperror "github.com/farid/reservation-service/pkg/error"
)

// pgErrCodes from libpq we care about.
const (
	codeUniqueViolation    = "23505"
	codeExclusionViolation = "23P01"
)

type reservationRepo struct{ db *sqlx.DB }

func NewReservationRepository(db *sqlx.DB) repository.ReservationRepository {
	return &reservationRepo{db: db}
}

// insertReservationSQL inserts the reservation and the outbox event in a
// single CTE. The outbox payload is built from the just-inserted row so it
// always contains the real reservation_id (previous Go-side pre-marshal
// embedded a placeholder string — billing consumed broken events).
const insertReservationSQL = `
WITH ins AS (
    INSERT INTO reservation (driver_id, spot_id, vehicle_type, state, hold_window, expires_at, idempotency_key)
    VALUES ($1, $2, $3, $4, tstzrange($5, $6, '[)'), $6, $7)
    RETURNING id, driver_id, spot_id, vehicle_type, version, created_at, updated_at
),
ob AS (
    INSERT INTO outbox_event (aggregate_type, aggregate_id, event_type, payload)
    SELECT 'reservation', ins.id, 'reservation.created.v1',
           jsonb_build_object(
             'reservation_id', ins.id,
             'driver_id',      ins.driver_id,
             'spot_id',        ins.spot_id,
             'vehicle_type',   ins.vehicle_type,
             'hold_end',       to_char($6::timestamptz AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
           )
    FROM ins
)
SELECT id, version, created_at, updated_at FROM ins
`

const insertOutboxSQL = `
INSERT INTO outbox_event (aggregate_type, aggregate_id, event_type, payload)
VALUES ('reservation', $1, $2, $3)
`

func (r *reservationRepo) Create(ctx context.Context, in *model.Reservation) (*model.Reservation, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	out := *in
	err = tx.QueryRowxContext(ctx, insertReservationSQL,
		in.DriverID, in.SpotID, in.VehicleType, in.State,
		in.HoldStart, in.HoldEnd, nullable(in.IdempotencyKey),
	).Scan(&out.ID, &out.Version, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, mapInsertErr(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &out, nil
}

const getByIDSQL = `
SELECT id, driver_id, spot_id, vehicle_type, state,
       lower(hold_window) AS hold_start, upper(hold_window) AS hold_end,
       confirmed_at, checked_in_at, checked_out_at, expires_at,
       coalesce(idempotency_key,'') AS idempotency_key,
       version, created_at, updated_at
FROM reservation
WHERE id = $1
`

func (r *reservationRepo) GetByID(ctx context.Context, id string) (*model.Reservation, error) {
	row := r.db.QueryRowxContext(ctx, getByIDSQL, id)
	var rv reservationRow
	if err := row.StructScan(&rv); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}
	return rv.toModel(), nil
}

const findByIdemSQL = `
SELECT id, driver_id, spot_id, vehicle_type, state,
       lower(hold_window) AS hold_start, upper(hold_window) AS hold_end,
       confirmed_at, checked_in_at, checked_out_at, expires_at,
       coalesce(idempotency_key,'') AS idempotency_key,
       version, created_at, updated_at
FROM reservation
WHERE idempotency_key = $1
`

func (r *reservationRepo) FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error) {
	if key == "" {
		return nil, nil
	}
	row := r.db.QueryRowxContext(ctx, findByIdemSQL, key)
	var rv reservationRow
	if err := row.StructScan(&rv); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rv.toModel(), nil
}

// ApplyTransition does:
//   1. SELECT current state + version (FOR UPDATE).
//   2. Compute next via model.Next; bail with INVALID_STATE on illegal.
//   3. UPDATE state, version+1, plus per-action timestamp columns.
//   4. INSERT outbox_event in the same tx.
func (r *reservationRepo) ApplyTransition(ctx context.Context, id string, action model.Action, eventType string, payload []byte) (*model.Reservation, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var current model.ReservationState
	var version int
	err = tx.QueryRowxContext(ctx,
		`SELECT state, version FROM reservation WHERE id = $1 FOR UPDATE`, id,
	).Scan(&current, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	next, err := model.Next(current, action)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, updateForAction(action),
		next, id, version,
	); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, insertOutboxSQL, id, eventType, payload); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

// updateForAction returns SQL that flips state + timestamps a relevant column.
func updateForAction(a model.Action) string {
	tsCol := ""
	switch a {
	case model.ActionConfirm:
		tsCol = ", confirmed_at = now()"
	case model.ActionCheckIn:
		tsCol = ", checked_in_at = now()"
	case model.ActionCheckOut:
		tsCol = ", checked_out_at = now()"
	}
	var sb strings.Builder
	sb.WriteString("UPDATE reservation SET state = $1, version = version + 1, updated_at = now()")
	sb.WriteString(tsCol)
	sb.WriteString(" WHERE id = $2 AND version = $3")
	return sb.String()
}

const expireSQL = `
WITH expired AS (
  SELECT id FROM reservation
  WHERE state = 'CONFIRMED' AND expires_at < now()
  ORDER BY expires_at
  FOR UPDATE SKIP LOCKED
  LIMIT $1
),
upd AS (
  UPDATE reservation r
     SET state = 'EXPIRED', version = r.version + 1, updated_at = now()
    FROM expired e
   WHERE r.id = e.id
  RETURNING r.id, r.driver_id
)
INSERT INTO outbox_event (aggregate_type, aggregate_id, event_type, payload)
SELECT 'reservation', upd.id, 'reservation.expired.v1',
       jsonb_build_object('reservation_id', upd.id, 'driver_id', upd.driver_id)
  FROM upd
RETURNING aggregate_id
`

func (r *reservationRepo) ExpireDueReservations(ctx context.Context, limit int) ([]string, error) {
	rows, err := r.db.QueryxContext(ctx, expireSQL, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// reservationRow is the scannable shape mirroring getByIDSQL columns.
type reservationRow struct {
	ID             string         `db:"id"`
	DriverID       string         `db:"driver_id"`
	SpotID         string         `db:"spot_id"`
	VehicleType    string         `db:"vehicle_type"`
	State          string         `db:"state"`
	HoldStart      pq.NullTime    `db:"hold_start"`
	HoldEnd        pq.NullTime    `db:"hold_end"`
	ConfirmedAt    pq.NullTime    `db:"confirmed_at"`
	CheckedInAt    pq.NullTime    `db:"checked_in_at"`
	CheckedOutAt   pq.NullTime    `db:"checked_out_at"`
	ExpiresAt      pq.NullTime    `db:"expires_at"`
	IdempotencyKey string         `db:"idempotency_key"`
	Version        int            `db:"version"`
	CreatedAt      pq.NullTime    `db:"created_at"`
	UpdatedAt      pq.NullTime    `db:"updated_at"`
}

func (r *reservationRow) toModel() *model.Reservation {
	out := &model.Reservation{
		ID:             r.ID,
		DriverID:       r.DriverID,
		SpotID:         r.SpotID,
		VehicleType:    model.VehicleType(r.VehicleType),
		State:          model.ReservationState(r.State),
		IdempotencyKey: r.IdempotencyKey,
		Version:        r.Version,
	}
	if r.HoldStart.Valid {
		out.HoldStart = r.HoldStart.Time
	}
	if r.HoldEnd.Valid {
		out.HoldEnd = r.HoldEnd.Time
	}
	if r.ConfirmedAt.Valid {
		t := r.ConfirmedAt.Time
		out.ConfirmedAt = &t
	}
	if r.CheckedInAt.Valid {
		t := r.CheckedInAt.Time
		out.CheckedInAt = &t
	}
	if r.CheckedOutAt.Valid {
		t := r.CheckedOutAt.Time
		out.CheckedOutAt = &t
	}
	if r.ExpiresAt.Valid {
		t := r.ExpiresAt.Time
		out.ExpiresAt = &t
	}
	if r.CreatedAt.Valid {
		out.CreatedAt = r.CreatedAt.Time
	}
	if r.UpdatedAt.Valid {
		out.UpdatedAt = r.UpdatedAt.Time
	}
	return out
}

func mapInsertErr(err error) error {
	if pgErr, ok := err.(*pq.Error); ok {
		switch string(pgErr.Code) {
		case codeExclusionViolation:
			return apperror.ErrDoubleBook
		case codeUniqueViolation:
			return apperror.ErrConflict
		}
	}
	return err
}

func nullable(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
