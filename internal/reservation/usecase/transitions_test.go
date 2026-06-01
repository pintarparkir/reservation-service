package usecase_test

import (
	"context"
	"testing"

	"github.com/farid/reservation-service/internal/reservation/model"
	mockrepo "github.com/farid/reservation-service/internal/reservation/repository/mock"
	"github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ── Confirm tests ────────────────────────────────────────────────────────────

func TestConfirm_HappyPath(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	pending := &model.Reservation{ID: "res-1", State: model.StatePending, DriverID: "drv-1"}
	pendingPayment := &model.Reservation{ID: "res-1", State: model.StatePendingPayment, DriverID: "drv-1"}

	resRepo.On("GetByID", ctx, "res-1").Return(pending, nil)
	resRepo.On("ApplyTransition", ctx, "res-1", model.ActionConfirm, model.EvtReservationPaymentPending, mock.Anything).
		Return(pendingPayment, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Confirm(ctx, "res-1")

	require.NoError(t, err)
	assert.Equal(t, model.StatePendingPayment, result.State)
	resRepo.AssertExpectations(t)
}

func TestConfirm_AlreadyConfirmed(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	confirmed := &model.Reservation{ID: "res-2", State: model.StateConfirmed, DriverID: "drv-2"}

	resRepo.On("GetByID", ctx, "res-2").Return(confirmed, nil)
	resRepo.On("ApplyTransition", ctx, "res-2", model.ActionConfirm, model.EvtReservationPaymentPending, mock.Anything).
		Return(confirmed, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Confirm(ctx, "res-2")

	require.NoError(t, err)
	assert.Equal(t, model.StateConfirmed, result.State)
	resRepo.AssertExpectations(t)
}

func TestConfirm_InvalidStateTransition(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	active := &model.Reservation{ID: "res-3", State: model.StateActive, DriverID: "drv-3"}

	resRepo.On("GetByID", ctx, "res-3").Return(active, nil)
	resRepo.On("ApplyTransition", ctx, "res-3", model.ActionConfirm, model.EvtReservationPaymentPending, mock.Anything).
		Return(nil, assert.AnError)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Confirm(ctx, "res-3")

	require.Error(t, err)
	assert.Nil(t, result)
	resRepo.AssertExpectations(t)
}

// ── Cancel tests ─────────────────────────────────────────────────────────────

func TestCancel_FromPending(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	pending := &model.Reservation{ID: "res-4", State: model.StatePending, DriverID: "drv-4"}
	cancelled := &model.Reservation{ID: "res-4", State: model.StateCancelled, DriverID: "drv-4"}

	resRepo.On("GetByID", ctx, "res-4").Return(pending, nil)
	resRepo.On("ApplyTransition", ctx, "res-4", model.ActionCancel, model.EvtReservationCancelled, mock.Anything).
		Return(cancelled, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Cancel(ctx, model.CancelRequest{ID: "res-4", Reason: "driver_request"})

	require.NoError(t, err)
	assert.Equal(t, model.StateCancelled, result.State)
	resRepo.AssertExpectations(t)
}

func TestCancel_FromConfirmed(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	confirmed := &model.Reservation{ID: "res-5", State: model.StateConfirmed, DriverID: "drv-5"}
	cancelled := &model.Reservation{ID: "res-5", State: model.StateCancelled, DriverID: "drv-5"}

	resRepo.On("GetByID", ctx, "res-5").Return(confirmed, nil)
	resRepo.On("ApplyTransition", ctx, "res-5", model.ActionCancel, model.EvtReservationCancelled, mock.Anything).
		Return(cancelled, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Cancel(ctx, model.CancelRequest{ID: "res-5", Reason: "driver_request"})

	require.NoError(t, err)
	assert.Equal(t, model.StateCancelled, result.State)
	resRepo.AssertExpectations(t)
}

func TestCancel_InvalidState(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	completed := &model.Reservation{ID: "res-6", State: model.StateCompleted, DriverID: "drv-6"}

	resRepo.On("GetByID", ctx, "res-6").Return(completed, nil)
	resRepo.On("ApplyTransition", ctx, "res-6", model.ActionCancel, model.EvtReservationCancelled, mock.Anything).
		Return(nil, assert.AnError)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Cancel(ctx, model.CancelRequest{ID: "res-6", Reason: "driver_request"})

	require.Error(t, err)
	assert.Nil(t, result)
	resRepo.AssertExpectations(t)
}

// ── CheckIn tests ────────────────────────────────────────────────────────────

func TestCheckIn_HappyPath(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	active := &model.Reservation{ID: "res-7", State: model.StateActive, DriverID: "drv-7"}

	resRepo.On("ApplyTransition", ctx, "res-7", model.ActionCheckIn, model.EvtReservationCheckedIn, mock.Anything).
		Return(active, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{
		GeofenceRadiusMeters: 100,
		BuildingLat:          -6.2,
		BuildingLng:          106.8,
	})

	result, err := uc.CheckIn(ctx, model.CheckInRequest{
		ID:        "res-7",
		Latitude:  -6.2,
		Longitude: 106.8,
	})

	require.NoError(t, err)
	assert.Equal(t, model.StateActive, result.State)
	resRepo.AssertExpectations(t)
}

func TestCheckIn_OutsideGeofence(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{
		GeofenceRadiusMeters: 100,
		BuildingLat:          -6.2,
		BuildingLng:          106.8,
	})

	result, err := uc.CheckIn(ctx, model.CheckInRequest{
		ID:        "res-8",
		Latitude:  -6.3,
		Longitude: 106.9,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "outside")
	resRepo.AssertExpectations(t)
}

// ── CheckOut tests ───────────────────────────────────────────────────────────

func TestCheckOut_HappyPath(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	active := &model.Reservation{ID: "res-9", State: model.StateActive, DriverID: "drv-9"}
	completed := &model.Reservation{ID: "res-9", State: model.StateCompleted, DriverID: "drv-9"}

	resRepo.On("GetByID", ctx, "res-9").Return(active, nil)
	resRepo.On("ApplyTransition", ctx, "res-9", model.ActionCheckOut, model.EvtReservationCheckedOut, mock.Anything).
		Return(completed, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.CheckOut(ctx, "res-9")

	require.NoError(t, err)
	assert.Equal(t, model.StateCompleted, result.State)
	resRepo.AssertExpectations(t)
}

func TestCheckOut_InvalidState(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	pending := &model.Reservation{ID: "res-10", State: model.StatePending, DriverID: "drv-10"}

	resRepo.On("GetByID", ctx, "res-10").Return(pending, nil)
	resRepo.On("ApplyTransition", ctx, "res-10", model.ActionCheckOut, model.EvtReservationCheckedOut, mock.Anything).
		Return(nil, assert.AnError)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.CheckOut(ctx, "res-10")

	require.Error(t, err)
	assert.Nil(t, result)
	resRepo.AssertExpectations(t)
}

// ── Get tests ────────────────────────────────────────────────────────────────

func TestGet_HappyPath(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	reservation := &model.Reservation{ID: "res-11", State: model.StatePending, DriverID: "drv-11"}
	resRepo.On("GetByID", ctx, "res-11").Return(reservation, nil)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Get(ctx, "res-11")

	require.NoError(t, err)
	assert.Equal(t, reservation, result)
	resRepo.AssertExpectations(t)
}

func TestGet_PropagatesRepoError(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)

	resRepo.On("GetByID", ctx, "missing").Return(nil, assert.AnError)

	uc := usecase.NewReservationUsecase(resRepo, nil, nil, usecase.Config{})

	result, err := uc.Get(ctx, "missing")

	require.Error(t, err)
	assert.Nil(t, result)
	resRepo.AssertExpectations(t)
}
