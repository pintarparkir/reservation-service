package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubUserClient struct {
	msisdn string
	err    error
}

func (s stubUserClient) GetMSISDN(context.Context, string) (string, error) {
	return s.msisdn, s.err
}

func TestReservationUsecase_WithUserClient_ReturnsSameUsecase(t *testing.T) {
	u := &reservationUsecase{}
	users := stubUserClient{msisdn: "628123"}

	got := u.WithUserClient(users)

	assert.Same(t, u, got)
	assert.Equal(t, "628123", u.lookupMSISDN(context.Background(), "driver-1"))
}

func TestReservationUsecase_lookupMSISDN_EmptyWithoutClientOrDriver(t *testing.T) {
	u := &reservationUsecase{}

	assert.Empty(t, u.lookupMSISDN(context.Background(), "driver-1"))

	u.users = stubUserClient{msisdn: "628123"}
	assert.Empty(t, u.lookupMSISDN(context.Background(), ""))
}

func TestReservationUsecase_lookupMSISDN_TrimsWhitespace(t *testing.T) {
	u := &reservationUsecase{users: stubUserClient{msisdn: "  628123  "}}

	got := u.lookupMSISDN(context.Background(), "driver-1")

	assert.Equal(t, "628123", got)
}

func TestReservationUsecase_lookupMSISDN_EmptyOnClientError(t *testing.T) {
	u := &reservationUsecase{users: stubUserClient{err: errors.New("unavailable")}}

	got := u.lookupMSISDN(context.Background(), "driver-1")

	assert.Empty(t, got)
}
