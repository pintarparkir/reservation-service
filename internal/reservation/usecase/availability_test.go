package usecase_test

import (
	"context"
	"testing"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
	mockrepo "github.com/farid/reservation-service/internal/reservation/repository/mock"
	"github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvailability_ReturnsFloorCounts(t *testing.T) {
	ctx := context.Background()
	spotRepo := new(mockrepo.MockSpotRepository)

	floors := []repository.FloorCount{
		{Floor: 1, Count: 5},
		{Floor: 2, Count: 3},
	}
	spotRepo.On("AvailabilityByFloor", ctx, model.VehicleTypeCar).
		Return(floors, 8, nil)

	uc := usecase.NewReservationUsecase(nil, spotRepo, nil, usecase.Config{})

	result, total, err := uc.Availability(ctx, model.VehicleTypeCar)

	require.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 8, total)
	assert.Equal(t, 1, result[0].Floor)
	assert.Equal(t, 5, result[0].Count)
	spotRepo.AssertExpectations(t)
}

func TestAvailability_InvalidVehicleType(t *testing.T) {
	ctx := context.Background()
	spotRepo := new(mockrepo.MockSpotRepository)

	uc := usecase.NewReservationUsecase(nil, spotRepo, nil, usecase.Config{})

	result, total, err := uc.Availability(ctx, "INVALID")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	spotRepo.AssertNotCalled(t, "AvailabilityByFloor")
}

func TestAvailability_MotorcycleType(t *testing.T) {
	ctx := context.Background()
	spotRepo := new(mockrepo.MockSpotRepository)

	floors := []repository.FloorCount{
		{Floor: 1, Count: 10},
	}
	spotRepo.On("AvailabilityByFloor", ctx, model.VehicleTypeMotorcycle).
		Return(floors, 10, nil)

	uc := usecase.NewReservationUsecase(nil, spotRepo, nil, usecase.Config{})

	result, total, err := uc.Availability(ctx, model.VehicleTypeMotorcycle)

	require.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, 10, total)
	spotRepo.AssertExpectations(t)
}

func TestAvailability_NoSpotsAvailable(t *testing.T) {
	ctx := context.Background()
	spotRepo := new(mockrepo.MockSpotRepository)

	spotRepo.On("AvailabilityByFloor", ctx, model.VehicleTypeCar).
		Return([]repository.FloorCount{}, 0, nil)

	uc := usecase.NewReservationUsecase(nil, spotRepo, nil, usecase.Config{})

	result, total, err := uc.Availability(ctx, model.VehicleTypeCar)

	require.NoError(t, err)
	assert.Equal(t, 0, len(result))
	assert.Equal(t, 0, total)
	spotRepo.AssertExpectations(t)
}
