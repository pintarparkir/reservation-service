// Package consumer provides RabbitMQ message handlers for billing events.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
)

type BillingPaymentConsumer struct {
	repo repository.ReservationRepository
}

func NewBillingPaymentConsumer(repo repository.ReservationRepository) *BillingPaymentConsumer {
	return &BillingPaymentConsumer{repo: repo}
}

// Handle routes messages to the appropriate handler based on the routing key.
// It satisfies the rabbit.Handler signature.
func (c *BillingPaymentConsumer) Handle(ctx context.Context, routingKey string, body []byte) error {
	switch routingKey {
	case model.EvtPaymentSuccess:
		return c.HandlePaymentConfirmed(ctx, body)
	case model.EvtPaymentFailed:
		return c.HandlePaymentFailed(ctx, body)
	default:
		return fmt.Errorf("unknown routing key: %s", routingKey)
	}
}

type paymentEvent struct {
	ReservationID string `json:"reservation_id"`
	PaymentRef    string `json:"payment_ref,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// HandlePaymentConfirmed processes billing.payment.success.v1 events
func (c *BillingPaymentConsumer) HandlePaymentConfirmed(ctx context.Context, body []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": ev.ReservationID})
	_, err := c.repo.ApplyTransition(ctx, ev.ReservationID, model.ActionPaymentSuccess, model.EvtReservationConfirmed, outboxPayload)
	return err
}

// HandlePaymentFailed processes billing.payment.failed.v1 events
func (c *BillingPaymentConsumer) HandlePaymentFailed(ctx context.Context, body []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": ev.ReservationID, "reason": ev.Reason})
	_, err := c.repo.ApplyTransition(ctx, ev.ReservationID, model.ActionPaymentFail, model.EvtReservationCancelled, outboxPayload)
	return err
}
