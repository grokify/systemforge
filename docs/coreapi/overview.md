# CoreAPI Overview

CoreAPI provides API management capabilities including rate limiting with policy-based controls. Assign different rate limit policies to OAuth clients for tiered access.

## Features

- **Policy-based rate limiting** - Define rate limits as reusable policies
- **Client bindings** - Assign policies to OAuth clients via `client_id`
- **Multiple time windows** - Per-second, per-minute, per-hour, per-day limits
- **Endpoint overrides** - Different limits for specific endpoints
- **Predefined tiers** - Free, Standard, Premium, Enterprise policies

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Request                             │
│                    (with JWT access token)                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────────┐
│                     JWT Middleware                                │
│           Extract: client_id (azp), principal_id (pid)           │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────────┐
│                      CoreAPI PolicyStore                          │
│        Look up policy for client_id → rate limits                │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────────┐
│                   Rate Limit Middleware                          │
│         Enforce limits using sliding window algorithm            │
└───────────────────────────┬───────────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────────┐
│                      Your API Handler                             │
└───────────────────────────────────────────────────────────────────┘
```

## Quick Start

```go
import (
    "github.com/grokify/coreforge/coreapi"
    "github.com/grokify/coreforge/session/ratelimit"
)

// 1. Create policy store
policyStore := coreapi.NewMemoryPolicyStore()

// 2. Add policies
policyStore.CreatePolicy(ctx, coreapi.PremiumPolicy)
policyStore.CreatePolicy(ctx, coreapi.EnterprisePolicy)

// 3. Assign policies to OAuth clients
policyStore.BindClientToPolicy(ctx, "mobile-app", "standard")
policyStore.BindClientToPolicy(ctx, "partner-api", "premium")
policyStore.BindClientToPolicy(ctx, "internal-svc", "enterprise")

// 4. Create rate limiter
limiter := ratelimit.New(
    ratelimit.NewMemoryStorage(),
    ratelimit.WithResolver(ratelimit.NewCoreAPIResolver(policyStore)),
    ratelimit.WithKeyFunc(ratelimit.CompositeKey()),
)

// 5. Apply middleware
router.Use(jwtMiddleware)         // Extract client_id from JWT
router.Use(limiter.Middleware())  // Enforce rate limits
```

## Predefined Policies

| Policy | Per Minute | Per Hour | Per Day | Use Case |
|--------|------------|----------|---------|----------|
| `FreePolicy` | 10 | 100 | 500 | Anonymous/free users |
| `StandardPolicy` | 100 | 1,000 | 10,000 | Paid users |
| `PremiumPolicy` | 1,000 | 10,000 | 100,000 | Premium tier |
| `EnterprisePolicy` | 10,000 | 100,000 | 1,000,000 | Enterprise |
| `InternalPolicy` | 10,000/sec | - | - | Service-to-service |

## Next Steps

- [Rate Limit Policies](policies.md) - Define custom policies
- [Rate Limiting](ratelimit.md) - Enforcement middleware
