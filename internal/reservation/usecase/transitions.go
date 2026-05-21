package usecase

import (
	"context"
	"encoding/json"

	"github.com/farid/reservation-service/internal/reservation/model"
	apperror "github.com/farid/reservation-service/pkg/error"
	"github.com/farid/reservation-service/pkg/geo"
)

func (u *reservationUsecase) Confirm(ctx context.Context, id string) (*model.Reservation, error) {
	r, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	payloadMap := map[string]any{
		"reservation_id": id,
		"driver_id":      r.DriverID,
	}
	if msisdn := u.lookupMSISDN(ctx, r.DriverID); msisdn != "" {
		payloadMap["msisdn"] = msisdn
	}
	payload, _ := json.Marshal(payloadMap)
	return u.repo.ApplyTransition(ctx, id, model.ActionConfirm, model.EvtReservationConfirmed, payload)
}

func (u *reservationUsecase) Cancel(ctx context.Context, req model.CancelRequest) (*model.Reservation, error) {
	r, err := u.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	payloadMap := map[string]any{
		"reservation_id": req.ID,
		"driver_id":      r.DriverID,
		"reason":         req.Reason,
	}
	if msisdn := u.lookupMSISDN(ctx, r.DriverID); msisdn != "" {
		payloadMap["msisdn"] = msisdn
	}
	payload, _ := json.Marshal(payloadMap)
	return u.repo.ApplyTransition(ctx, req.ID, model.ActionCancel, model.EvtReservationCancelled, payload)
}

func (u *reservationUsecase) CheckIn(ctx context.Context, req model.CheckInRequest) (*model.Reservation, error) {
	if !req.GPSUnavailable {
		dist := geo.Haversine(req.Latitude, req.Longitude, u.cfg.BuildingLat, u.cfg.BuildingLng)
		if dist > u.cfg.GeofenceRadiusMeters {
			return nil, &apperror.AppError{
				Code:    "GEOFENCE_VIOLATION",
				Message: "outside permitted radius",
			}
		}
	}
	payload, _ := json.Marshal(map[string]any{
		"reservation_id":  req.ID,
		"gps_unavailable": req.GPSUnavailable,
	})
	return u.repo.ApplyTransition(ctx, req.ID, model.ActionCheckIn, model.EvtReservationCheckedIn, payload)
}

func (u *reservationUsecase) CheckOut(ctx context.Context, id string) (*model.Reservation, error) {
	r, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Embed timestamps in the outbox event so billing's RabbitMQ consumer can
	// run the pricing engine with full session info. The consumer dispatches
	// CloseInvoice on `reservation.checked_out.v1`; we don't call billing
	// synchronously here — keeps the check-out path fast and decouples it
	// from billing availability.
	payload, _ := json.Marshal(map[string]any{
		"reservation_id": id,
		"confirmed_at":   r.ConfirmedAt,
		"checked_in_at":  r.CheckedInAt,
		// checked_out_at is set by the transition itself; consumer can
		// recompute or read from a future GetReservation call.
	})
	return u.repo.ApplyTransition(ctx, id, model.ActionCheckOut, model.EvtReservationCheckedOut, payload)
}

func (u *reservationUsecase) Get(ctx context.Context, id string) (*model.Reservation, error) {
	return u.repo.GetByID(ctx, id)
}
