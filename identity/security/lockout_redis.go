package security

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLockoutStore is a Redis-backed implementation of LockoutStore.
// Suitable for distributed deployments.
type RedisLockoutStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

// RedisLockoutOption configures RedisLockoutStore.
type RedisLockoutOption func(*RedisLockoutStore)

// WithLockoutKeyPrefix sets a prefix for all lockout keys in Redis.
func WithLockoutKeyPrefix(prefix string) RedisLockoutOption {
	return func(r *RedisLockoutStore) {
		r.keyPrefix = prefix
	}
}

// NewRedisLockoutStore creates a new Redis-backed lockout store.
func NewRedisLockoutStore(client redis.UniversalClient, opts ...RedisLockoutOption) *RedisLockoutStore {
	r := &RedisLockoutStore{
		client:    client,
		keyPrefix: "lockout:",
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *RedisLockoutStore) attemptsKey(identifier string) string {
	return r.keyPrefix + "attempts:" + identifier
}

func (r *RedisLockoutStore) lockKey(identifier string) string {
	return r.keyPrefix + "lock:" + identifier
}

// RecordAttempt implements LockoutStore.
func (r *RedisLockoutStore) RecordAttempt(ctx context.Context, identifier string, success bool) error {
	if success {
		return nil
	}

	key := r.attemptsKey(identifier)
	now := time.Now()
	nowMs := now.UnixMilli()

	pipe := r.client.Pipeline()

	// Add attempt with timestamp as score
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(nowMs),
		Member: fmt.Sprintf("%d:%d", nowMs, now.UnixNano()),
	})

	// Set expiration (1 hour is generous, will be cleaned by score)
	pipe.Expire(ctx, key, time.Hour)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// GetStatus implements LockoutStore.
func (r *RedisLockoutStore) GetStatus(ctx context.Context, identifier string, cfg LockoutConfig) (LockoutStatus, error) {
	now := time.Now()
	status := LockoutStatus{}

	// Check explicit lock
	lockKey := r.lockKey(identifier)
	lockUntilStr, err := r.client.Get(ctx, lockKey).Result()
	if err == nil {
		lockUntilMs, _ := strconv.ParseInt(lockUntilStr, 10, 64)
		lockUntil := time.UnixMilli(lockUntilMs)
		if now.Before(lockUntil) {
			status.IsLocked = true
			status.LockedUntil = lockUntil
		}
	} else if err != redis.Nil {
		return status, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	// Count recent failed attempts
	attemptsKey := r.attemptsKey(identifier)
	windowStart := now.Add(-cfg.AttemptWindow).UnixMilli()

	// Remove old attempts and count remaining
	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, attemptsKey, "-inf", strconv.FormatInt(windowStart, 10))
	countCmd := pipe.ZCard(ctx, attemptsKey)
	latestCmd := pipe.ZRevRangeWithScores(ctx, attemptsKey, 0, 0)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return status, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	status.FailedAttempts = int(countCmd.Val())
	status.RemainingAttempts = cfg.MaxAttempts - status.FailedAttempts
	if status.RemainingAttempts < 0 {
		status.RemainingAttempts = 0
	}

	// Get last attempt time
	latest := latestCmd.Val()
	if len(latest) > 0 {
		status.LastAttempt = time.UnixMilli(int64(latest[0].Score))
	}

	// Check if should be locked due to attempts
	if status.FailedAttempts >= cfg.MaxAttempts && !status.IsLocked {
		status.IsLocked = true
		status.LockedUntil = status.LastAttempt.Add(cfg.LockoutDuration)
		if now.After(status.LockedUntil) {
			status.IsLocked = false
			status.LockedUntil = time.Time{}
		}
	}

	return status, nil
}

// Lock implements LockoutStore.
func (r *RedisLockoutStore) Lock(ctx context.Context, identifier string, until time.Time) error {
	key := r.lockKey(identifier)
	ttl := time.Until(until)
	if ttl <= 0 {
		return nil
	}

	err := r.client.Set(ctx, key, strconv.FormatInt(until.UnixMilli(), 10), ttl).Err()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// Unlock implements LockoutStore.
func (r *RedisLockoutStore) Unlock(ctx context.Context, identifier string) error {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.lockKey(identifier))
	pipe.Del(ctx, r.attemptsKey(identifier))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// Reset implements LockoutStore.
func (r *RedisLockoutStore) Reset(ctx context.Context, identifier string) error {
	return r.Unlock(ctx, identifier)
}

// Close implements LockoutStore.
// Note: This does NOT close the Redis client since it may be shared.
func (r *RedisLockoutStore) Close() error {
	return nil
}
