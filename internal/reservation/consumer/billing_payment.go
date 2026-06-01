package consumer

import (
	"context"
	"encoding/json"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
)

type BillingPaymentConsumer struct {
	repo repository.ReservationRepository
}

func NewBillingPaymentConsumer(repo repository.ReservationRepository) *BillingPaymentConsumer {
	return &BillingPaymentConsumer{repo: repo}
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
