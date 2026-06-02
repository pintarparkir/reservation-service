package grpcclient

import "context"

// UserClient defines the contract for calling user-service.
type UserClient interface {
	// UpsertDriver creates or fetches the driver by external_user_id, returns internal UUID.
	UpsertDriver(ctx context.Context, externalUserID, phone, fullName string) (string, error)
	GetMSISDN(ctx context.Context, driverID string) (string, error)
	GetDefaultPaymentMethod(ctx context.Context, driverID string) (*PaymentMethod, error)
}

type PaymentMethod struct {
	Type    string
	CCToken string
	Last4   string
	Brand   string
}
