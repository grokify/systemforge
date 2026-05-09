# PRD: JWT Audience Validation

> **Status**: Draft
> **Target**: SystemForge v0.5.0

## Overview

Add audience (`aud`) claim validation to SystemForge's JWT service to enable path-bound token separation. This prevents credential replay attacks where tokens obtained for web browser access (BFF) are used for direct API access, and vice versa.

## Problem Statement

Currently, SystemForge JWT tokens can be used interchangeably across all endpoints. This creates security risks:

1. **Token Replay**: User extracts JWT from browser DevTools, replays against raw API
2. **Bypassed Restrictions**: UI-enforced limits (rate limits, feature gates) can be bypassed
3. **Audit Gap**: No distinction between WebUI and programmatic access in logs
4. **Compliance Risk**: Some regulations require explicit API access grants

## Goals

1. **Token Isolation**: BFF tokens only work on BFF paths, API tokens only on API paths
2. **Explicit API Grants**: Users must explicitly create API tokens with scoped permissions
3. **Backward Compatibility**: Existing tokens continue to work during migration
4. **Zero Friction for Web Users**: No UX changes for normal web app usage

## Non-Goals

- Token encryption (handled separately by DPoP)
- Per-endpoint scopes (handled by existing scope middleware)
- Token revocation (handled by session/apikey stores)

## User Stories

### WebUI Users

**US-1**: As a web user, my session token cannot be used outside the web application context.

**US-2**: As a web user, I don't need to do anything different - authentication works seamlessly.

### API Developers

**US-3**: As a developer, I can create API tokens that work only on `/api/v1/*` endpoints.

**US-4**: As a developer, my API tokens are rejected if used on BFF endpoints.

**US-5**: As a developer, I get clear error messages when using the wrong token type.

### Platform Operators

**US-6**: As an operator, I can configure which audiences are accepted on which paths.

**US-7**: As an operator, I can audit token usage by audience type.

## Target Applications

| Application | BFF Path | API Path | Migration Status |
|-------------|----------|----------|------------------|
| App3 | `/bff/v1/*` | `/api/v1/*` | New implementation |
| App1 | `/bff/*` | `/api/*` | Future migration |
| Dashforge | `/bff/*` | `/api/*` | Future migration |

## Success Metrics

1. **Security**: 0 successful BFF-to-API token replay attacks
2. **Adoption**: All 3 apps using audience validation within 6 months
3. **Developer Experience**: <5% increase in auth-related support tickets

## Dependencies

- SystemForge `session/jwt` package
- SystemForge `session/bff` package
- App-level router configuration

## Risks

| Risk | Mitigation |
|------|------------|
| Breaking existing tokens | Provide migration period with optional audience validation |
| Developer confusion | Clear documentation and error messages |
| Performance overhead | Audience check is O(1) string comparison |
