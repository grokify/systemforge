package ratelimit

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/grokify/systemforge/coreapi"
	"github.com/grokify/systemforge/session/middleware"
)

// CoreAPIResolver adapts coreapi.PolicyStore to the LimitResolver interface.
// It looks up rate limit policies based on the OAuth client ID from JWT claims.
type CoreAPIResolver struct {
	store        coreapi.PolicyStore
	cache        *resolverCache
	cacheEnabled bool
	cacheTTL     time.Duration
}

// CoreAPIResolverOption configures a CoreAPIResolver.
type CoreAPIResolverOption func(*CoreAPIResolver)

// WithResolverCache enables caching of policy lookups.
func WithResolverCache(ttl time.Duration) CoreAPIResolverOption {
	return func(r *CoreAPIResolver) {
		r.cacheEnabled = true
		r.cacheTTL = ttl
		r.cache = newResolverCache(ttl)
	}
}

// NewCoreAPIResolver creates a LimitResolver that uses coreapi.PolicyStore.
func NewCoreAPIResolver(store coreapi.PolicyStore, opts ...CoreAPIResolverOption) *CoreAPIResolver {
	r := &CoreAPIResolver{
		store:    store,
		cacheTTL: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Resolve implements LimitResolver by looking up the policy for the OAuth client.
func (r *CoreAPIResolver) Resolve(ctx context.Context, key string, req *http.Request) Limit {
	claims := middleware.ClaimsFromContext(ctx)

	var clientID string
	if claims != nil {
		clientID = claims.ClientID
	}

	policy := r.resolvePolicy(ctx, clientID)
	if policy == nil {
		return DefaultLimit()
	}

	// Check for endpoint-specific overrides
	if len(policy.EndpointOverrides) > 0 && req != nil {
		path := req.URL.Path
		for prefix, limits := range policy.EndpointOverrides {
			if strings.HasPrefix(path, prefix) {
				rate, period, burst := limits.MostGranularLimit()
				return Limit{Rate: rate, Period: period, Burst: burst}
			}
		}
	}

	rate, period, burst := policy.Limits.MostGranularLimit()
	return Limit{Rate: rate, Period: period, Burst: burst}
}

// resolvePolicy looks up the policy for a client, with optional caching.
func (r *CoreAPIResolver) resolvePolicy(ctx context.Context, clientID string) *coreapi.RateLimitPolicy {
	// Check cache first
	if r.cacheEnabled && r.cache != nil {
		if policy, ok := r.cache.get(clientID); ok {
			return policy
		}
	}

	// Look up policy from store
	policy, err := r.store.GetPolicyForClient(ctx, clientID)
	if err != nil || policy == nil {
		// Fall back to default
		policy, err = r.store.GetDefaultPolicy(ctx)
		if err != nil {
			return nil
		}
	}

	// Cache the result
	if r.cacheEnabled && r.cache != nil {
		r.cache.set(clientID, policy)
	}

	return policy
}

// GetPolicyForRequest returns the policy that would be applied to a request.
// Useful for debugging and displaying policy info in responses.
func (r *CoreAPIResolver) GetPolicyForRequest(ctx context.Context, req *http.Request) *coreapi.RateLimitPolicy {
	claims := middleware.ClaimsFromContext(ctx)

	var clientID string
	if claims != nil {
		clientID = claims.ClientID
	}

	return r.resolvePolicy(ctx, clientID)
}

// InvalidateCache clears the policy cache.
func (r *CoreAPIResolver) InvalidateCache(clientID string) {
	if r.cache != nil {
		if clientID == "" {
			r.cache.clear()
		} else {
			r.cache.delete(clientID)
		}
	}
}

// resolverCache caches policy lookups.
type resolverCache struct {
	entries map[string]*resolverCacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type resolverCacheEntry struct {
	policy    *coreapi.RateLimitPolicy
	expiresAt time.Time
}

func newResolverCache(ttl time.Duration) *resolverCache {
	c := &resolverCache{
		entries: make(map[string]*resolverCacheEntry),
		ttl:     ttl,
	}
	go c.cleanup()
	return c
}

func (c *resolverCache) get(clientID string) (*coreapi.RateLimitPolicy, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := clientID
	if key == "" {
		key = "__default__"
	}

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.policy, true
}

func (c *resolverCache) set(clientID string, policy *coreapi.RateLimitPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := clientID
	if key == "" {
		key = "__default__"
	}

	c.entries[key] = &resolverCacheEntry{
		policy:    policy,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *resolverCache) delete(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := clientID
	if key == "" {
		key = "__default__"
	}
	delete(c.entries, key)
}

func (c *resolverCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*resolverCacheEntry)
}

func (c *resolverCache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.expiresAt) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
