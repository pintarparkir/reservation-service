package mock

import (
	"context"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/stretchr/testify/mock"
)

type MockReservationRepository struct {
	mock.Mock
}

func (m *MockReservationRepository) Create(ctx context.Context, r *model.Reservation) (*model.Reservation, error) {
	args := m.Called(ctx, r)
	out, _ := args.Get(0).(*model.Reservation)
	return out, args.Error(1)
}

func (m *MockReservationRepository) GetByID(ctx context.Context, id string) (*model.Reservation, error) {
	args := m.Called(ctx, id)
	out, _ := args.Get(0).(*model.Reservation)
	return out, args.Error(1)
}

func (m *MockReservationRepository) ApplyTransition(ctx context.Context, id string, action model.Action, eventType string, eventPayload []byte) (*model.Reservation, error) {
	args := m.Called(ctx, id, action, eventType, eventPayload)
	out, _ := args.Get(0).(*model.Reservation)
	return out, args.Error(1)
}

func (m *MockReservationRepository) ExpireDueReservations(ctx context.Context, limit int) ([]string, error) {
	args := m.Called(ctx, limit)
	out, _ := args.Get(0).([]string)
	return out, args.Error(1)
}

func (m *MockReservationRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error) {
	args := m.Called(ctx, key)
	out, _ := args.Get(0).(*model.Reservation)
	return out, args.Error(1)
}
