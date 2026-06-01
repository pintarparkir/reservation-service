package grpcclient

import "context"

type UserClient interface {
	GetMSISDN(ctx context.Context, driverID string) (string, error)
	GetDefaultPaymentMethod(ctx context.Context, driverID string) (*PaymentMethod, error)
}

type PaymentMethod struct {
	Type    string
	CCToken string
	Last4   string
	Brand   string
}
