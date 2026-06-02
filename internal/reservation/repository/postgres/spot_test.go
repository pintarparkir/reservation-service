package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository/postgres"
	apperror "github.com/farid/reservation-service/pkg/error"
)

func TestSpotRepo_Assign_SystemAssigned(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewSpotRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT s.id FROM spot s`).
		WithArgs(model.VehicleTypeCar).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("spot-1"))
	mock.ExpectCommit()

	got, err := repo.Assign(ctx, model.VehicleTypeCar, "")

	require.NoError(t, err)
	assert.Equal(t, "spot-1", got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSpotRepo_Assign_PreferredSpot(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewSpotRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM spot`).
		WithArgs("spot-2", model.VehicleTypeMotorcycle).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("spot-2"))
	mock.ExpectCommit()

	got, err := repo.Assign(ctx, model.VehicleTypeMotorcycle, "spot-2")

	require.NoError(t, err)
	assert.Equal(t, "spot-2", got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSpotRepo_Assign_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewSpotRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT s.id FROM spot s`).
		WithArgs(model.VehicleTypeCar).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err := repo.Assign(ctx, model.VehicleTypeCar, "")

	require.Error(t, err)
	assert.True(t, apperror.Is(err, apperror.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSpotRepo_AvailabilityByFloor(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := postgres.NewSpotRepository(db)

	mock.ExpectQuery(`SELECT s.floor`).
		WithArgs(model.VehicleTypeCar).
		WillReturnRows(sqlmock.NewRows([]string{"floor", "count"}).
			AddRow(1, 3).
			AddRow(2, 4))

	floors, total, err := repo.AvailabilityByFloor(ctx, model.VehicleTypeCar)

	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Len(t, floors, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}
