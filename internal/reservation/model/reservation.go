// Package model defines reservation-domain types. No I/O.
package model

import "time"

type ReservationState string

const (
	StatePending        ReservationState = "PENDING"
	StatePendingPayment ReservationState = "PENDING_PAYMENT"
	StateConfirmed      ReservationState = "CONFIRMED"
	StateActive         ReservationState = "ACTIVE"
	StateCompleted      ReservationState = "COMPLETED"
	StateCancelled      ReservationState = "CANCELLED"
	StateExpired        ReservationState = "EXPIRED"
)

type VehicleType string

const (
	VehicleTypeCar        VehicleType = "CAR"
	VehicleTypeMotorcycle VehicleType = "MOTORCYCLE"
)

type Reservation struct {
	ID             string
	DriverID       string
	SpotID         string
	VehicleType    VehicleType
	State          ReservationState
	HoldStart      time.Time
	HoldEnd        time.Time
	PaymentExpiresAt *time.Time
	ConfirmedAt    *time.Time
	CheckedInAt    *time.Time
	CheckedOutAt   *time.Time
	ExpiresAt      *time.Time
	IdempotencyKey string
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Spot struct {
	ID          string // e.g. "F2-C-014"
	Floor       int
	VehicleType VehicleType
	Status      string // AVAILABLE | OCCUPIED | OUT_OF_SERVICE
	Version     int
}
