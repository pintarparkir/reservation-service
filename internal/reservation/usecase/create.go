package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
	apperror "github.com/farid/reservation-service/pkg/error"
)

// Create persists the reservation and emits `reservation.created.v1` to the
// outbox in a single transaction. Billing is opened asynchronously by the
// billing-service consumer on that event (idempotency_key = reservation_id).
func (u *reservationUsecase) Create(ctx context.Context, req model.CreateReservationRequest) (*model.Reservation, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}

	// Idempotency check at the row level (gRPC interceptor handles the wire-level
	// replay; this layer also de-dupes when a stale interceptor cache misses but
	// the original row is still in DB).
	if existing, err := u.repo.FindByIdempotencyKey(ctx, req.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	preferred := strings.TrimSpace(req.PreferredSpotID)
	if req.Mode == model.ModeUserSelected && preferred == "" {
		return nil, &apperror.AppError{Code: "VALIDATION", Message: "preferred_spot_id required for USER_SELECTED"}
	}
	if req.Mode == model.ModeSystemAssigned {
		preferred = ""
	}

	spotID, err := u.spotRepo.Assign(ctx, req.VehicleType, preferred)
	if err != nil {
		return nil, err
	}

	// Defence-in-depth Redis lock; the EXCLUDE constraint is the authoritative guard.
	tok, err := u.lock.Acquire(ctx, "spot:"+spotID, 30*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = u.lock.Release(ctx, "spot:"+spotID, tok) }()

	now := time.Now().UTC()
	end := now.Add(u.cfg.HoldDuration)
	return u.repo.Create(ctx, &model.Reservation{
		DriverID:       req.DriverID,
		SpotID:         spotID,
		VehicleType:    req.VehicleType,
		State:          model.StatePending,
		HoldStart:      now,
		HoldEnd:        end,
		ExpiresAt:      &end,
		IdempotencyKey: req.IdempotencyKey,
	})
}

func validateCreate(r model.CreateReservationRequest) error {
	switch {
	case strings.TrimSpace(r.DriverID) == "":
		return &apperror.AppError{Code: "VALIDATION", Message: "driver_id required"}
	case !model.IsValidVehicleType(r.VehicleType):
		return &apperror.AppError{Code: "VALIDATION", Message: "vehicle_type must be CAR or MOTORCYCLE"}
	case r.Mode != model.ModeSystemAssigned && r.Mode != model.ModeUserSelected:
		return &apperror.AppError{Code: "VALIDATION", Message: "mode must be SYSTEM_ASSIGNED or USER_SELECTED"}
	}
	return nil
}
