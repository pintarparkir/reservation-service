// Package consumer provides RabbitMQ message handlers for billing events.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
)

type BillingPaymentConsumer struct {
	repo       repository.ReservationRepository
	billingURL string
}

func NewBillingPaymentConsumer(repo repository.ReservationRepository, billingURL string) *BillingPaymentConsumer {
	return &BillingPaymentConsumer{repo: repo, billingURL: billingURL}
}

// Handle routes messages to the appropriate handler based on the routing key.
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
	OrderID       string `json:"order_id"`
	PgReference   string `json:"pg_reference,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func (c *BillingPaymentConsumer) HandlePaymentConfirmed(ctx context.Context, body []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	resID := ev.ReservationID
	if resID == "" && ev.OrderID != "" {
		// order_id is invoice_id — resolve reservation_id via billing
		resID = c.resolveReservationID(ctx, ev.OrderID)
	}
	if resID == "" {
		return fmt.Errorf("cannot resolve reservation_id from event: %s", string(body))
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": resID})
	_, err := c.repo.ApplyTransition(ctx, resID, model.ActionPaymentSuccess, model.EvtReservationConfirmed, outboxPayload)
	return err
}

func (c *BillingPaymentConsumer) HandlePaymentFailed(ctx context.Context, body []byte) error {
	var ev paymentEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	resID := ev.ReservationID
	if resID == "" && ev.OrderID != "" {
		resID = c.resolveReservationID(ctx, ev.OrderID)
	}
	if resID == "" {
		return fmt.Errorf("cannot resolve reservation_id from event: %s", string(body))
	}
	outboxPayload, _ := json.Marshal(map[string]any{"reservation_id": resID, "reason": ev.Reason})
	_, err := c.repo.ApplyTransition(ctx, resID, model.ActionPaymentFail, model.EvtReservationCancelled, outboxPayload)
	return err
}

// resolveReservationID calls billing-service to get reservation_id from invoice_id.
func (c *BillingPaymentConsumer) resolveReservationID(ctx context.Context, invoiceID string) string {
	if c.billingURL == "" {
		return ""
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/v1/invoices/%s", c.billingURL, invoiceID))
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	var inv struct {
		ReservationID string `json:"reservation_id"`
	}
	if json.Unmarshal(body, &inv) == nil {
		return inv.ReservationID
	}
	return ""
}
