package coreapi

import "time"

// Predefined rate limit policies for common use cases.
var (
	// FreePolicy is for free tier users and anonymous requests.
	FreePolicy = &RateLimitPolicy{
		ID:          "free",
		Name:        "Free Tier",
		Description: "Rate limits for free tier users and anonymous requests",
		Enabled:     true,
		Limits: RateLimits{
			PerMinute: 10,
			PerHour:   100,
			PerDay:    500,
			BurstSize: 5,
		},
	}

	// StandardPolicy is for standard paid users.
	StandardPolicy = &RateLimitPolicy{
		ID:          "standard",
		Name:        "Standard Tier",
		Description: "Rate limits for standard tier users",
		Enabled:     true,
		Limits: RateLimits{
			PerMinute: 100,
			PerHour:   1000,
			PerDay:    10000,
			BurstSize: 20,
		},
	}

	// PremiumPolicy is for premium users.
	PremiumPolicy = &RateLimitPolicy{
		ID:          "premium",
		Name:        "Premium Tier",
		Description: "Rate limits for premium tier users",
		Enabled:     true,
		Limits: RateLimits{
			PerMinute: 1000,
			PerHour:   10000,
			PerDay:    100000,
			BurstSize: 100,
		},
	}

	// EnterprisePolicy is for enterprise users with high limits.
	EnterprisePolicy = &RateLimitPolicy{
		ID:          "enterprise",
		Name:        "Enterprise Tier",
		Description: "Rate limits for enterprise tier users",
		Enabled:     true,
		Limits: RateLimits{
			PerMinute: 10000,
			PerHour:   100000,
			PerDay:    1000000,
			BurstSize: 500,
		},
	}

	// InternalPolicy is for internal service-to-service calls.
	InternalPolicy = &RateLimitPolicy{
		ID:          "internal",
		Name:        "Internal Service",
		Description: "High limits for internal service-to-service communication",
		Enabled:     true,
		Priority:    100,
		Limits: RateLimits{
			PerSecond: 10000,
			BurstSize: 1000,
		},
	}

	// UnlimitedPolicy has no rate limits (use with caution).
	UnlimitedPolicy = &RateLimitPolicy{
		ID:          "unlimited",
		Name:        "Unlimited",
		Description: "No rate limits - use for trusted internal services only",
		Enabled:     true,
		Priority:    1000,
		Limits:      RateLimits{}, // All zeros = no limits
	}
)

// DefaultPolicies returns all predefined policies.
func DefaultPolicies() []*RateLimitPolicy {
	return []*RateLimitPolicy{
		FreePolicy,
		StandardPolicy,
		PremiumPolicy,
		EnterprisePolicy,
		InternalPolicy,
		UnlimitedPolicy,
	}
}

// MostGranularLimit returns the most granular non-zero limit from RateLimits.
// Returns rate, period, and burst.
func (r RateLimits) MostGranularLimit() (rate int, period time.Duration, burst int) {
	burst = r.BurstSize

	if r.PerSecond > 0 {
		if burst == 0 {
			burst = r.PerSecond
		}
		return r.PerSecond, time.Second, burst
	}
	if r.PerMinute > 0 {
		if burst == 0 {
			burst = r.PerMinute
		}
		return r.PerMinute, time.Minute, burst
	}
	if r.PerHour > 0 {
		if burst == 0 {
			burst = r.PerHour
		}
		return r.PerHour, time.Hour, burst
	}
	if r.PerDay > 0 {
		if burst == 0 {
			burst = r.PerDay
		}
		return r.PerDay, 24 * time.Hour, burst
	}

	// No limits configured - return very high rate
	return 1000000, time.Second, 1000000
}

// IsUnlimited returns true if no rate limits are configured.
func (r RateLimits) IsUnlimited() bool {
	return r.PerSecond == 0 && r.PerMinute == 0 && r.PerHour == 0 && r.PerDay == 0
}

// WithEndpointOverride returns a copy of the policy with an endpoint override added.
func (p *RateLimitPolicy) WithEndpointOverride(pathPrefix string, limits RateLimits) *RateLimitPolicy {
	copy := *p
	if copy.EndpointOverrides == nil {
		copy.EndpointOverrides = make(map[string]RateLimits)
	}
	copy.EndpointOverrides[pathPrefix] = limits
	return &copy
}
