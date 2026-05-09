// Package worker holds the background loops: no-show expirer + outbox publisher.
package worker

import (
	"context"
	"time"

	"github.com/farid/reservation-service/internal/reservation/repository"
	"github.com/farid/reservation-service/pkg/logger"
)

type NoShowExpirer struct {
	repo     repository.ReservationRepository
	interval time.Duration
	batch    int
}

func NewNoShowExpirer(repo repository.ReservationRepository) *NoShowExpirer {
	return &NoShowExpirer{repo: repo, interval: 30 * time.Second, batch: 100}
}

// Run blocks until ctx is cancelled. Safe to run with multiple replicas — the
// SQL uses FOR UPDATE SKIP LOCKED so each row is processed exactly once.
func (w *NoShowExpirer) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ids, err := w.repo.ExpireDueReservations(ctx, w.batch)
			if err != nil {
				logger.Error(ctx, "noshow expirer: tick failed",
					map[string]interface{}{logger.ErrorKey: err.Error()})
				continue
			}
			if len(ids) > 0 {
				logger.Info(ctx, "noshow expirer: flipped reservations",
					map[string]interface{}{"count": len(ids)})
			}
		}
	}
}
