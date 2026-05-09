# TRD: JWT Audience Validation

> **Status**: Draft
> **Target**: SystemForge v0.5.0

## Overview

Technical design for adding audience (`aud`) claim validation to SystemForge's JWT service.

## Architecture

### Token Flow with Audience Separation

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Client Applications                            │
├────────────────────────────────┬────────────────────────────────────────┤
│        Web Browser (SPA)       │       Programmatic (CLI/SDK)           │
│                                │                                        │
│  Login via OAuth/Password      │  Create API Token via BFF              │
│           ↓                    │           ↓                            │
│  Receives JWT with:            │  Receives JWT with:                    │
│    aud: "bff"                  │    aud: "api"                          │
│    scopes: null                │    scopes: ["evidence:read", ...]      │
│                                │                                        │
│  Uses cookie on /bff/v1/*      │  Uses Bearer on /api/v1/*              │
└────────────────────────────────┴────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            API Gateway                                   │
├────────────────────────────────┬────────────────────────────────────────┤
│         /bff/v1/*              │            /api/v1/*                   │
│                                │                                        │
│  CookieAuthMiddleware          │  BearerAuthMiddleware                  │
│  ↓                             │  ↓                                     │
│  ValidateAccessToken(          │  ValidateAccessToken(                  │
│    token,                      │    token,                              │
│    audience: "bff"             │    audience: "api"                     │
│  )                             │  )                                     │
│  ↓                             │  ↓                                     │
│  Reject if aud != "bff"        │  Reject if aud != "api"                │
│  ↓                             │  ↓                                     │
│  CSRFMiddleware                │  ScopeMiddleware                       │
└────────────────────────────────┴────────────────────────────────────────┘
```

## JWT Claims Structure

### Standard Claims (RFC 7519)

The `aud` (audience) claim is already supported via `jwt.RegisteredClaims.Audience`.

```go
// Current SystemForge Claims structure
type Claims struct {
    jwt.RegisteredClaims  // Includes Audience []string

    PrincipalID    uuid.UUID `json:"pid,omitempty"`
    PrincipalType  string    `json:"pty,omitempty"`
    Email          string    `json:"email,omitempty"`
    Name           string    `json:"name,omitempty"`
    Scopes         []string  `json:"scp,omitempty"`
    TokenType      TokenType `json:"type,omitempty"`
    // ... other fields
}
```

### Audience Values

| Audience | Purpose | Used By |
|----------|---------|---------|
| `bff` | Web browser sessions via BFF | CookieAuthMiddleware |
| `api` | Programmatic API access | BearerAuthMiddleware |
| `internal` | Service-to-service (future) | InternalAuthMiddleware |

## API Changes

### JWT Service Changes

```go
// session/jwt/service.go

// GenerateAccessTokenWithAudience creates an access token with explicit audience.
func (s *Service) GenerateAccessTokenWithAudience(
    principalID uuid.UUID,
    email, name string,
    audience string,
    scopes []string,
) (string, error)

// ValidateAccessTokenWithAudience validates and checks expected audience.
// Returns ErrAudienceMismatch if the token audience doesn't match.
func (s *Service) ValidateAccessTokenWithAudience(
    tokenString string,
    expectedAudience string,
) (*Claims, error)

// New error type
var ErrAudienceMismatch = errors.New("audience mismatch")
```

### Claims Helpers

```go
// session/jwt/claims.go

// Audience returns the first audience value.
func (c *Claims) Audience() string {
    if len(c.RegisteredClaims.Audience) > 0 {
        return c.RegisteredClaims.Audience[0]
    }
    return ""
}

// HasAudience checks if the claims include a specific audience.
func (c *Claims) HasAudience(aud string) bool {
    for _, a := range c.RegisteredClaims.Audience {
        if a == aud {
            return true
        }
    }
    return false
}

// WithAudience sets the audience claim.
func (c *Claims) WithAudience(audiences ...string) *Claims {
    c.RegisteredClaims.Audience = audiences
    return c
}
```

### Token Pair Generation

```go
// session/jwt/service.go

// GenerateBFFTokenPair creates tokens for BFF (web) clients.
func (s *Service) GenerateBFFTokenPair(
    principalID uuid.UUID,
    email, name string,
) (*TokenPair, error) {
    return s.GenerateTokenPairWithAudience(principalID, email, name, "bff", nil)
}

// GenerateAPIToken creates a scoped token for API clients.
// Note: API tokens typically don't have refresh tokens.
func (s *Service) GenerateAPIToken(
    principalID uuid.UUID,
    email, name string,
    scopes []string,
    duration time.Duration,
) (string, error)

// GenerateTokenPairWithAudience creates a token pair with explicit audience.
func (s *Service) GenerateTokenPairWithAudience(
    principalID uuid.UUID,
    email, name string,
    audience string,
    scopes []string,
) (*TokenPair, error)
```

## Middleware Integration

### BFF Cookie Middleware

```go
// session/bff/middleware.go

// RequireSessionWithAudience validates the session cookie and checks audience.
func RequireSessionWithAudience(
    store Store,
    cookieManager *CookieManager,
    jwtService *jwt.Service,
    expectedAudience string,
) func(http.Handler) http.Handler
```

### API Bearer Middleware

```go
// session/middleware/apikey.go

// BearerAuthMiddleware validates Bearer tokens with audience check.
func BearerAuthMiddleware(config BearerAuthConfig) func(http.Handler) http.Handler

type BearerAuthConfig struct {
    JWTService       *jwt.Service
    ExpectedAudience string  // e.g., "api"
    // ... other fields
}
```

## Migration Strategy

### Phase 1: Add Audience Support (Non-Breaking)

1. Add `ValidateAccessTokenWithAudience` method
2. Add audience helper methods to Claims
3. Existing `ValidateAccessToken` continues to work without audience check

### Phase 2: Add Generation Methods

1. Add `GenerateBFFTokenPair`
2. Add `GenerateAPIToken`
3. Add `GenerateTokenPairWithAudience`

### Phase 3: App Migration

1. Apps update token generation to use audience-aware methods
2. Apps update middleware to check expected audience
3. Existing tokens continue to work (no audience = any path)

### Phase 4: Enforce Audience (Optional)

1. Add config option to require audience on all tokens
2. Apps can enable enforcement when ready
3. Reject tokens without audience claim

## Error Handling

```go
// Error responses
var (
    ErrAudienceMismatch = errors.New("audience mismatch")
    ErrMissingAudience  = errors.New("missing audience claim")
)

// HTTP error responses
// 401 Unauthorized: "Invalid token: audience mismatch (expected: bff, got: api)"
```

## Testing Strategy

### Unit Tests

1. Test audience is set correctly during token generation
2. Test audience validation passes for matching audience
3. Test audience validation fails for mismatched audience
4. Test backward compatibility (no audience = passes if not required)

### Integration Tests

1. BFF path rejects API tokens
2. API path rejects BFF tokens
3. Token replay from DevTools is rejected
4. Migration scenario: old tokens still work

## Security Considerations

1. **Audience as Defense in Depth**: Not a replacement for other security measures
2. **Clear Error Messages**: Don't leak information about expected audience to attackers
3. **Audit Logging**: Log audience mismatches for security monitoring
