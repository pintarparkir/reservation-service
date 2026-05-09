package usecase

import (
	"context"
	"encoding/json"

	"github.com/farid/reservation-service/internal/reservation/model"
	apperror "github.com/farid/reservation-service/pkg/error"
	"github.com/farid/reservation-service/pkg/geo"
)

func (u *reservationUsecase) Confirm(ctx context.Context, id string) (*model.Reservation, error) {
	payload, _ := json.Marshal(map[string]any{"reservation_id": id})
	return u.repo.ApplyTransition(ctx, id, model.ActionConfirm, model.EvtReservationConfirmed, payload)
}

func (u *reservationUsecase) Cancel(ctx context.Context, req model.CancelRequest) (*model.Reservation, error) {
	payload, _ := json.Marshal(map[string]any{
		"reservation_id": req.ID,
		"reason":         req.Reason,
	})
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
		"reservation_id":   req.ID,
		"gps_unavailable":  req.GPSUnavailable,
	})
	return u.repo.ApplyTransition(ctx, req.ID, model.ActionCheckIn, model.EvtReservationCheckedIn, payload)
}

func (u *reservationUsecase) CheckOut(ctx context.Context, id string) (*model.Reservation, error) {
	payload, _ := json.Marshal(map[string]any{"reservation_id": id})
	r, err := u.repo.ApplyTransition(ctx, id, model.ActionCheckOut, model.EvtReservationCheckedOut, payload)
	if err != nil {
		return nil, err
	}

	// Trigger billing close. Same logging-only failure mode as Create.
	_ = u.billing.CloseInvoice(ctx, "stub-invoice-"+id)
	return r, nil
}

func (u *reservationUsecase) Get(ctx context.Context, id string) (*model.Reservation, error) {
	return u.repo.GetByID(ctx, id)
}
