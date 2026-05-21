package worker

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/farid/reservation-service/pkg/logger"
)

// Reconciler scans reservations older than threshold that lack a
// reservation.created.v1 outbox event, and republishes the event.
// This recovers from missed event publishing due to crashes or bugs.
type Reconciler struct {
	db        *sqlx.DB
	interval  time.Duration
	threshold time.Duration
	batch     int
}

func NewReconciler(db *sqlx.DB) *Reconciler {
	return &Reconciler{
		db:        db,
		interval:  5 * time.Minute,
		threshold: 5 * time.Minute,
		batch:     100,
	}
}

func (r *Reconciler) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.tick(ctx)
		}
	}
}

// tick finds reservations older than threshold without reservation.created.v1
// outbox event, and inserts the missing event. The outbox publisher worker
// will drain it to RabbitMQ on its next tick.
func (r *Reconciler) tick(ctx context.Context) {
	const query = `
	WITH missing AS (
	  SELECT r.id, r.driver_id, r.spot_id, r.vehicle_type,
	         to_char(upper(r.hold_window) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS hold_end
	    FROM reservation r
	   WHERE r.created_at < now() - $1::interval
	     AND NOT EXISTS (
	           SELECT 1 FROM outbox_event o
	            WHERE o.aggregate_id = r.id
	              AND o.event_type = 'reservation.created.v1'
	         )
	   ORDER BY r.created_at
	   LIMIT $2
	)
	INSERT INTO outbox_event (aggregate_type, aggregate_id, event_type, payload)
	SELECT 'reservation', m.id, 'reservation.created.v1',
	       jsonb_build_object(
	         'reservation_id', m.id,
	         'driver_id',      m.driver_id,
	         'spot_id',        m.spot_id,
	         'vehicle_type',   m.vehicle_type,
	         'hold_end',       m.hold_end
	       )
	  FROM missing m
	RETURNING aggregate_id
	`

	rows, err := r.db.QueryxContext(ctx, query, r.threshold, r.batch)
	if err != nil {
		logger.Error(ctx, "reconcile: query failed", map[string]interface{}{logger.ErrorKey: err.Error()})
		return
	}
	defer func() { _ = rows.Close() }()

	var count int
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			logger.Error(ctx, "reconcile: scan failed", map[string]interface{}{logger.ErrorKey: err.Error()})
			continue
		}
		count++
	}

	if count > 0 {
		logger.Warn(ctx, "reconcile: re-published reservation.created.v1 events", map[string]interface{}{
			"count": count,
		})
	}
}
