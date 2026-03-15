package ratelimit

import (
	"context"
	"sync"
	"time"
)

// MemoryStorage is an in-memory rate limit storage using sliding window algorithm.
// Suitable for single-instance deployments and development/testing.
type MemoryStorage struct {
	windows map[string]*slidingWindow
	mu      sync.RWMutex
	closed  bool

	// cleanupInterval controls how often expired windows are cleaned up.
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

type slidingWindow struct {
	count     int
	startTime time.Time
	mu        sync.Mutex
}

// MemoryOption configures MemoryStorage.
type MemoryOption func(*MemoryStorage)

// WithCleanupInterval sets the interval for cleaning up expired windows.
func WithCleanupInterval(d time.Duration) MemoryOption {
	return func(m *MemoryStorage) {
		m.cleanupInterval = d
	}
}

// NewMemoryStorage creates a new in-memory rate limit storage.
func NewMemoryStorage(opts ...MemoryOption) *MemoryStorage {
	m := &MemoryStorage{
		windows:         make(map[string]*slidingWindow),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Start background cleanup goroutine
	go m.cleanup()

	return m
}

// Allow implements Storage.Allow using a sliding window algorithm.
func (m *MemoryStorage) Allow(ctx context.Context, key string, limit Limit) (Result, error) {
	m.mu.Lock()
	window, exists := m.windows[key]
	if !exists {
		window = &slidingWindow{
			startTime: time.Now(),
		}
		m.windows[key] = window
	}
	m.mu.Unlock()

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	windowEnd := window.startTime.Add(limit.Period)

	// Check if we're in a new window
	if now.After(windowEnd) {
		window.count = 0
		window.startTime = now
		windowEnd = now.Add(limit.Period)
	}

	burst := limit.Burst
	if burst == 0 {
		burst = limit.Rate
	}

	// Check if under limit
	if window.count < burst {
		window.count++
		return Result{
			Allowed:   true,
			Remaining: burst - window.count,
			ResetAt:   windowEnd,
		}, nil
	}

	// Rate limited
	retryAfter := windowEnd.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return Result{
		Allowed:    false,
		Remaining:  0,
		ResetAt:    windowEnd,
		RetryAfter: retryAfter,
	}, nil
}

// Reset implements Storage.Reset.
func (m *MemoryStorage) Reset(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.windows, key)
	return nil
}

// Close implements Storage.Close.
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.stopCleanup)
	m.windows = nil
	return nil
}

// cleanup periodically removes expired windows to prevent memory growth.
func (m *MemoryStorage) cleanup() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCleanup:
			return
		case <-ticker.C:
			m.cleanupExpired()
		}
	}
}

func (m *MemoryStorage) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	// We'll use a generous expiration time (10x the cleanup interval)
	// since we don't know the period for each window
	expirationThreshold := max(10*m.cleanupInterval, time.Hour)
	now := time.Now()

	for key, window := range m.windows {
		window.mu.Lock()
		if now.Sub(window.startTime) > expirationThreshold {
			delete(m.windows, key)
		}
		window.mu.Unlock()
	}
}
