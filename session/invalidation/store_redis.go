package invalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed implementation of SessionStore.
// Suitable for distributed deployments.
type RedisStore struct {
	client    redis.UniversalClient
	keyPrefix string
}

// RedisStoreOption configures RedisStore.
type RedisStoreOption func(*RedisStore)

// WithKeyPrefix sets a prefix for all session keys in Redis.
func WithKeyPrefix(prefix string) RedisStoreOption {
	return func(r *RedisStore) {
		r.keyPrefix = prefix
	}
}

// NewRedisStore creates a new Redis-backed session store.
func NewRedisStore(client redis.UniversalClient, opts ...RedisStoreOption) *RedisStore {
	r := &RedisStore{
		client:    client,
		keyPrefix: "session:",
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *RedisStore) sessionKey(sessionID string) string {
	return r.keyPrefix + "id:" + sessionID
}

func (r *RedisStore) userSessionsKey(userID string) string {
	return r.keyPrefix + "user:" + userID
}

// Create implements SessionStore.
func (r *RedisStore) Create(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return nil // Already expired
	}

	pipe := r.client.Pipeline()

	// Store session data
	pipe.Set(ctx, r.sessionKey(session.ID), data, ttl)

	// Add to user's session set with expiration as score
	pipe.ZAdd(ctx, r.userSessionsKey(session.UserID), redis.Z{
		Score:  float64(session.ExpiresAt.Unix()),
		Member: session.ID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// Get implements SessionStore.
func (r *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	data, err := r.client.Get(ctx, r.sessionKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return &session, nil
}

// Update implements SessionStore.
func (r *RedisStore) Update(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return r.Delete(ctx, session.ID)
	}

	pipe := r.client.Pipeline()

	// Update session data
	pipe.Set(ctx, r.sessionKey(session.ID), data, ttl)

	// Update expiration in user's session set
	pipe.ZAdd(ctx, r.userSessionsKey(session.UserID), redis.Z{
		Score:  float64(session.ExpiresAt.Unix()),
		Member: session.ID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// Delete implements SessionStore.
func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	// Get session first to find user ID
	session, err := r.Get(ctx, sessionID)
	if err != nil {
		if err == ErrSessionNotFound {
			return nil // Already deleted
		}
		return err
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, r.sessionKey(sessionID))
	pipe.ZRem(ctx, r.userSessionsKey(session.UserID), sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	return nil
}

// ListByUser implements SessionStore.
func (r *RedisStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	// Get all session IDs for user
	sessionIDs, err := r.client.ZRange(ctx, r.userSessionsKey(userID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if len(sessionIDs) == 0 {
		return nil, nil
	}

	// Get all sessions
	keys := make([]string, len(sessionIDs))
	for i, id := range sessionIDs {
		keys[i] = r.sessionKey(id)
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	sessions := make([]*Session, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}

		var session Session
		if err := json.Unmarshal([]byte(result.(string)), &session); err != nil {
			continue
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// DeleteByUser implements SessionStore.
func (r *RedisStore) DeleteByUser(ctx context.Context, userID string) (int, error) {
	// Get all session IDs for user
	sessionIDs, err := r.client.ZRange(ctx, r.userSessionsKey(userID), 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	// Delete all sessions
	keys := make([]string, len(sessionIDs)+1)
	for i, id := range sessionIDs {
		keys[i] = r.sessionKey(id)
	}
	keys[len(sessionIDs)] = r.userSessionsKey(userID)

	deleted, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrStorageFailure, err)
	}

	// Subtract 1 for the user sessions key
	return int(deleted) - 1, nil
}

// DeleteByDevice implements SessionStore.
func (r *RedisStore) DeleteByDevice(ctx context.Context, userID, deviceID string) (int, error) {
	sessions, err := r.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, session := range sessions {
		if session.DeviceID == deviceID {
			if err := r.Delete(ctx, session.ID); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// DeleteExpired implements SessionStore.
func (r *RedisStore) DeleteExpired(ctx context.Context) (int, error) {
	// Redis automatically expires keys, but we need to clean up the user session sets
	// This is a simplified implementation that relies on Redis TTL for session keys
	// A production implementation might scan for expired entries in the sorted sets
	return 0, nil
}

// Close implements SessionStore.
// Note: This does NOT close the Redis client since it may be shared.
func (r *RedisStore) Close() error {
	return nil
}
