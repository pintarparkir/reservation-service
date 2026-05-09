package usecase

import (
	"context"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
)

func (u *reservationUsecase) Availability(ctx context.Context, vt model.VehicleType) ([]repository.FloorCount, int, error) {
	if !model.IsValidVehicleType(vt) {
		return nil, 0, nil // empty result, not an error
	}
	return u.spotRepo.AvailabilityByFloor(ctx, vt)
}
