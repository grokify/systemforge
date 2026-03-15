# Rate Limit Policies

Policies define rate limits that can be assigned to OAuth clients. Each policy specifies limits across multiple time windows.

## Policy Structure

```go
type RateLimitPolicy struct {
    ID                string                  // Unique identifier (e.g., "premium")
    Name              string                  // Display name
    Limits            RateLimits              // Default limits
    EndpointOverrides map[string]RateLimits   // Per-endpoint limits
    Enabled           bool                    // Whether policy is active
}

type RateLimits struct {
    PerSecond int // Requests per second (0 = unlimited)
    PerMinute int // Requests per minute
    PerHour   int // Requests per hour
    PerDay    int // Requests per day
}
```

## Predefined Policies

CoreAPI includes predefined policies for common use cases:

### FreePolicy

```go
coreapi.FreePolicy
// Per-Minute: 10, Per-Hour: 100, Per-Day: 500
```

For anonymous users or free tier accounts. Very restrictive limits.

### StandardPolicy

```go
coreapi.StandardPolicy
// Per-Minute: 100, Per-Hour: 1,000, Per-Day: 10,000
```

For paid individual users. Suitable for typical API usage.

### PremiumPolicy

```go
coreapi.PremiumPolicy
// Per-Minute: 1,000, Per-Hour: 10,000, Per-Day: 100,000
```

For premium tier users. Higher limits for power users.

### EnterprisePolicy

```go
coreapi.EnterprisePolicy
// Per-Minute: 10,000, Per-Hour: 100,000, Per-Day: 1,000,000
```

For enterprise customers with high-volume needs.

### InternalPolicy

```go
coreapi.InternalPolicy
// Per-Second: 10,000, no other limits
```

For internal service-to-service communication. Very high limits.

## Creating Custom Policies

```go
customPolicy := &coreapi.RateLimitPolicy{
    ID:      "startup-tier",
    Name:    "Startup Plan",
    Enabled: true,
    Limits: coreapi.RateLimits{
        PerMinute: 500,
        PerHour:   5000,
        PerDay:    50000,
    },
}

policyStore.CreatePolicy(ctx, customPolicy)
```

## Endpoint Overrides

Apply different limits to specific endpoints:

```go
policy := &coreapi.RateLimitPolicy{
    ID:      "with-overrides",
    Name:    "Custom with Overrides",
    Enabled: true,
    Limits: coreapi.RateLimits{
        PerMinute: 100,
    },
    EndpointOverrides: map[string]coreapi.RateLimits{
        // Expensive search endpoint gets stricter limits
        "/api/v1/search": {
            PerMinute: 10,
        },
        // Health check is unlimited
        "/health": {
            PerMinute: 0, // 0 = unlimited
        },
    },
}
```

## Policy Store Interface

```go
type PolicyStore interface {
    // CRUD operations
    CreatePolicy(ctx context.Context, policy *RateLimitPolicy) error
    GetPolicy(ctx context.Context, policyID string) (*RateLimitPolicy, error)
    UpdatePolicy(ctx context.Context, policy *RateLimitPolicy) error
    DeletePolicy(ctx context.Context, policyID string) error
    ListPolicies(ctx context.Context) ([]*RateLimitPolicy, error)

    // Client bindings
    GetPolicyForClient(ctx context.Context, clientID string) (*RateLimitPolicy, error)
    BindClientToPolicy(ctx context.Context, clientID, policyID string) error
    UnbindClient(ctx context.Context, clientID string) error

    // Defaults
    GetDefaultPolicy(ctx context.Context) (*RateLimitPolicy, error)
    SetDefaultPolicy(ctx context.Context, policyID string) error
}
```

## Memory Policy Store

For development and testing:

```go
store := coreapi.NewMemoryPolicyStore()

// Add predefined policies
store.CreatePolicy(ctx, coreapi.StandardPolicy)
store.CreatePolicy(ctx, coreapi.PremiumPolicy)

// Set default
store.SetDefaultPolicy(ctx, "standard")

// Bind clients
store.BindClientToPolicy(ctx, "mobile-app", "standard")
store.BindClientToPolicy(ctx, "partner-api", "premium")
```

## Lookup Behavior

When looking up a policy for a client:

1. Check if client has explicit binding → return bound policy
2. If no binding, return default policy
3. If no default, return error

```go
policy, err := store.GetPolicyForClient(ctx, "mobile-app")
if err == coreapi.ErrNoPolicy {
    // No policy found, decide how to handle
}
```

## Next Steps

- [Rate Limiting](ratelimit.md) - Apply policies with middleware
- [Overview](overview.md) - CoreAPI architecture
