package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryStorage_Allow(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()

	limit := Limit{Rate: 3, Period: time.Minute, Burst: 3}
	key := "test-key"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		result, err := storage.Allow(t.Context(), key, limit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		if result.Remaining != 3-i-1 {
			t.Errorf("expected remaining %d, got %d", 3-i-1, result.Remaining)
		}
	}

	// 4th request should be rate limited
	result, err := storage.Allow(t.Context(), key, limit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("4th request should be rate limited")
	}
	if result.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

func TestMemoryStorage_Reset(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()

	limit := Limit{Rate: 1, Period: time.Minute, Burst: 1}
	key := "reset-test"

	// Use up the limit
	result, _ := storage.Allow(t.Context(), key, limit)
	if !result.Allowed {
		t.Fatal("first request should be allowed")
	}

	// Should be rate limited
	result, _ = storage.Allow(t.Context(), key, limit)
	if result.Allowed {
		t.Fatal("should be rate limited")
	}

	// Reset the limit
	err := storage.Reset(t.Context(), key)
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Should be allowed again
	result, _ = storage.Allow(t.Context(), key, limit)
	if !result.Allowed {
		t.Error("should be allowed after reset")
	}
}

func TestLimiter_Middleware(t *testing.T) {
	storage := NewMemoryStorage()
	defer storage.Close()

	limiter := New(storage,
		WithLimit(Limit{Rate: 2, Period: time.Minute, Burst: 2}),
		WithKeyFunc(IPKey()),
	)

	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	// Check rate limit headers
	if rec.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("expected X-RateLimit-Limit header to be 2")
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestStaticResolver(t *testing.T) {
	limit := PerMinute(100)
	resolver := &StaticResolver{Limit: limit}

	req := httptest.NewRequest("GET", "/test", nil)
	result := resolver.Resolve(req.Context(), "test", req)

	if result.Rate != 100 {
		t.Errorf("expected rate 100, got %d", result.Rate)
	}
	if result.Period != time.Minute {
		t.Errorf("expected period 1m, got %v", result.Period)
	}
}

func TestPerSecond(t *testing.T) {
	limit := PerSecond(10)
	if limit.Rate != 10 {
		t.Errorf("expected rate 10, got %d", limit.Rate)
	}
	if limit.Period != time.Second {
		t.Errorf("expected period 1s, got %v", limit.Period)
	}
}

func TestPerHour(t *testing.T) {
	limit := PerHour(1000)
	if limit.Rate != 1000 {
		t.Errorf("expected rate 1000, got %d", limit.Rate)
	}
	if limit.Period != time.Hour {
		t.Errorf("expected period 1h, got %v", limit.Period)
	}
}
