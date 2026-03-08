# Authorization Integration Tasks

> **Status**: In Progress for v0.2.x
>
> Wire SpiceDB authorization to identity operations so that membership changes automatically sync to the authorization layer.

| Task Group | Status |
|------------|--------|
| 1. AuthZ Provider Interface | ✅ Complete |
| 2. Organization → SpiceDB Sync | ✅ Complete |
| 3. Principal → SpiceDB Sync | ✅ Complete |
| 4. Integration Tests | Pending |
| 5. SpiceDB Documentation | Pending |

---

## 1. AuthZ Provider Interface

Create an injectable authorization sync interface that identity services can use.

### 1.1 Define AuthZ Sync Interface
- [x] Create `authz/sync.go` with `RelationshipSyncer` interface:
  ```go
  type RelationshipSyncer interface {
      // Organization membership
      AddOrgMembership(ctx context.Context, principalID, orgID uuid.UUID, role string) error
      RemoveOrgMembership(ctx context.Context, principalID, orgID uuid.UUID, role string) error
      UpdateOrgMembership(ctx context.Context, principalID, orgID uuid.UUID, oldRole, newRole string) error

      // Principal lifecycle
      RegisterPrincipal(ctx context.Context, principalID uuid.UUID) error
      UnregisterPrincipal(ctx context.Context, principalID uuid.UUID) error

      // Organization lifecycle
      RegisterOrganization(ctx context.Context, orgID uuid.UUID, ownerID uuid.UUID) error
      UnregisterOrganization(ctx context.Context, orgID uuid.UUID) error
  }
  ```

### 1.2 Implement SpiceDB Syncer
- [x] Create `authz/spicedb/syncer.go` implementing `RelationshipSyncer`
- [x] Use existing `Provider.AddOrgMember()` / `RemoveOrgMember()` methods
- [ ] Add batch operations for efficiency (deferred)

### 1.3 Implement No-Op Syncer
- [x] Create `authz/noop/syncer.go` for deployments without SpiceDB
- [x] All methods return nil (silent no-op)

---

## 2. Organization → SpiceDB Sync

Wire organization service to sync membership changes to SpiceDB.

### 2.1 Inject Syncer into Organization Service
- [x] Add `RelationshipSyncer` field to `organization.Service` struct
- [x] Add `WithAuthzSyncer(syncer)` option to service constructor
- [x] Default to no-op syncer if not provided

### 2.2 Sync on Membership Changes
- [x] `AddMember()`: Call `syncer.AddOrgMembership()` after DB write
- [x] `RemoveMember()`: Call `syncer.RemoveOrgMembership()` after DB delete
- [x] `UpdateMemberRole()`: Call `syncer.UpdateOrgMembership()` after DB update

### 2.3 Sync on Organization Lifecycle
- [x] `CreateOrganization()`: Call `syncer.RegisterOrganization()` with creator as owner
- [x] `DeleteOrganization()`: Call `syncer.UnregisterOrganization()` before/after DB delete

### 2.4 Handle Sync Failures
- [x] Define error handling strategy (log and continue vs. rollback)
- [x] Add `SyncMode` config: `"strict"` (fail on sync error) vs `"eventual"` (log and continue)
- [ ] Consider background retry queue for eventual consistency (deferred)

---

## 3. Principal → SpiceDB Sync

Wire principal service to register/unregister principals in SpiceDB.

### 3.1 Inject Syncer into Principal Service
- [x] Add `RelationshipSyncer` field to `principal.Service` struct
- [x] Add `WithAuthzSyncer(syncer)` option to service constructor

### 3.2 Sync on Principal Lifecycle
- [x] `CreateHuman()`: Call `syncer.RegisterPrincipal()` after DB write
- [x] `CreateApplication()`: Call `syncer.RegisterPrincipal()` after DB write
- [x] `CreateAgent()`: Call `syncer.RegisterPrincipal()` after DB write
- [x] `CreateServicePrincipal()`: Call `syncer.RegisterPrincipal()` after DB write
- [ ] `DeletePrincipal()`: Call `syncer.UnregisterPrincipal()` before DB delete (not yet implemented)

### 3.3 Platform Admin Registration
- [x] Add method to register principal as platform admin in SpiceDB
- [x] Wire to principal service (CreateHuman with IsPlatformAdmin=true)

---

## 4. Integration Tests

Test the full authorization flow with identity operations.

### 4.1 Test Infrastructure
- [ ] Create `authz/spicedb/integration_test.go`
- [ ] Use embedded SpiceDB (in-memory) for tests
- [ ] Create test helpers for setting up principals and orgs

### 4.2 Organization Membership Tests
- [ ] Test: Create org → owner automatically has `manage` permission
- [ ] Test: Add member → member has `view` permission
- [ ] Test: Add admin → admin has `manage` permission
- [ ] Test: Remove member → member loses all permissions
- [ ] Test: Update role (member → admin) → permissions change accordingly

### 4.3 Principal Lifecycle Tests
- [ ] Test: Create principal → principal exists in SpiceDB
- [ ] Test: Delete principal → all relationships removed

### 4.4 Cross-Service Integration Tests
- [ ] Create `identity/integration_test.go` for full workflow tests
- [ ] Test: Signup flow → personal org created → user is owner → can manage org
- [ ] Test: Create team org → invite member → accept → member can view
- [ ] Test: Platform admin → can access any org

### 4.5 Error Handling Tests
- [ ] Test: SpiceDB unavailable in strict mode → operation fails
- [ ] Test: SpiceDB unavailable in eventual mode → operation succeeds, logged

---

## 5. SpiceDB Documentation

Create comprehensive documentation for SpiceDB setup and usage.

### 5.1 Setup Guide
- [ ] Create `docs/authorization/spicedb-setup.md`
- [ ] Document embedded mode (development)
  - In-memory datastore
  - PostgreSQL datastore
- [ ] Document remote mode (production)
  - Connection configuration
  - TLS setup
  - Token authentication
- [ ] Include Docker Compose example for local SpiceDB

### 5.2 Configuration Reference
- [ ] Document all `Config` fields with examples
- [ ] Environment variable mapping
- [ ] Example YAML configuration

### 5.3 Schema Guide
- [ ] Create `docs/authorization/spicedb-schema.md`
- [ ] Explain `BaseSchema` (principal, organization, platform)
- [ ] Explain `ResourceSchema()` helper for custom resources
- [ ] Document permission inheritance model
- [ ] Provide examples for common patterns:
  - Organization-scoped resources
  - User-owned resources
  - Shared resources

### 5.4 Integration Guide
- [ ] Create `docs/authorization/integration.md`
- [ ] How to wire SpiceDB to identity services
- [ ] How to perform authorization checks in handlers
- [ ] Middleware usage examples
- [ ] Best practices for permission design

### 5.5 Migration Guide
- [ ] Document migrating from Casbin to SpiceDB
- [ ] When to use which provider
- [ ] Comparison table (Casbin vs SpiceDB vs Simple)

---

## Implementation Order

1. **Phase 1: Interface & SpiceDB Syncer** (Tasks 1.1-1.3)
   - Define the sync interface
   - Implement SpiceDB and no-op syncers
   - Unit tests for syncers

2. **Phase 2: Organization Wiring** (Tasks 2.1-2.4)
   - Inject syncer into org service
   - Wire all membership operations
   - Handle error modes

3. **Phase 3: Principal Wiring** (Tasks 3.1-3.3)
   - Inject syncer into principal service
   - Wire lifecycle operations

4. **Phase 4: Integration Tests** (Tasks 4.1-4.5)
   - Build test infrastructure
   - Write comprehensive tests
   - Verify full workflows

5. **Phase 5: Documentation** (Tasks 5.1-5.5)
   - Setup and configuration docs
   - Schema and integration guides
   - Migration guidance

---

## Verification

After implementation, verify:

```bash
# Run unit tests
go test -v ./authz/...

# Run integration tests (requires embedded SpiceDB)
go test -v ./authz/spicedb/... -tags=integration
go test -v ./identity/... -tags=integration

# Build docs
cd docs && mkdocs build

# Manual verification
# 1. Start with embedded SpiceDB
# 2. Create user via signup
# 3. Verify user's personal org has correct permissions
# 4. Create team org, add member
# 5. Verify member permissions
```

---

## Dependencies

- `github.com/authzed/authzed-go` - SpiceDB Go client
- `github.com/authzed/spicedb` - Embedded SpiceDB server
- `github.com/authzed/grpcutil` - gRPC utilities

All dependencies already added to go.mod in v0.2.0.
