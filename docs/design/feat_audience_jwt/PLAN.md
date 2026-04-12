# Implementation Plan: JWT Audience Validation

> **Status**: Draft
> **Target**: CoreForge v0.5.0

## Implementation Phases

### Phase 1: Claims Helpers (Day 1)

Add helper methods to the Claims struct for audience handling.

**Files to modify:**
- `session/jwt/claims.go`

**Changes:**
1. Add `Audience() string` method
2. Add `HasAudience(aud string) bool` method
3. Add `WithAudience(audiences ...string) *Claims` method
4. Add unit tests

**Verification:**
```bash
go test ./session/jwt/... -v -run TestAudience
```

### Phase 2: Validation with Audience (Day 1)

Add audience-aware token validation.

**Files to modify:**
- `session/jwt/service.go`
- `session/jwt/errors.go` (new file for errors)

**Changes:**
1. Add `ErrAudienceMismatch` error
2. Add `ValidateAccessTokenWithAudience(token, audience string) (*Claims, error)`
3. Add unit tests for validation

**Verification:**
```bash
go test ./session/jwt/... -v -run TestValidateWithAudience
```

### Phase 3: Generation with Audience (Day 2)

Add audience-aware token generation methods.

**Files to modify:**
- `session/jwt/service.go`

**Changes:**
1. Add `GenerateAccessTokenWithAudience(principalID, email, name, audience string, scopes []string)`
2. Add `GenerateBFFTokenPair(principalID, email, name string)`
3. Add `GenerateAPIToken(principalID, email, name string, scopes []string, duration time.Duration)`
4. Add `GenerateTokenPairWithAudience(principalID, email, name, audience string, scopes []string)`
5. Add unit tests

**Verification:**
```bash
go test ./session/jwt/... -v -run TestGenerate
```

### Phase 4: BFF Integration (Day 2)

Update BFF session middleware to use audience validation.

**Files to modify:**
- `session/bff/middleware.go`
- `session/bff/handler.go`

**Changes:**
1. Add audience parameter to session middleware
2. Update CreateSession to use audience-aware tokens
3. Add integration tests

**Verification:**
```bash
go test ./session/bff/... -v
```

### Phase 5: Documentation (Day 3)

Update documentation.

**Files to create/modify:**
- `docs/bff/audience.md`
- `docs/identity/api-tokens.md`
- Update README with new methods

## Rollout Plan

### Week 1: CoreForge Implementation
- Implement Phases 1-4
- Code review and merge
- Release as CoreForge v0.5.0-beta

### Week 2: App Integration
- App3: Update to use audience-aware methods
- Test BFF/API separation
- Monitor for issues

### Week 3: General Availability
- Release CoreForge v0.5.0
- Update other apps (App1, Dashforge)
- Documentation updates

## Backward Compatibility

| Scenario | Behavior |
|----------|----------|
| Old token (no audience) + `ValidateAccessToken` | ✅ Works |
| Old token (no audience) + `ValidateAccessTokenWithAudience` | ❌ Fails (audience mismatch) |
| New token (with audience) + `ValidateAccessToken` | ✅ Works |
| New token (with audience) + `ValidateAccessTokenWithAudience` | ✅ Works if audience matches |

## Dependencies

```
session/jwt (modify)
├── github.com/golang-jwt/jwt/v5 (existing)
└── github.com/google/uuid (existing)

session/bff (modify)
├── session/jwt (existing)
└── No new dependencies
```

## Testing Checklist

- [ ] Unit tests for Claims helpers
- [ ] Unit tests for audience validation
- [ ] Unit tests for audience generation
- [ ] Integration tests for BFF middleware
- [ ] Integration tests for Bearer middleware
- [ ] End-to-end test: BFF token rejected on API path
- [ ] End-to-end test: API token rejected on BFF path
- [ ] Migration test: Old tokens still work
