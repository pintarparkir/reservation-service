package grpcclient

import (
	"context"

	"github.com/farid/reservation-service/pkg/logger"
)

// BillingClient is the slice of billing-service the reservation usecase needs.
// Implementations:
//   - billingStub      — for local dev / tests; logs the call and returns synthetic IDs.
//   - billingGrpc (TBD)— wraps the generated billingv1.BillingServiceClient. Lands once
//                        `buf generate` is wired into CI (see roadmap).
type BillingClient interface {
	OpenInvoice(ctx context.Context, reservationID, driverID, idempotencyKey string) (invoiceID string, err error)
	CloseInvoice(ctx context.Context, invoiceID string) error
}

// NewBillingStub returns a BillingClient that logs but never errors.
// Useful when running reservation-service before billing-service exists.
func NewBillingStub() BillingClient { return &billingStub{} }

type billingStub struct{}

func (s *billingStub) OpenInvoice(ctx context.Context, reservationID, driverID, idem string) (string, error) {
	logger.Info(ctx, "[billing-stub] OpenInvoice", map[string]interface{}{
		"reservation_id": reservationID,
		"driver_id":      driverID,
		"idem":           idem,
	})
	return "stub-invoice-" + reservationID, nil
}

func (s *billingStub) CloseInvoice(ctx context.Context, invoiceID string) error {
	logger.Info(ctx, "[billing-stub] CloseInvoice", map[string]interface{}{"invoice_id": invoiceID})
	return nil
}
