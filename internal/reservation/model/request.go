package model

const (
	ModeSystemAssigned = "SYSTEM_ASSIGNED"
	ModeUserSelected   = "USER_SELECTED"
)

// CreateReservationRequest is the usecase-level shape; both REST and gRPC
// handlers translate their wire format into this.
type CreateReservationRequest struct {
	DriverID        string
	VehicleType     VehicleType
	Mode            string
	PreferredSpotID string
	IdempotencyKey  string
}

// CheckInRequest carries GPS coords + the soft-fail flag for missing GPS.
type CheckInRequest struct {
	ID             string
	Latitude       float64
	Longitude      float64
	GPSUnavailable bool
}

// CancelRequest captures cancel reason for audit / pricing engine input.
type CancelRequest struct {
	ID     string
	Reason string
}
