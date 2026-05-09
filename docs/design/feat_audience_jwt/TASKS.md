# Tasks: JWT Audience Validation

> **Status**: Draft
> **Target**: SystemForge v0.5.0

## Priority Levels

- **P0**: Critical path, blocks other work
- **P1**: High priority, core functionality
- **P2**: Important, enhances security/UX
- **P3**: Nice to have, future iteration

---

## Phase 1: Claims Helpers (P0)

### Task 1.1: Add Audience Helper Methods
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `session/jwt/claims.go`
- `session/jwt/claims_test.go`

**Implementation**:
```go
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

// WithAudience sets the audience claim (builder pattern).
func (c *Claims) WithAudience(audiences ...string) *Claims {
    c.RegisteredClaims.Audience = audiences
    return c
}
```

**Acceptance Criteria**:
- [ ] `Audience()` returns first audience or empty string
- [ ] `HasAudience()` returns true if audience is present
- [ ] `WithAudience()` sets audience and returns self for chaining
- [ ] Unit tests with 100% coverage

---

## Phase 2: Validation with Audience (P0)

### Task 2.1: Add Audience Mismatch Error
**Priority**: P0
**Estimate**: 30 minutes
**Files**:
- `session/jwt/service.go`

**Implementation**:
```go
var (
    ErrAudienceMismatch = errors.New("audience mismatch")
)
```

**Acceptance Criteria**:
- [ ] Error is exported
- [ ] Error message is clear

### Task 2.2: Add ValidateAccessTokenWithAudience
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `session/jwt/service.go`
- `session/jwt/service_test.go`

**Implementation**:
```go
// ValidateAccessTokenWithAudience validates and checks expected audience.
func (s *Service) ValidateAccessTokenWithAudience(
    tokenString string,
    expectedAudience string,
) (*Claims, error) {
    claims, err := s.ValidateAccessToken(tokenString)
    if err != nil {
        return nil, err
    }

    // Check audience if expected audience is specified
    if expectedAudience != "" && !claims.HasAudience(expectedAudience) {
        return nil, fmt.Errorf("%w: expected %s, got %s",
            ErrAudienceMismatch, expectedAudience, claims.Audience())
    }

    return claims, nil
}
```

**Acceptance Criteria**:
- [ ] Returns claims if audience matches
- [ ] Returns `ErrAudienceMismatch` if audience doesn't match
- [ ] Empty expected audience skips check (backward compat)
- [ ] Unit tests for all scenarios

---

## Phase 3: Generation with Audience (P0)

### Task 3.1: Add GenerateAccessTokenWithAudience
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `session/jwt/service.go`
- `session/jwt/service_test.go`

**Implementation**:
```go
// GenerateAccessTokenWithAudience creates an access token with explicit audience.
func (s *Service) GenerateAccessTokenWithAudience(
    principalID uuid.UUID,
    email, name string,
    audience string,
    scopes []string,
) (string, error) {
    claims := NewAccessClaims(s.config, principalID, email, name).
        WithAudience(audience)

    if len(scopes) > 0 {
        claims.Scopes = scopes
    }

    return s.signToken(claims)
}
```

**Acceptance Criteria**:
- [ ] Token includes audience claim
- [ ] Token includes scopes if provided
- [ ] Unit tests verify claims structure

### Task 3.2: Add GenerateBFFTokenPair
**Priority**: P0
**Estimate**: 1 hour
**Files**:
- `session/jwt/service.go`
- `session/jwt/service_test.go`

**Implementation**:
```go
// AudienceBFF is the audience for BFF (web browser) clients.
const AudienceBFF = "bff"

// AudienceAPI is the audience for programmatic API clients.
const AudienceAPI = "api"

// GenerateBFFTokenPair creates tokens for BFF (web) clients.
func (s *Service) GenerateBFFTokenPair(
    principalID uuid.UUID,
    email, name string,
) (*TokenPair, error) {
    return s.GenerateTokenPairWithAudience(principalID, email, name, AudienceBFF, nil)
}
```

**Acceptance Criteria**:
- [ ] Returns token pair with `aud: "bff"`
- [ ] No scopes in BFF tokens
- [ ] Unit tests verify audience

### Task 3.3: Add GenerateAPIToken
**Priority**: P0
**Estimate**: 1 hour
**Files**:
- `session/jwt/service.go`
- `session/jwt/service_test.go`

**Implementation**:
```go
// GenerateAPIToken creates a scoped token for API clients.
// Duration overrides the default access token expiry.
func (s *Service) GenerateAPIToken(
    principalID uuid.UUID,
    email, name string,
    scopes []string,
    duration time.Duration,
) (string, error) {
    if len(scopes) == 0 {
        return "", errors.New("at least one scope is required for API tokens")
    }

    claims := NewAccessClaims(s.config, principalID, email, name).
        WithAudience(AudienceAPI)
    claims.Scopes = scopes

    // Override expiry if duration provided
    if duration > 0 {
        claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(duration))
    }

    return s.signToken(claims)
}
```

**Acceptance Criteria**:
- [ ] Token has `aud: "api"`
- [ ] Token has scopes
- [ ] Custom duration is respected
- [ ] Error if no scopes provided

### Task 3.4: Add GenerateTokenPairWithAudience
**Priority**: P1
**Estimate**: 1 hour
**Files**:
- `session/jwt/service.go`
- `session/jwt/service_test.go`

**Implementation**:
```go
// GenerateTokenPairWithAudience creates a token pair with explicit audience.
func (s *Service) GenerateTokenPairWithAudience(
    principalID uuid.UUID,
    email, name string,
    audience string,
    scopes []string,
) (*TokenPair, error) {
    accessToken, err := s.GenerateAccessTokenWithAudience(
        principalID, email, name, audience, scopes,
    )
    if err != nil {
        return nil, fmt.Errorf("generating access token: %w", err)
    }

    family := uuid.NewString()
    refreshToken, err := s.GenerateRefreshToken(principalID, family)
    if err != nil {
        return nil, fmt.Errorf("generating refresh token: %w", err)
    }

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
    }, nil
}
```

**Acceptance Criteria**:
- [ ] Both tokens generated successfully
- [ ] Access token has audience
- [ ] Refresh token does not have audience (stateless)

---

## Phase 4: BFF Integration (P1)

### Task 4.1: Update BFF Handler to Use Audience
**Priority**: P1
**Estimate**: 2 hours
**Files**:
- `session/bff/handler.go`
- `session/bff/handler_test.go`

**Changes**:
- Update `CreateSession` to generate BFF tokens with audience
- Add config option for expected audience

**Acceptance Criteria**:
- [ ] New sessions get `aud: "bff"` tokens
- [ ] Configurable audience for custom setups

### Task 4.2: Add Audience Check to Session Middleware
**Priority**: P1
**Estimate**: 2 hours
**Files**:
- `session/bff/middleware.go`
- `session/bff/middleware_test.go`

**Changes**:
- Add optional audience validation in session middleware
- Reject requests with wrong audience

**Acceptance Criteria**:
- [ ] Middleware validates audience when configured
- [ ] Clear error response for audience mismatch

---

## Phase 5: Documentation (P2)

### Task 5.1: Update JWT Documentation
**Priority**: P2
**Estimate**: 2 hours
**Files**:
- `docs/identity/jwt.md` (create or update)

**Content**:
- Explain audience claim purpose
- Document new methods
- Migration guide

### Task 5.2: Update BFF Documentation
**Priority**: P2
**Estimate**: 1 hour
**Files**:
- `docs/bff/overview.md`

**Content**:
- Explain BFF vs API separation
- Configuration examples

---

## Summary

| Phase | Tasks | Estimate | Priority |
|-------|-------|----------|----------|
| Phase 1 | 1 | 2 hours | P0 |
| Phase 2 | 2 | 2.5 hours | P0 |
| Phase 3 | 4 | 5 hours | P0/P1 |
| Phase 4 | 2 | 4 hours | P1 |
| Phase 5 | 2 | 3 hours | P2 |
| **Total** | **11** | **~16.5 hours** | |
