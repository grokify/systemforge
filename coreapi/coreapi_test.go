package coreapi

import (
	"testing"
)

func TestMemoryPolicyStore_CRUD(t *testing.T) {
	store := NewMemoryPolicyStore()
	ctx := t.Context()

	// Test default policy exists
	defaultPolicy, err := store.GetDefaultPolicy(ctx)
	if err != nil {
		t.Fatalf("GetDefaultPolicy failed: %v", err)
	}
	if defaultPolicy.ID != "standard" {
		t.Errorf("expected default policy 'standard', got '%s'", defaultPolicy.ID)
	}

	// Test CreatePolicy
	customPolicy := &RateLimitPolicy{
		ID:      "custom",
		Name:    "Custom Policy",
		Enabled: true,
		Limits: RateLimits{
			PerMinute: 500,
			BurstSize: 50,
		},
	}
	err = store.CreatePolicy(ctx, customPolicy)
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	// Test GetPolicy
	retrieved, err := store.GetPolicy(ctx, "custom")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if retrieved.Name != "Custom Policy" {
		t.Errorf("expected 'Custom Policy', got '%s'", retrieved.Name)
	}

	// Test UpdatePolicy
	retrieved.Name = "Updated Policy"
	err = store.UpdatePolicy(ctx, retrieved)
	if err != nil {
		t.Fatalf("UpdatePolicy failed: %v", err)
	}
	retrieved, _ = store.GetPolicy(ctx, "custom")
	if retrieved.Name != "Updated Policy" {
		t.Errorf("expected 'Updated Policy', got '%s'", retrieved.Name)
	}

	// Test ListPolicies
	policies, err := store.ListPolicies(ctx)
	if err != nil {
		t.Fatalf("ListPolicies failed: %v", err)
	}
	if len(policies) != 2 { // standard + custom
		t.Errorf("expected 2 policies, got %d", len(policies))
	}

	// Test DeletePolicy
	err = store.DeletePolicy(ctx, "custom")
	if err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}
	_, err = store.GetPolicy(ctx, "custom")
	if err != ErrPolicyNotFound {
		t.Error("expected ErrPolicyNotFound after delete")
	}
}

func TestMemoryPolicyStore_ClientBindings(t *testing.T) {
	store := NewMemoryPolicyStore()
	ctx := t.Context()

	// Create premium policy
	err := store.CreatePolicy(ctx, PremiumPolicy)
	if err != nil {
		t.Fatal(err)
	}

	// Bind client to premium
	err = store.BindClientToPolicy(ctx, "client-123", "premium")
	if err != nil {
		t.Fatalf("BindClientToPolicy failed: %v", err)
	}

	// Get policy for client
	policy, err := store.GetPolicyForClient(ctx, "client-123")
	if err != nil {
		t.Fatalf("GetPolicyForClient failed: %v", err)
	}
	if policy.ID != "premium" {
		t.Errorf("expected 'premium', got '%s'", policy.ID)
	}

	// Get policy for unknown client (should return default)
	policy, err = store.GetPolicyForClient(ctx, "unknown-client")
	if err != nil {
		t.Fatalf("GetPolicyForClient failed: %v", err)
	}
	if policy.ID != "standard" {
		t.Errorf("expected default 'standard', got '%s'", policy.ID)
	}

	// List bindings
	bindings, err := store.ListClientBindings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(bindings))
	}

	// List clients for policy
	clients, err := store.ListClientsForPolicy(ctx, "premium")
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 || clients[0] != "client-123" {
		t.Errorf("expected ['client-123'], got %v", clients)
	}

	// Unbind client
	err = store.UnbindClient(ctx, "client-123")
	if err != nil {
		t.Fatal(err)
	}
	policy, _ = store.GetPolicyForClient(ctx, "client-123")
	if policy.ID != "standard" {
		t.Error("expected default policy after unbind")
	}
}

func TestRateLimits_MostGranularLimit(t *testing.T) {
	tests := []struct {
		name       string
		limits     RateLimits
		wantRate   int
		wantPeriod string
	}{
		{
			name:       "per second takes precedence",
			limits:     RateLimits{PerSecond: 10, PerMinute: 100, PerHour: 1000},
			wantRate:   10,
			wantPeriod: "1s",
		},
		{
			name:       "per minute when no second",
			limits:     RateLimits{PerMinute: 60, PerHour: 1000},
			wantRate:   60,
			wantPeriod: "1m0s",
		},
		{
			name:       "per hour when no minute",
			limits:     RateLimits{PerHour: 1000, PerDay: 10000},
			wantRate:   1000,
			wantPeriod: "1h0m0s",
		},
		{
			name:       "per day only",
			limits:     RateLimits{PerDay: 10000},
			wantRate:   10000,
			wantPeriod: "24h0m0s",
		},
		{
			name:       "unlimited returns high rate",
			limits:     RateLimits{},
			wantRate:   1000000,
			wantPeriod: "1s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, period, _ := tt.limits.MostGranularLimit()
			if rate != tt.wantRate {
				t.Errorf("rate: got %d, want %d", rate, tt.wantRate)
			}
			if period.String() != tt.wantPeriod {
				t.Errorf("period: got %s, want %s", period.String(), tt.wantPeriod)
			}
		})
	}
}

func TestRateLimits_IsUnlimited(t *testing.T) {
	unlimited := RateLimits{}
	if !unlimited.IsUnlimited() {
		t.Error("expected IsUnlimited to be true for empty limits")
	}

	limited := RateLimits{PerMinute: 100}
	if limited.IsUnlimited() {
		t.Error("expected IsUnlimited to be false when limits set")
	}
}

func TestPredefinedPolicies(t *testing.T) {
	policies := DefaultPolicies()

	if len(policies) != 6 {
		t.Errorf("expected 6 predefined policies, got %d", len(policies))
	}

	// Verify tier ordering
	freeLimits := FreePolicy.Limits
	standardLimits := StandardPolicy.Limits
	premiumLimits := PremiumPolicy.Limits
	enterpriseLimits := EnterprisePolicy.Limits

	if freeLimits.PerMinute >= standardLimits.PerMinute {
		t.Error("free should have lower limits than standard")
	}
	if standardLimits.PerMinute >= premiumLimits.PerMinute {
		t.Error("standard should have lower limits than premium")
	}
	if premiumLimits.PerMinute >= enterpriseLimits.PerMinute {
		t.Error("premium should have lower limits than enterprise")
	}
}

func TestPolicy_WithEndpointOverride(t *testing.T) {
	policy := StandardPolicy.WithEndpointOverride("/api/v1/uploads", RateLimits{
		PerMinute: 10,
		BurstSize: 5,
	})

	if policy.EndpointOverrides == nil {
		t.Fatal("expected endpoint overrides to be set")
	}

	override, ok := policy.EndpointOverrides["/api/v1/uploads"]
	if !ok {
		t.Fatal("expected upload endpoint override")
	}
	if override.PerMinute != 10 {
		t.Errorf("expected PerMinute 10, got %d", override.PerMinute)
	}

	// Original should be unchanged
	if StandardPolicy.EndpointOverrides != nil {
		t.Error("original policy should not have overrides")
	}
}
