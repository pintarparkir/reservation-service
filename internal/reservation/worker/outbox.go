package worker

import (
	"context"
	"time"

	"github.com/farid/reservation-service/internal/reservation/repository"
	"github.com/farid/reservation-service/pkg/logger"
	"github.com/farid/reservation-service/pkg/rabbit"
)

type OutboxPublisher struct {
	repo      repository.OutboxRepository
	publisher *rabbit.Publisher
	interval  time.Duration
	batch     int
}

func NewOutboxPublisher(repo repository.OutboxRepository, p *rabbit.Publisher) *OutboxPublisher {
	return &OutboxPublisher{repo: repo, publisher: p, interval: time.Second, batch: 200}
}

// Run continuously drains outbox_event into RabbitMQ. Per tick:
//  1. SELECT unpublished rows with FOR UPDATE SKIP LOCKED.
//  2. Publish each → ack from broker.
//  3. UPDATE published_at on the successfully-published rows.
//
// Crash mid-loop = safe (rows still NULL, next tick republishes — at-least-once).
func (w *OutboxPublisher) Run(ctx context.Context) {
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.tick(ctx)
		}
	}
}

func (w *OutboxPublisher) tick(ctx context.Context) {
	rows, err := w.repo.FetchUnpublished(ctx, w.batch)
	if err != nil {
		logger.Error(ctx, "outbox publisher: fetch failed",
			map[string]interface{}{logger.ErrorKey: err.Error()})
		return
	}
	if len(rows) == 0 {
		return
	}
	var published []int64
	for _, r := range rows {
		if err := w.publisher.Publish(ctx, r.EventType, r.Payload); err != nil {
			logger.Error(ctx, "outbox publisher: publish failed",
				map[string]interface{}{
					"id":            r.ID,
					"event_type":    r.EventType,
					logger.ErrorKey: err.Error(),
				})
			break // stop the batch; retry next tick
		}
		published = append(published, r.ID)
	}
	if err := w.repo.MarkPublished(ctx, published); err != nil {
		logger.Error(ctx, "outbox publisher: mark failed",
			map[string]interface{}{logger.ErrorKey: err.Error()})
	}
	logger.Info(ctx, "outbox publisher: tick", map[string]interface{}{
		"fetched":   len(rows),
		"published": len(published),
	})
}
