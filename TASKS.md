# CoreForge Tasks

## Implementation Tasks (Pre-App1)

These tasks must be completed before App1 integration.

### P0: Marketplace Module (Critical - Enables Course Sales)

The marketplace module has types and interfaces defined but lacks concrete implementations.

#### Phase 1: Ent Schemas ✅

- [x] Create `identity/ent/schema/listing.go` - Product/course listing entity
- [x] Create `identity/ent/schema/license.go` - License entitlement entity
- [x] Create `identity/ent/schema/subscription.go` - Subscription entity
- [x] Create `identity/ent/schema/seat_assignment.go` - Seat assignment entity
- [x] Run `go generate ./identity/ent/...` to generate Ent code
- [x] Add database migrations for new tables

#### Phase 2: Service Implementations ✅

- [x] Implement `ListingService` in `marketplace/listing_service.go`
  - [x] `Create(ctx, input)` - Create new listing
  - [x] `Get(ctx, id)` - Get listing by ID
  - [x] `List(ctx, opts)` - List with filtering
  - [x] `Update(ctx, id, input)` - Update listing
  - [x] `Delete(ctx, id)` - Delete listing
  - [x] `Publish(ctx, id)` - Publish listing
  - [x] `Archive(ctx, id)` - Archive listing

- [x] Implement `LicenseService` in `marketplace/license_service.go`
  - [x] `Grant(ctx, input)` - Grant license to org
  - [x] `Revoke(ctx, id)` - Revoke license
  - [x] `Get(ctx, id)` - Get license by ID
  - [x] `ListForOrg(ctx, orgID)` - List org's licenses
  - [x] `AssignSeat(ctx, licenseID, principalID)` - Assign seat
  - [x] `UnassignSeat(ctx, assignmentID)` - Remove seat assignment
  - [x] `GetAvailableSeats(ctx, licenseID)` - Check available seats

- [x] Implement `SubscriptionService` in `marketplace/subscription_service.go`
  - [x] `Create(ctx, input)` - Create subscription
  - [x] `Get(ctx, id)` - Get subscription
  - [x] `Cancel(ctx, id)` - Cancel subscription
  - [x] `UpdateStatus(ctx, id, status)` - Update status
  - [x] `ListForOrg(ctx, orgID)` - List org subscriptions

#### Phase 3: Stripe Integration ✅

- [x] Create `marketplace/stripe/` package
- [x] Implement `CheckoutService` in `marketplace/stripe/checkout.go`
  - [x] `CreateCheckoutSession(ctx, input)` - Create Stripe Checkout
  - [ ] `CreatePortalSession(ctx, customerID)` - Customer portal (deferred)
- [x] Implement webhook handlers in `marketplace/stripe/webhooks.go`
  - [x] `HandleCheckoutCompleted` - Grant license on purchase
  - [x] `HandleSubscriptionCreated` - Create subscription record
  - [x] `HandleSubscriptionUpdated` - Update subscription status
  - [x] `HandleSubscriptionDeleted` - Handle cancellation
  - [x] `HandleInvoicePaid` - Record payment
  - [x] `HandleInvoicePaymentFailed` - Handle failed payment
- [x] Implement Stripe Connect in `marketplace/stripe/connect.go`
  - [x] `CreateConnectAccount(ctx, orgID)` - Create seller account
  - [x] `CreateAccountLink(ctx, accountID)` - Onboarding link
  - [x] `CreatePayout(ctx, accountID, amount)` - Trigger payout

#### Phase 4: API Handlers ✅

- [x] Create `marketplace/handlers.go` with Huma handlers
  - [x] `POST /api/v1/listings` - Create listing
  - [x] `GET /api/v1/listings` - List listings
  - [x] `GET /api/v1/listings/{id}` - Get listing
  - [x] `PATCH /api/v1/listings/{id}` - Update listing
  - [x] `DELETE /api/v1/listings/{id}` - Delete listing
  - [x] `POST /api/v1/listings/{id}/publish` - Publish
  - [x] `POST /api/v1/listings/{id}/archive` - Archive
  - [x] `POST /api/v1/checkout` - Create checkout session
  - [x] `GET /api/v1/licenses` - List licenses
  - [x] `GET /api/v1/licenses/{id}` - Get license
  - [x] `POST /api/v1/licenses/{license_id}/seats` - Assign seat
  - [x] `DELETE /api/v1/licenses/{license_id}/seats/{principal_id}` - Unassign seat
  - [x] `GET /api/v1/licenses/{license_id}/seats` - List seats
  - [ ] `POST /api/v1/webhooks/stripe` - Stripe webhook endpoint (requires raw body handling)

#### Phase 5: Tests

- [x] Add `marketplace/listing_service_test.go`
- [x] Add `marketplace/license_service_test.go`
- [x] Add `marketplace/subscription_service_test.go`
- [ ] Add `marketplace/stripe/checkout_test.go`
- [ ] Add `marketplace/stripe/webhooks_test.go`
- [ ] Add `marketplace/handlers_test.go`

### P2: Rate Limiting Redis Backend

- [ ] Implement Redis storage in `session/ratelimit/redis.go`
  - [ ] `NewRedisStorage(client)` - Create Redis storage
  - [ ] `Allow(ctx, key, limit, window)` - Check and increment
  - [ ] `Reset(ctx, key)` - Reset counter
  - [ ] `GetUsage(ctx, key)` - Get current usage
- [ ] Add `session/ratelimit/redis_test.go` with integration tests
- [ ] Add Redis connection configuration

### P2: Observability Integration

- [ ] Add marketplace metrics to `observability/`
  - [ ] `marketplace.checkouts_total` - Checkout attempts
  - [ ] `marketplace.purchases_total` - Successful purchases
  - [ ] `marketplace.licenses_granted_total` - Licenses granted
  - [ ] `marketplace.seats_assigned_total` - Seats assigned
- [ ] Add Stripe webhook metrics
  - [ ] `marketplace.webhook_received_total` - Webhooks received
  - [ ] `marketplace.webhook_processed_total` - Webhooks processed
  - [ ] `marketplace.webhook_errors_total` - Webhook errors

---

## GoDoc Documentation

### Overview

GoDoc coverage analysis identified ~600+ exported items missing documentation comments. This excludes auto-generated ent code (~100+ items) which should not be manually documented.

### Priority 1: Public API Types (High Impact)

These are used directly by consumers of the library.

#### Configuration Types (`identity/coreauth/config.go`)

- [ ] `Config` - Main configuration struct
- [ ] `ClientConfig` - OAuth client configuration
- [ ] `Duration` - Custom duration type for config parsing
- [ ] `FederationConfig` - Federation settings
- [ ] `TokenConfig` - Token generation settings

#### Session Middleware (`session/middleware/`)

- [ ] `ClaimsFromContext` - Extract JWT claims from context
- [ ] `UserIDFromContext` - Extract user ID from context
- [ ] `OrganizationIDFromContext` - Extract org ID from context
- [ ] `PrincipalIDFromContext` - Extract principal ID from context
- [ ] `RoleFromContext` - Extract role from context
- [ ] `PermissionsFromContext` - Extract permissions from context
- [ ] `HTTPAuth` - HTTP authentication middleware
- [ ] `HTTPAuthOptional` - Optional HTTP auth middleware
- [ ] `RequireRole` - Role requirement middleware
- [ ] `RequirePermission` - Permission requirement middleware
- [ ] `RequireAnyRole` - Any role requirement middleware
- [ ] `RequireAnyPermission` - Any permission requirement middleware
- [ ] `RequirePlatformAdmin` - Platform admin requirement
- [ ] `RequireOrganization` - Organization requirement middleware
- [ ] `ChiAuth` - Chi router auth middleware
- [ ] `ChiAuthOptional` - Chi router optional auth
- [ ] `ChiRequireRole` - Chi router role requirement
- [ ] `RequireAPIKey` - API key requirement middleware
- [ ] `RequireScope` - OAuth scope requirement middleware

#### JWT Service (`session/jwt/`)

- [ ] `Claims` - JWT claims structure
- [ ] `TokenPair` - Access/refresh token pair
- [ ] `CNFClaim` - Confirmation claim for DPoP
- [ ] `DefaultConfig` - Default JWT configuration
- [ ] `AccessTokenTTL` - Access token TTL method
- [ ] `RefreshTokenTTL` - Refresh token TTL method
- [ ] `GenerateRefreshToken` - Refresh token generation
- [ ] `GenerateTokenPairLegacy` - Legacy token pair generation
- [ ] `WithDPoPBinding` - DPoP binding option
- [ ] `DPoPThumbprint` - DPoP thumbprint extraction
- [ ] `ComputeTokenHash` - Token hash computation

### Priority 2: Authorization Module (Security Critical)

#### Core Types (`authz/`)

- [ ] `Authorizer` - Main authorizer interface
- [ ] `OrgAuthorizer` - Organization-scoped authorizer
- [ ] `RelationshipSyncer` - Relationship synchronization interface
- [ ] `ResourceExtractor` - Resource ID extraction function type

#### Middleware (`authz/middleware.go`)

- [ ] `RequireAction` - Action requirement middleware
- [ ] `RequireAllActions` - All actions requirement
- [ ] `RequireAnyAction` - Any action requirement
- [ ] `RequireMembership` - Membership requirement
- [ ] `RequireResourceAction` - Resource action requirement
- [ ] `RequireRole` - Role requirement
- [ ] `WithResourceID` - Resource ID injection
- [ ] `Error` - Error handling method

#### SpiceDB Provider (`authz/spicedb/`)

- [ ] `BaseSchema` - Base SpiceDB schema constant
- [ ] `ResourceSchema` - Resource schema generation
- [ ] `RegisterPrincipal` - Principal registration
- [ ] `UpdateOrgMembership` - Org membership update

### Priority 3: Identity & Authentication

#### CoreAuth Server (`identity/coreauth/`)

- [ ] `NewEmbedded` - Create embedded auth server
- [ ] `AuthenticationProvider` - Authentication provider interface
- [ ] `IdentityProvider` - Identity provider interface
- [ ] `OAuthProvider` - OAuth provider interface
- [ ] `OAuthClientStore` - OAuth client storage interface
- [ ] `Identity` - Identity type
- [ ] `Providers` - Provider collection
- [ ] `NewProviders` - Create provider collection
- [ ] `Storage` - Storage interface
- [ ] `MemoryStorage` - In-memory storage
- [ ] `CleanupExpired` - Expired token cleanup

#### Session Management (`identity/coreauth/session.go`)

- [ ] `SessionProvider` - Session provider interface
- [ ] `DefaultSessionProvider` - Default session implementation
- [ ] `AuthorizationSession` - Authorization session type
- [ ] `GetUserClaims` - Get user claims from session
- [ ] `HasConsent` - Check consent status
- [ ] `SaveConsent` - Save user consent
- [ ] `WithUserIDHeader` - User ID header option

#### OAuth Types (`identity/coreauth/types_oauth.go`, `identity/oauth/types_oauth.go`)

- [ ] `TokenInput` - Token request input
- [ ] `TokenResponse` - Token response structure

#### SCIM (`identity/scim/`)

- [ ] `Provider` - SCIM provider interface
- [ ] `Store` - SCIM store interface
- [ ] `ListUsers` - List users method
- [ ] `ListGroups` - List groups method
- [ ] `ToSCIMError` - Error conversion
- [ ] `CompositeAuthorizationHook` - Composite auth hook
- [ ] `RoleBasedAuthorizationHook` - Role-based auth hook
- [ ] `ScopedAuthorizationHook` - Scoped auth hook
- [ ] `PrincipalUserMapper` - Principal to user mapping
- [ ] `EnterpriseExtension` - Enterprise SCIM extension
- [ ] `ServiceProviderConfig` - SCIM service provider config
- [ ] `MultiValue` - Multi-value SCIM type

#### Password Utilities (`identity/password.go`)

- [ ] `HashPassword` - Password hashing
- [ ] `VerifyPassword` - Password verification
- [ ] `NeedsRehash` - Check if rehash needed
- [ ] `DefaultArgon2idParams` - Default Argon2id parameters

### Priority 4: Row-Level Security (`rls/`)

- [ ] `Middleware` - RLS middleware type
- [ ] `DBWithRLS` - Database wrapper with RLS
- [ ] `EntDriver` - Ent driver with RLS
- [ ] `EntHook` - Ent hook for RLS
- [ ] `ContextInjector` - Context injection type
- [ ] `Executor` - SQL executor with RLS
- [ ] `TenantIDFromContext` - Get tenant from context
- [ ] `UserIDFromContext` - Get user from context
- [ ] `BypassRLS` - Bypass RLS for admin operations
- [ ] `SetTenant` - Set tenant in context
- [ ] `WithTenant` - Transaction with tenant
- [ ] `WithTenantFromContext` - Transaction with context tenant
- [ ] `RequireTenant` - Require tenant middleware
- [ ] `InjectContext` - Inject RLS context
- [ ] `SetContextFromContext` - Set context from existing
- [ ] `GenerateMigrationSQL` - Generate RLS migration SQL
- [ ] `CoreForgeTables` - Tables requiring RLS

#### Testing Helpers (`rls/testing.go`)

- [ ] `AsUser` - Run as specific user
- [ ] `WithoutRLS` - Run without RLS
- [ ] `AssertTenantIsolation` - Assert tenant isolation

### Priority 5: Session Components

#### BFF (Backend for Frontend) (`session/bff/`)

- [ ] `Session` - BFF session type
- [ ] `Store` - Session store interface
- [ ] `MemoryStore` - In-memory session store
- [ ] `NeedsRefresh` - Check if refresh needed
- [ ] `GetSessionID` - Get session ID from cookie
- [ ] `OptionalSessionMiddleware` - Optional session middleware
- [ ] `RequireSessionMiddleware` - Required session middleware
- [ ] `OriginMiddleware` - Origin validation middleware
- [ ] `APIProxyMiddleware` - API proxy middleware
- [ ] `RefreshHandler` - Token refresh handler
- [ ] `TokenResponse` - BFF token response

#### DPoP (`session/dpop/`)

- [ ] `ProofHeader` - DPoP proof header type
- [ ] `SerializedKeyPair` - Serialized key pair
- [ ] `NewProofClaims` - Create proof claims
- [ ] `ComputeAccessTokenHash` - Compute access token hash
- [ ] `ComputeThumbprint` - Compute key thumbprint
- [ ] `GenerateKeyPair` - Generate DPoP key pair
- [ ] `CreateProof` - Create DPoP proof
- [ ] `ParseProof` - Parse DPoP proof
- [ ] `VerifyTokenBinding` - Verify token binding

#### Rate Limiting (`session/ratelimit/`)

- [ ] `KeyFunc` - Rate limit key function type
- [ ] `LimitResolver` - Limit resolver interface
- [ ] `MemoryStorage` - In-memory rate limit storage
- [ ] `RedisStorage` - Redis rate limit storage
- [ ] `NewRedisStorage` - Create Redis storage
- [ ] `CoreAPIResolver` - CoreAPI rate limit resolver
- [ ] `GetPolicyForRequest` - Get policy for request
- [ ] `ClientKey` - Client-based key function
- [ ] `PrincipalKey` - Principal-based key function
- [ ] `EndpointKey` - Endpoint-based key function
- [ ] `CompositeKey` - Composite key function

#### OAuth Handlers (`session/oauth/`)

- [ ] `MemoryStateStore` - In-memory OAuth state store
- [ ] `UserInfo` - OAuth user info type

### Priority 6: Other Components

#### CoreAPI (`coreapi/`)

- [ ] `RateLimits` - Rate limits configuration
- [ ] `MostGranularLimit` - Get most granular limit
- [ ] `MemoryPolicyStore` - In-memory policy store
- [ ] `NewMemoryPolicyStore` - Create memory policy store

#### Contract (`contract/`)

- [ ] `WithLogger` - Logger option
- [ ] `ToContractError` - Error conversion
- [ ] `RequireAuth` - Auth requirement middleware
- [ ] `Middleware` - Contract middleware
- [ ] `RecordAuditEvent` - Record audit event
- [ ] `StartSync` - Start provider sync
- [ ] `SyncLagSeconds` - Get sync lag
- [ ] `MemoryStore` - In-memory audit store

#### Marketplace (`marketplace/`)

- [ ] `MarketplaceSchema` - Marketplace SpiceDB schema
- [ ] `MergeSchema` - Merge schemas
- [ ] `SeatsRemaining` - Get remaining seats
- [ ] `SyncLicense` - Sync license to SpiceDB
- [ ] `SyncLicenseRevocation` - Sync license revocation
- [ ] `SyncListing` - Sync listing
- [ ] `SyncSubscription` - Sync subscription
- [ ] `SyncSeatAssignment` - Sync seat assignment
- [ ] `SyncSeatUnassignment` - Sync seat unassignment

#### Observability (`observability/`)

- [ ] `New` - Create observability provider
- [ ] `ConfigFromEnv` - Load config from environment
- [ ] `Middleware` - Observability middleware
- [ ] `SlogHandler` - Get slog handler

#### Feature Flags (`featureflags/`)

- [ ] `MemoryStore` - In-memory feature flag store

#### Ent Mixins (`identity/ent/mixin/`)

- [ ] `BaseMixin` - Base entity mixin
- [ ] `UUIDMixin` - UUID field mixin
- [ ] `TimestampMixin` - Timestamp fields mixin
- [ ] `UserBase` - User base mixin
- [ ] `OrganizationBase` - Organization base mixin
- [ ] `MembershipBase` - Membership base mixin
- [ ] `PrincipalMixin` - Principal mixin
- [ ] `HumanMixin` - Human entity mixin
- [ ] `AgentMixin` - Agent entity mixin
- [ ] `ApplicationMixin` - Application mixin
- [ ] `ServicePrincipalMixin` - Service principal mixin
- [ ] `PrincipalMembershipMixin` - Principal membership mixin
- [ ] `OAuthAccountMixin` - OAuth account mixin
- [ ] `RefreshTokenMixin` - Refresh token mixin

### Excluded from Documentation

The following are auto-generated and should NOT be manually documented:

- `identity/ent/*.go` (except `mixin/` and `schema/`)
- `identity/ent/hook/hook.go` - Generated hook types
- `identity/ent/privacy/privacy.go` - Generated privacy rules
- `identity/ent/migrate/` - Generated migrations
- `identity/ent/internal/` - Internal generated code

### Documentation Guidelines

1. **Start comment with item name**: `// Config holds the main configuration...`
2. **Describe purpose, not implementation**: Focus on what it does, not how
3. **Document parameters for functions**: Explain each parameter's purpose
4. **Document return values**: Explain what is returned and when errors occur
5. **Add examples for complex APIs**: Use `// Example:` blocks where helpful
6. **Cross-reference related items**: Use `// See also: OtherType` when relevant

### Example Documentation

```go
// Config holds the main configuration for the CoreAuth server.
// It includes settings for OAuth clients, token generation, and federation.
type Config struct {
    // Clients defines the OAuth 2.0 clients that can authenticate.
    Clients []ClientConfig `json:"clients" yaml:"clients"`

    // Token configures access and refresh token generation.
    Token TokenConfig `json:"token" yaml:"token"`

    // Federation configures cross-application identity federation.
    Federation FederationConfig `json:"federation" yaml:"federation"`
}

// HTTPAuth returns middleware that requires a valid JWT in the Authorization header.
// It extracts claims and adds them to the request context.
// Returns 401 Unauthorized if the token is missing or invalid.
// Returns 403 Forbidden if the token is expired.
//
// See also: HTTPAuthOptional, ClaimsFromContext
func HTTPAuth(jwtService *jwt.Service) func(http.Handler) http.Handler
```
