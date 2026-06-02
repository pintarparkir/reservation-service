package grpcclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	userv1 "github.com/farid/reservation-service/api/proto/user/v1"
)

// userGrpc implements UserClient via gRPC to user-service.
type userGrpc struct {
	pb      userv1.UserServiceClient
	timeout time.Duration
}

// NewUserGrpc returns a UserClient backed by the real user-service gRPC.
func NewUserGrpc(addr string) (UserClient, error) {
	var creds grpc.DialOption
	if strings.HasSuffix(addr, ":443") {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}))
	} else {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	conn, err := grpc.NewClient(addr, creds,
		grpc.WithDefaultServiceConfig(serviceConfig),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc dial user-service %s: %w", addr, err)
	}
	return &userGrpc{pb: userv1.NewUserServiceClient(conn), timeout: 3 * time.Second}, nil
}

func (c *userGrpc) UpsertDriver(ctx context.Context, externalUserID, phone, fullName string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.pb.UpsertDriver(cctx, &userv1.UpsertDriverRequest{
		ExternalUserId: externalUserID,
		PhoneE164:      phone,
		FullName:       fullName,
	})
	if err != nil {
		return "", fmt.Errorf("user.UpsertDriver: %w", err)
	}
	return resp.GetId(), nil
}

func (c *userGrpc) GetMSISDN(ctx context.Context, driverID string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.pb.GetUserById(cctx, &userv1.GetUserByIdRequest{Id: driverID})
	if err != nil {
		return "", fmt.Errorf("user.GetUserById: %w", err)
	}
	return resp.GetPhoneE164(), nil
}

func (c *userGrpc) GetDefaultPaymentMethod(ctx context.Context, driverID string) (*PaymentMethod, error) {
	cctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp, err := c.pb.GetDefaultPaymentMethod(cctx, &userv1.GetDefaultPaymentMethodRequest{UserId: driverID})
	if err != nil {
		return nil, fmt.Errorf("user.GetDefaultPaymentMethod: %w", err)
	}
	return &PaymentMethod{
		Type:  resp.GetType(),
		Last4: resp.GetLast4(),
		Brand: resp.GetBrand(),
	}, nil
}
