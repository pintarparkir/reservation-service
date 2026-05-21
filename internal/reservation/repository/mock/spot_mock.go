package mock

import (
	"context"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
	"github.com/stretchr/testify/mock"
)

type MockSpotRepository struct {
	mock.Mock
}

func (m *MockSpotRepository) Assign(ctx context.Context, vt model.VehicleType, preferred string) (string, error) {
	args := m.Called(ctx, vt, preferred)
	return args.String(0), args.Error(1)
}

func (m *MockSpotRepository) AvailabilityByFloor(ctx context.Context, vt model.VehicleType) ([]repository.FloorCount, int, error) {
	args := m.Called(ctx, vt)
	out, _ := args.Get(0).([]repository.FloorCount)
	return out, args.Int(1), args.Error(2)
}
