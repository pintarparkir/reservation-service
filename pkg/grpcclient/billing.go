package grpcclient

import (
	"context"

	"github.com/farid/reservation-service/pkg/logger"
)

// BillingClient is the slice of billing-service the reservation usecase needs.
// Implementations:
//   - billingStub      — for local dev / tests; logs the call and returns synthetic IDs.
//   - billingGrpc (TBD)— wraps the generated billingv1.BillingServiceClient. Lands once
//     `buf generate` is wired into CI (see roadmap).
type BillingClient interface {
	OpenInvoice(ctx context.Context, reservationID, driverID, idempotencyKey string) (invoiceID string, err error)
	CloseInvoice(ctx context.Context, invoiceID string) error
	CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (*PaymentRequest, error)
}

type CreatePaymentRequest struct {
	ReservationID string
	DriverID      string
	AmountIDR     int64
	Method        string
	CCToken       string
}

type PaymentRequest struct {
	ID        string
	Method    string
	Status    string
	QRISURL   string
	ExpiresAt int64
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

func (s *billingStub) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (*PaymentRequest, error) {
	logger.Info(ctx, "[billing-stub] CreatePaymentRequest", map[string]interface{}{
		"reservation_id": req.ReservationID,
		"driver_id":      req.DriverID,
		"amount_idr":     req.AmountIDR,
		"method":         req.Method,
	})
	return &PaymentRequest{
		ID:        "stub-payment-" + req.ReservationID,
		Method:    req.Method,
		Status:    "PENDING",
		QRISURL:   "https://qris.stub/pay/" + req.ReservationID,
		ExpiresAt: 0,
	}, nil
}
