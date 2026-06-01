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
}

func (c *BillingPaymentConsumer) HandlePaymentConfirmed(ctx context.Context, payload []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return err
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": ev.ReservationID})
	_, err := c.repo.ApplyTransition(ctx, ev.ReservationID, model.ActionPaymentSuccess, model.EvtReservationConfirmed, outboxPayload)
	return err
}

func (c *BillingPaymentConsumer) HandlePaymentFailed(ctx context.Context, payload []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return err
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": ev.ReservationID, "reason": "payment_failed"})
	_, err := c.repo.ApplyTransition(ctx, ev.ReservationID, model.ActionPaymentFail, model.EvtReservationCancelled, outboxPayload)
	return err
}
