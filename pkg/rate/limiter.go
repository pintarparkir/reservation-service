// Package rate implements a sliding-window rate limiter backed by Redis.
package rate

import (
	"context"
	"time"
)

// Store is the minimal set of Redis operations the limiter needs.
// pkg/redis.Collections satisfies this interface.
type Store interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// Limiter decides whether a request identified by key is within quota.
type Limiter interface {
	// Allow returns true if the request is allowed. When denied, retryAfter
	// indicates how long the caller should wait before retrying.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
}

type limiter struct {
	store Store
}

// New creates a Limiter backed by the given Store.
func New(store Store) Limiter {
	return &limiter{store: store}
}

func (l *limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	count, err := l.store.Incr(ctx, key)
	if err != nil {
		return false, 0, err
	}

	// First request in window — set TTL so the counter auto-expires.
	if count == 1 {
		if err := l.store.Expire(ctx, key, window); err != nil {
			return false, 0, err
		}
	}

	if int(count) > limit {
		ttl, err := l.store.TTL(ctx, key)
		if err != nil {
			return false, 0, err
		}
		if ttl <= 0 {
			ttl = window
		}
		return false, ttl, nil
	}

	return true, 0, nil
}
