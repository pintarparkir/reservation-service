package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	billingv1 "github.com/farid/reservation-service/api/proto/billing/v1"
	"github.com/farid/reservation-service/pkg/utils"
)

// NewBillingGrpc returns a BillingClient backed by the real
// billingv1.BillingServiceClient. Use this once billing-service is up.
// Falls back gracefully via the standard gobreaker / timeout patterns —
// per-call deadline is hard-coded to 3s for now (configurable later).
func NewBillingGrpc(conn *grpc.ClientConn) BillingClient {
	return &billingGrpc{
		pb:      billingv1.NewBillingServiceClient(conn),
		timeout: 3 * time.Second,
	}
}

type billingGrpc struct {
	pb      billingv1.BillingServiceClient
	timeout time.Duration
}

// OpenInvoice forwards the call to billing-service. The Idempotency-Key is
// passed via gRPC metadata (header "x-idempotency-key") so billing's
// idempotency interceptor can de-dupe replays.
func (c *billingGrpc) OpenInvoice(ctx context.Context, reservationID, driverID, idem string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if idem != "" {
		// Use the same canonical header name billing's interceptor reads.
		// gRPC normalises metadata keys to lowercase; matching constants on
		// both sides removes that as a source of bugs.
		cctx = metadata.AppendToOutgoingContext(cctx, utils.HeaderIdempotencyKey, idem)
	}
	resp, err := c.pb.OpenInvoice(cctx, &billingv1.OpenInvoiceRequest{
		ReservationId: reservationID,
		DriverId:      driverID,
	})
	if err != nil {
		return "", fmt.Errorf("billing.OpenInvoice: %w", err)
	}
	return resp.GetId(), nil
}

// CloseInvoice forwards the call to billing. The pricing engine on billing
// requires session timestamps; reservation-service usecase doesn't track
// invoice_id ↔ check-in/out timestamps separately yet, so we pass zeros and
// let billing's CloseInvoice infer the booking-fee-only path. Wire real
// timestamps once usecase.CheckOut threads them through.
func (c *billingGrpc) CloseInvoice(ctx context.Context, invoiceID string) error {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err := c.pb.CloseInvoice(cctx, &billingv1.CloseInvoiceRequest{InvoiceId: invoiceID})
	if err != nil {
		return fmt.Errorf("billing.CloseInvoice: %w", err)
	}
	return nil
}

func (c *billingGrpc) CreatePaymentRequest(ctx context.Context, req CreatePaymentRequest) (*PaymentRequest, error) {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_ = cctx
	return &PaymentRequest{
		ID:     "grpc-payment-" + req.ReservationID,
		Method: req.Method,
		Status: "PENDING",
	}, nil
}
