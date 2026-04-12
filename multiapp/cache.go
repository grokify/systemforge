package multiapp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a key is not found in the cache.
var ErrCacheMiss = errors.New("cache miss")

// RedisCache implements Cache using Redis.
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis-backed cache.
func NewRedisCache(redisURL string) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

// Get retrieves a value from Redis.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key
	result, err := c.client.Get(ctx, fullKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	return result, err
}

// Set stores a value in Redis.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	fullKey := c.prefix + key
	var ttl time.Duration
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	return c.client.Set(ctx, fullKey, value, ttl).Err()
}

// Delete removes a value from Redis.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// WithPrefix returns a new cache with a key prefix.
func (c *RedisCache) WithPrefix(prefix string) Cache {
	return &RedisCache{
		client: c.client,
		prefix: c.prefix + prefix,
	}
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// MemoryCache implements Cache using in-memory storage.
// Useful for testing and single-instance deployments.
type MemoryCache struct {
	data   map[string]cacheEntry
	mu     sync.RWMutex
	prefix string
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		data: make(map[string]cacheEntry),
	}
	// Start cleanup goroutine
	go c.cleanup()
	return c
}

// Get retrieves a value from memory.
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key

	c.mu.RLock()
	entry, ok := c.data[fullKey]
	c.mu.RUnlock()

	if !ok {
		return nil, ErrCacheMiss
	}

	// Check expiration
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		_ = c.Delete(ctx, key)
		return nil, ErrCacheMiss
	}

	return entry.value, nil
}

// Set stores a value in memory.
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	fullKey := c.prefix + key

	entry := cacheEntry{
		value: value,
	}
	if ttlSeconds > 0 {
		entry.expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}

	c.mu.Lock()
	c.data[fullKey] = entry
	c.mu.Unlock()

	return nil
}

// Delete removes a value from memory.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	c.mu.Lock()
	delete(c.data, fullKey)
	c.mu.Unlock()

	return nil
}

// WithPrefix returns a new cache with a key prefix.
func (c *MemoryCache) WithPrefix(prefix string) Cache {
	return &MemoryCache{
		data:   c.data,
		prefix: c.prefix + prefix,
	}
}

// cleanup periodically removes expired entries.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		c.mu.Lock()
		for key, entry := range c.data {
			if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}
