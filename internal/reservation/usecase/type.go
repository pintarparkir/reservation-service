// Package usecase orchestrates reservation business logic.
package usecase

import (
	"context"
	"time"

	"github.com/farid/reservation-service/internal/reservation/model"
	"github.com/farid/reservation-service/internal/reservation/repository"
	"github.com/farid/reservation-service/pkg/lock"
)

type ReservationUsecase interface {
	Availability(ctx context.Context, vt model.VehicleType) ([]repository.FloorCount, int, error)
	Create(ctx context.Context, req model.CreateReservationRequest) (*model.Reservation, error)
	Confirm(ctx context.Context, id string) (*model.Reservation, error)
	Cancel(ctx context.Context, req model.CancelRequest) (*model.Reservation, error)
	CheckIn(ctx context.Context, req model.CheckInRequest) (*model.Reservation, error)
	CheckOut(ctx context.Context, id string) (*model.Reservation, error)
	Get(ctx context.Context, id string) (*model.Reservation, error)
}

type reservationUsecase struct {
	repo     repository.ReservationRepository
	spotRepo repository.SpotRepository
	lock     *lock.Lock
	cfg      Config
}

// Config carries the runtime knobs the usecase needs at boot.
type Config struct {
	HoldDuration         time.Duration
	GeofenceRadiusMeters float64
	BuildingLat          float64
	BuildingLng          float64
}

// NewReservationUsecase wires the usecase struct.
//
// Billing is reached asynchronously: Create only inserts the reservation +
// `reservation.created.v1` outbox row; billing-service consumes the event and
// opens the invoice idempotently. See docs/architecture/service-communication.
func NewReservationUsecase(
	repo repository.ReservationRepository,
	spotRepo repository.SpotRepository,
	l *lock.Lock,
	cfg Config,
) ReservationUsecase {
	return &reservationUsecase{
		repo: repo, spotRepo: spotRepo, lock: l,
		cfg: cfg,
	}
}
