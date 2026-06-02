package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

type PaymentTimeoutWorker struct {
	db       *sqlx.DB
	interval time.Duration
}

func NewPaymentTimeoutWorker(db *sqlx.DB, interval time.Duration) *PaymentTimeoutWorker {
	return &PaymentTimeoutWorker{db: db, interval: interval}
}

func (w *PaymentTimeoutWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = w.ExpireBatch(ctx)
		}
	}
}

func (w *PaymentTimeoutWorker) ExpireBatch(ctx context.Context) error {
	tx, err := w.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryxContext(ctx, `
		SELECT id
		FROM reservation
		WHERE state = 'PENDING_PAYMENT'
		  AND payment_expires_at < now()
		FOR UPDATE SKIP LOCKED
		LIMIT 100
	`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE reservation
			SET state='CANCELLED', version=version+1, updated_at=now()
			WHERE id=$1 AND state='PENDING_PAYMENT'
		`, id); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"reservation_id": id, "reason": "payment_timeout"})
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO outbox_event (aggregate_id, event_type, payload, created_at)
			VALUES ($1, 'reservation.cancelled.v1', $2, now())
		`, id, payload); err != nil {
			return err
		}
	}
	return tx.Commit()
}
