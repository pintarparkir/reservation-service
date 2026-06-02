// Package model defines reservation domain models and constants.
package model

// Idempotent gRPC method full names used by the idempotency interceptor.
// Keep in sync with proto definitions in api/proto/reservation/v1/reservation.proto.
const (
	ScopeCreateReservation  = "/parkirpintar.reservation.v1.ReservationService/CreateReservation"
	ScopeConfirmReservation = "/parkirpintar.reservation.v1.ReservationService/ConfirmReservation"
	ScopeCancelReservation  = "/parkirpintar.reservation.v1.ReservationService/CancelReservation"
	ScopeCheckIn            = "/parkirpintar.reservation.v1.ReservationService/CheckIn"
	ScopeCheckOut           = "/parkirpintar.reservation.v1.ReservationService/CheckOut"
)

// Routing keys we publish on parkirpintar.events.
const (
	EvtReservationCreated        = "reservation.created.v1"
	EvtReservationPaymentPending = "reservation.payment_pending.v1"
	EvtReservationConfirmed      = "reservation.confirmed.v1"
	EvtReservationCancelled      = "reservation.cancelled.v1"
	EvtReservationExpired        = "reservation.expired.v1"
	EvtReservationCheckedIn      = "reservation.checked_in.v1"
	EvtReservationCheckedOut     = "reservation.checked_out.v1"
)

// Routing keys we consume from billing-service events.
const (
	EvtPaymentSuccess = "payment.paid.v1"
	EvtPaymentFailed  = "payment.failed.v1"
)

func IsValidVehicleType(vt VehicleType) bool {
	return vt == VehicleTypeCar || vt == VehicleTypeMotorcycle
}
