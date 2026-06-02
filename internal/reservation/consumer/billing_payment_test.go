package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/farid/reservation-service/internal/reservation/model"
	mockrepo "github.com/farid/reservation-service/internal/reservation/repository/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandle_PaymentSuccess_RoutesToConfirmed(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	body, _ := json.Marshal(paymentEvent{ReservationID: "res-1", PaymentRef: "pay-ref-1"})

	repo.On("ApplyTransition", ctx, "res-1", model.ActionPaymentSuccess, model.EvtReservationConfirmed, mock.AnythingOfType("[]uint8")).
		Return(&model.Reservation{ID: "res-1", State: model.StateConfirmed}, nil)

	err := c.Handle(ctx, model.EvtPaymentSuccess, body)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandle_PaymentFailed_RoutesToCancelled(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	body, _ := json.Marshal(paymentEvent{ReservationID: "res-2", Reason: "insufficient_funds"})

	repo.On("ApplyTransition", ctx, "res-2", model.ActionPaymentFail, model.EvtReservationCancelled, mock.AnythingOfType("[]uint8")).
		Return(&model.Reservation{ID: "res-2", State: model.StateCancelled}, nil)

	err := c.Handle(ctx, model.EvtPaymentFailed, body)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandle_UnknownRoutingKey_ReturnsError(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	err := c.Handle(ctx, "billing.unknown.event.v1", []byte(`{}`))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown routing key")
	repo.AssertNotCalled(t, "ApplyTransition")
}

func TestHandlePaymentConfirmed_InvalidJSON(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	err := c.HandlePaymentConfirmed(ctx, []byte(`not json`))

	assert.Error(t, err)
	repo.AssertNotCalled(t, "ApplyTransition")
}

func TestHandlePaymentFailed_InvalidJSON(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	err := c.HandlePaymentFailed(ctx, []byte(`{invalid`))

	assert.Error(t, err)
	repo.AssertNotCalled(t, "ApplyTransition")
}

func TestHandlePaymentConfirmed_RepoError_Propagates(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	body, _ := json.Marshal(paymentEvent{ReservationID: "res-3"})
	repoErr := errors.New("db connection lost")

	repo.On("ApplyTransition", ctx, "res-3", model.ActionPaymentSuccess, model.EvtReservationConfirmed, mock.AnythingOfType("[]uint8")).
		Return(nil, repoErr)

	err := c.HandlePaymentConfirmed(ctx, body)

	assert.ErrorIs(t, err, repoErr)
	repo.AssertExpectations(t)
}

func TestHandlePaymentFailed_RepoError_Propagates(t *testing.T) {
	repo := new(mockrepo.MockReservationRepository)
	c := NewBillingPaymentConsumer(repo)
	ctx := context.Background()

	body, _ := json.Marshal(paymentEvent{ReservationID: "res-4", Reason: "timeout"})
	repoErr := errors.New("deadlock detected")

	repo.On("ApplyTransition", ctx, "res-4", model.ActionPaymentFail, model.EvtReservationCancelled, mock.AnythingOfType("[]uint8")).
		Return(nil, repoErr)

	err := c.HandlePaymentFailed(ctx, body)

	assert.ErrorIs(t, err, repoErr)
	repo.AssertExpectations(t)
}
