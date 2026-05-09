package model

// Idempotent gRPC method full names used by the idempotency interceptor.
// Keep in sync with proto definitions in api/proto/reservation/v1/reservation.proto.
const (
	SCOPE_CREATE_RESERVATION  = "/parkirpintar.reservation.v1.ReservationService/CreateReservation"
	SCOPE_CONFIRM_RESERVATION = "/parkirpintar.reservation.v1.ReservationService/ConfirmReservation"
	SCOPE_CANCEL_RESERVATION  = "/parkirpintar.reservation.v1.ReservationService/CancelReservation"
	SCOPE_CHECK_IN            = "/parkirpintar.reservation.v1.ReservationService/CheckIn"
	SCOPE_CHECK_OUT           = "/parkirpintar.reservation.v1.ReservationService/CheckOut"
)

// Routing keys we publish on parkirpintar.events.
const (
	EvtReservationCreated   = "reservation.created.v1"
	EvtReservationConfirmed = "reservation.confirmed.v1"
	EvtReservationCancelled = "reservation.cancelled.v1"
	EvtReservationExpired   = "reservation.expired.v1"
	EvtReservationCheckedIn = "reservation.checked_in.v1"
	EvtReservationCheckedOut = "reservation.checked_out.v1"
)

func IsValidVehicleType(vt VehicleType) bool {
	return vt == VehicleTypeCar || vt == VehicleTypeMotorcycle
}
