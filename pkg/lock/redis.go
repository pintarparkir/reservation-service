// Package lock provides a Redis-based advisory lock.
// The DB EXCLUDE constraint on reservation is the authoritative double-book guard;
// this lock just gives us a fast contention signal so we can return 429 quickly.
package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	apperror "github.com/farid/reservation-service/pkg/error"
	"github.com/farid/reservation-service/pkg/redis"
)

// Token uniquely identifies an acquired lock — required for safe Release.
type Token string

const releaseScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

type Lock struct {
	cache redis.Collections
}

func New(c redis.Collections) *Lock { return &Lock{cache: c} }

// Acquire returns a Token if the lock was set; otherwise apperror.ErrLockUnavailable.
func (l *Lock) Acquire(ctx context.Context, key string, ttl time.Duration) (Token, error) {
	tok := newToken()
	ok, err := l.cache.SetNX(ctx, key, string(tok), ttl)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", apperror.ErrLockUnavailable
	}
	return tok, nil
}

// Release deletes the lock iff we still hold the same token.
func (l *Lock) Release(ctx context.Context, key string, tok Token) error {
	_, err := l.cache.Eval(ctx, releaseScript, []string{key}, string(tok))
	return err
}

func newToken() Token {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return Token(hex.EncodeToString(b[:]))
}
