# CoreAuth Overview

CoreAuth provides authentication and OAuth 2.0 functionality through clean, swappable provider interfaces. This design allows you to start with embedded implementations and migrate to external services (like Ory Hydra/Kratos) when needed.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Your Application                            │
├─────────────────────────────────────────────────────────────────┤
│                         CoreAuth                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │IdentityProvider │  │AuthenticationPr │  │  OAuthProvider  │ │
│  │  (User CRUD)    │  │ (Sessions)      │  │ (OAuth 2.0)     │ │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘ │
│           │                    │                    │           │
│  ┌────────▼────────────────────▼────────────────────▼────────┐ │
│  │              Embedded (Fosite) or Ory Adapters             │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Provider Interfaces

| Interface | Purpose | Maps to Ory |
|-----------|---------|-------------|
| `IdentityProvider` | User CRUD operations | Kratos Identity API |
| `AuthenticationProvider` | Session management | Kratos Session API |
| `OAuthProvider` | OAuth 2.0/OIDC flows | Hydra Public/Admin API |
| `OAuthClientStore` | OAuth client management | Hydra Client API |

## Quick Start

### Option 1: Full OAuth Server

```go
import "github.com/grokify/coreforge/identity/coreauth"

// Create embedded OAuth server
server, err := coreauth.NewEmbedded(coreauth.Config{
    Issuer: "https://auth.example.com",
})

// Get all providers
providers := coreauth.NewProviders(server,
    coreauth.WithProviderSessionDuration(24 * time.Hour),
    coreauth.WithProviderPasswordVerifier(myPasswordVerifier),
)

// Use providers
identity, _ := providers.Identity.GetIdentity(ctx, userID)
session, _ := providers.Authentication.ValidateSession(ctx, token)
userInfo, _ := providers.OAuth.UserInfo(ctx, accessToken)
```

### Option 2: Identity/Auth Only (No OAuth)

```go
// Create from storage directly
storage := coreauth.NewMemoryStorage() // or Ent storage
providers := coreauth.NewProvidersFromStorage(storage)

// Identity and Authentication available
// OAuth is nil (no server)
```

## Embedded vs Ory

| Aspect | Embedded | Ory Services |
|--------|----------|--------------|
| Deployment | Single binary | Multiple services |
| Database | App's database | Separate databases |
| Scaling | With app | Independent |
| Customization | Full code control | Config + webhooks |
| Production Ready | ✅ (Fosite-based) | ✅ (Battle-tested) |

## Migration Path

Start embedded, migrate to Ory when needed:

```go
// Today: Embedded
providers := coreauth.NewProviders(server)

// Future: Ory adapters (same interface)
providers := &coreauth.Providers{
    Identity:       ory.NewKratosIdentityProvider(kratosClient),
    Authentication: ory.NewKratosAuthProvider(kratosClient),
    OAuth:          ory.NewHydraOAuthProvider(hydraClient),
    OAuthClients:   ory.NewHydraClientStore(hydraAdminClient),
}

// Application code using providers.* doesn't change!
```

## Next Steps

- [Identity Provider](providers.md#identity-provider) - User management
- [Authentication Provider](providers.md#authentication-provider) - Sessions
- [OAuth Provider](providers.md#oauth-provider) - OAuth 2.0 flows
