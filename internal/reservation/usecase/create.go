package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
	apperror "github.com/farid/reservation-service/pkg/error"
)

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
	r := &model.Reservation{
		DriverID:       req.DriverID,
		SpotID:         spotID,
		VehicleType:    req.VehicleType,
		State:          model.StatePending,
		HoldStart:      now,
		HoldEnd:        end,
		ExpiresAt:      &end,
		IdempotencyKey: req.IdempotencyKey,
	}

	payload, _ := json.Marshal(map[string]any{
		"reservation_id": "<assigned-on-insert>",
		"driver_id":      req.DriverID,
		"spot_id":        spotID,
		"vehicle_type":   req.VehicleType,
		"hold_end":       end,
	})

	created, err := u.repo.Create(ctx, r, payload)
	if err != nil {
		return nil, err
	}

	// Open invoice — idempotency-keyed on the same client key. Stub today; real
	// gRPC client lands once billing-service ships (see ROADMAP.md).
	if _, err := u.billing.OpenInvoice(ctx, created.ID, req.DriverID, req.IdempotencyKey); err != nil {
		// We deliberately do NOT roll back the reservation: the outbox event will
		// trigger billing async via the publisher. Log + continue.
		// In the future, wrap this in a saga/compensating action.
		return created, nil
	}
	return created, nil
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
