package http

import (
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
)

type reservationDTO struct {
	ID           string     `json:"id"`
	DriverID     string     `json:"driver_id"`
	SpotID       string     `json:"spot_id"`
	VehicleType  string     `json:"vehicle_type"`
	State        string     `json:"state"`
	HoldStart    time.Time  `json:"hold_start"`
	HoldEnd      time.Time  `json:"hold_end"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`
	CheckedInAt  *time.Time `json:"checked_in_at,omitempty"`
	CheckedOutAt *time.Time `json:"checked_out_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Version      int        `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
}

func toDTO(r *model.Reservation) *reservationDTO {
	if r == nil {
		return nil
	}
	return &reservationDTO{
		ID: r.ID, DriverID: r.DriverID, SpotID: r.SpotID,
		VehicleType: string(r.VehicleType),
		State:       string(r.State),
		HoldStart:   r.HoldStart, HoldEnd: r.HoldEnd,
		ConfirmedAt: r.ConfirmedAt, CheckedInAt: r.CheckedInAt,
		CheckedOutAt: r.CheckedOutAt, ExpiresAt: r.ExpiresAt,
		Version:   r.Version,
		CreatedAt: r.CreatedAt,
	}
}

type createReq struct {
	VehicleType     string `json:"vehicle_type" binding:"required"`
	Mode            string `json:"mode"          binding:"required"`
	PreferredSpotID string `json:"preferred_spot_id,omitempty"`
}

type cancelReq struct {
	Reason string `json:"reason"`
}

type checkInReq struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	GPSUnavailable bool    `json:"gps_unavailable,omitempty"`
}

type availabilityResp struct {
	AvailableCount int       `json:"available_count"`
	ByFloor        []byFloor `json:"by_floor"`
}

type byFloor struct {
	Floor int `json:"floor"`
	Count int `json:"count"`
}
