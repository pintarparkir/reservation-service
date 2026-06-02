package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository/postgres"
	apperror "github.com/farid/reservation-service/pkg/error"
)

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return sqlx.NewDb(db, "postgres"), mock
}

func reservationRows() *sqlmock.Rows {
	now := time.Now().UTC()
	return sqlmock.NewRows([]string{
		"id", "driver_id", "spot_id", "vehicle_type", "state",
		"hold_start", "hold_end", "confirmed_at", "checked_in_at", "checked_out_at", "expires_at",
		"idempotency_key", "version", "created_at", "updated_at",
	}).AddRow(
		"res-1", "drv-1", "spot-1", "CAR", "PENDING",
		now, now.Add(15*time.Minute), nil, nil, nil, now.Add(15*time.Minute),
		"idem-1", 1, now, now,
	)
}

func TestReservationRepo_Create_HappyPath(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`WITH ins AS`).
		WithArgs("drv-1", "spot-1", model.VehicleTypeCar, model.StatePending, now, now.Add(15*time.Minute), "idem-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).
			AddRow("res-1", 1, now, now))
	mock.ExpectCommit()

	got, err := repo.Create(ctx, &model.Reservation{
		DriverID:       "drv-1",
		SpotID:         "spot-1",
		VehicleType:    model.VehicleTypeCar,
		State:          model.StatePending,
		HoldStart:      now,
		HoldEnd:        now.Add(15 * time.Minute),
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "res-1", got.ID)
	assert.Equal(t, 1, got.Version)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_Create_MapsDoubleBook(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`WITH ins AS`).
		WillReturnError(&pq.Error{Code: "23P01"})
	mock.ExpectRollback()

	_, err := repo.Create(ctx, &model.Reservation{
		DriverID: "drv-1", SpotID: "spot-1", VehicleType: model.VehicleTypeCar,
		State: model.StatePending, HoldStart: now, HoldEnd: now.Add(15 * time.Minute),
	})

	require.Error(t, err)
	assert.True(t, apperror.Is(err, apperror.ErrDoubleBook))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_GetByID_Found(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectQuery(`SELECT id, driver_id, spot_id`).
		WithArgs("res-1").
		WillReturnRows(reservationRows())

	got, err := repo.GetByID(ctx, "res-1")

	require.NoError(t, err)
	assert.Equal(t, "res-1", got.ID)
	assert.Equal(t, model.VehicleTypeCar, got.VehicleType)
	assert.Equal(t, model.StatePending, got.State)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectQuery(`SELECT id, driver_id, spot_id`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(ctx, "missing")

	require.Error(t, err)
	assert.True(t, apperror.Is(err, apperror.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_FindByIdempotencyKey_EmptyKey(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := postgres.NewReservationRepository(db)

	got, err := repo.FindByIdempotencyKey(context.Background(), "")

	require.NoError(t, err)
	assert.Nil(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_FindByIdempotencyKey_Found(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectQuery(`WHERE idempotency_key`).
		WithArgs("idem-1").
		WillReturnRows(reservationRows())

	got, err := repo.FindByIdempotencyKey(ctx, "idem-1")

	require.NoError(t, err)
	assert.Equal(t, "res-1", got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_ApplyTransition_HappyPath(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT state, version FROM reservation`).
		WithArgs("res-1").
		WillReturnRows(sqlmock.NewRows([]string{"state", "version"}).AddRow(model.StatePending, 1))
	mock.ExpectExec(`UPDATE reservation SET state`).
		WithArgs(model.StatePendingPayment, "res-1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO outbox_event`).
		WithArgs("res-1", model.EvtReservationConfirmed, []byte(`{}`)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT id, driver_id, spot_id`).
		WithArgs("res-1").
		WillReturnRows(reservationRows())

	got, err := repo.ApplyTransition(ctx, "res-1", model.ActionConfirm, model.EvtReservationConfirmed, []byte(`{}`))

	require.NoError(t, err)
	assert.Equal(t, "res-1", got.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_ApplyTransition_InvalidState(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT state, version FROM reservation`).
		WithArgs("res-1").
		WillReturnRows(sqlmock.NewRows([]string{"state", "version"}).AddRow(model.StateCompleted, 1))
	mock.ExpectRollback()

	_, err := repo.ApplyTransition(ctx, "res-1", model.ActionCancel, model.EvtReservationCancelled, nil)

	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReservationRepo_ExpireDueReservations_HappyPath(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewReservationRepository(db)

	mock.ExpectQuery(`WITH expired AS`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"aggregate_id"}).AddRow("res-1").AddRow("res-2"))

	ids, err := repo.ExpireDueReservations(ctx, 100)

	require.NoError(t, err)
	assert.Equal(t, []string{"res-1", "res-2"}, ids)
	assert.NoError(t, mock.ExpectationsWereMet())
}
