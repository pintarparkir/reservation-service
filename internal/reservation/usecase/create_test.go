package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
	mockrepo "github.com/farid/reservation-service/internal/reservation/repository/mock"
	"github.com/farid/reservation-service/internal/reservation/usecase"
	"github.com/farid/reservation-service/pkg/lock"
	"github.com/farid/reservation-service/pkg/redis"
	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	locks map[string]string
}

func newFakeCache() *fakeCache {
	return &fakeCache{locks: make(map[string]string)}
}

func (f *fakeCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}
func (f *fakeCache) Get(ctx context.Context, key string) (string, error) {
	return f.locks[key], nil
}
func (f *fakeCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if _, exists := f.locks[key]; exists {
		return false, nil
	}
	f.locks[key] = value.(string)
	return true, nil
}
func (f *fakeCache) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(f.locks, k)
	}
	return nil
}
func (f *fakeCache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if len(keys) > 0 {
		delete(f.locks, keys[0])
	}
	return int64(1), nil
}
func (f *fakeCache) Incr(ctx context.Context, key string) (int64, error)              { return 1, nil }
func (f *fakeCache) Expire(ctx context.Context, key string, ttl time.Duration) error  { return nil }
func (f *fakeCache) TTL(ctx context.Context, key string) (time.Duration, error)       { return time.Minute, nil }
func (f *fakeCache) Ping(ctx context.Context) error                                   { return nil }
func (f *fakeCache) Raw() *goredis.Client                                             { return nil }

var _ redis.Collections = (*fakeCache)(nil)

func TestCreate_HappyPath(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	resRepo.On("FindByIdempotencyKey", ctx, "idem-1").Return(nil, nil)
	spotRepo.On("Assign", ctx, model.VehicleTypeCar, "").Return("F1-C-001", nil)
	resRepo.On("Create", ctx, mock.AnythingOfType("*model.Reservation")).
		Return(&model.Reservation{
			ID:          "res-new",
			DriverID:    "drv-1",
			SpotID:      "F1-C-001",
			VehicleType: model.VehicleTypeCar,
			State:       model.StatePending,
		}, nil)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{
		HoldDuration: 15 * time.Minute,
	})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:       "drv-1",
		VehicleType:    model.VehicleTypeCar,
		Mode:           model.ModeSystemAssigned,
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "res-new", result.ID)
	assert.Equal(t, model.StatePending, result.State)
	resRepo.AssertExpectations(t)
	spotRepo.AssertExpectations(t)
}

func TestCreate_IdempotentReplay(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	existing := &model.Reservation{
		ID:          "res-existing",
		DriverID:    "drv-1",
		SpotID:      "F1-C-002",
		VehicleType: model.VehicleTypeCar,
		State:       model.StatePending,
	}
	resRepo.On("FindByIdempotencyKey", ctx, "idem-dup").Return(existing, nil)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{
		HoldDuration: 15 * time.Minute,
	})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:       "drv-1",
		VehicleType:    model.VehicleTypeCar,
		Mode:           model.ModeSystemAssigned,
		IdempotencyKey: "idem-dup",
	})

	require.NoError(t, err)
	assert.Equal(t, "res-existing", result.ID)
	spotRepo.AssertNotCalled(t, "Assign")
	resRepo.AssertNotCalled(t, "Create")
}

func TestCreate_ValidationError_MissingDriverID(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:       "",
		VehicleType:    model.VehicleTypeCar,
		Mode:           model.ModeSystemAssigned,
		IdempotencyKey: "idem-val",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "driver_id")
	resRepo.AssertNotCalled(t, "FindByIdempotencyKey")
}

func TestCreate_ValidationError_InvalidVehicleType(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:       "drv-1",
		VehicleType:    "TRUCK",
		Mode:           model.ModeSystemAssigned,
		IdempotencyKey: "idem-vt",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "vehicle_type")
}

func TestCreate_ValidationError_InvalidMode(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:       "drv-1",
		VehicleType:    model.VehicleTypeCar,
		Mode:           "INVALID_MODE",
		IdempotencyKey: "idem-mode",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mode")
}

func TestCreate_UserSelected_MissingPreferredSpot(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	resRepo.On("FindByIdempotencyKey", ctx, "idem-us").Return(nil, nil)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:        "drv-1",
		VehicleType:     model.VehicleTypeCar,
		Mode:            model.ModeUserSelected,
		PreferredSpotID: "",
		IdempotencyKey:  "idem-us",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "preferred_spot_id")
}

func TestCreate_UserSelected_WithPreferredSpot(t *testing.T) {
	ctx := context.Background()
	resRepo := new(mockrepo.MockReservationRepository)
	spotRepo := new(mockrepo.MockSpotRepository)
	cache := newFakeCache()
	lk := lock.New(cache)

	resRepo.On("FindByIdempotencyKey", ctx, "idem-pref").Return(nil, nil)
	spotRepo.On("Assign", ctx, model.VehicleTypeCar, "F2-C-010").Return("F2-C-010", nil)
	resRepo.On("Create", ctx, mock.AnythingOfType("*model.Reservation")).
		Return(&model.Reservation{
			ID:          "res-pref",
			DriverID:    "drv-1",
			SpotID:      "F2-C-010",
			VehicleType: model.VehicleTypeCar,
			State:       model.StatePending,
		}, nil)

	uc := usecase.NewReservationUsecase(resRepo, spotRepo, lk, usecase.Config{
		HoldDuration: 15 * time.Minute,
	})

	result, err := uc.Create(ctx, model.CreateReservationRequest{
		DriverID:        "drv-1",
		VehicleType:     model.VehicleTypeCar,
		Mode:            model.ModeUserSelected,
		PreferredSpotID: "F2-C-010",
		IdempotencyKey:  "idem-pref",
	})

	require.NoError(t, err)
	assert.Equal(t, "F2-C-010", result.SpotID)
	spotRepo.AssertExpectations(t)
}
