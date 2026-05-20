// Package repository defines persistence contracts for reservation-service.
// Concrete implementations live under repository/postgres/.
package repository

import (
	"context"

	"github.com/farid/reservation-service/internal/reservation/model"
)

// FloorCount is the row shape for availability aggregations.
type FloorCount struct {
	Floor int `db:"floor"`
	Count int `db:"count"`
}

// ReservationRepository persists Reservation aggregates.
// Insert + outbox append happen in the same TX inside the postgres impl.
type ReservationRepository interface {
	// Create inserts the reservation and a `reservation.created.v1` outbox row
	// in a single transaction. The outbox payload is built from the inserted
	// row inside the SQL CTE so it always carries the real reservation_id.
	Create(ctx context.Context, r *model.Reservation) (*model.Reservation, error)
	GetByID(ctx context.Context, id string) (*model.Reservation, error)
	// ApplyTransition flips state with optimistic-lock + outbox append, all in one tx.
	ApplyTransition(ctx context.Context, id string, action model.Action, eventType string, eventPayload []byte) (*model.Reservation, error)
	// ExpireDueReservations is the worker entry point — returns the IDs that were flipped.
	ExpireDueReservations(ctx context.Context, limit int) ([]string, error)
	// FindByIdempotencyKey returns the original reservation for a replay; nil/nil if none.
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error)
}

// SpotRepository owns the spot inventory.
type SpotRepository interface {
	// Assign returns the assigned spot id under FOR UPDATE SKIP LOCKED.
	Assign(ctx context.Context, vt model.VehicleType, preferred string) (string, error)
	AvailabilityByFloor(ctx context.Context, vt model.VehicleType) ([]FloorCount, int, error)
}

// OutboxRepository drives the publisher worker.
type OutboxRepository interface {
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxRow, error)
	MarkPublished(ctx context.Context, ids []int64) error
}

type OutboxRow struct {
	ID         int64  `db:"id"`
	EventType  string `db:"event_type"`
	Payload    []byte `db:"payload"`
}
