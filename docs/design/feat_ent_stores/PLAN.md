# Implementation Plan: Ent Store Implementations

> **Status**: Draft
> **Target**: CoreForge v0.5.0

## Implementation Phases

### Phase 1: Ent Schema Mixins (Day 1-2)

Create reusable Ent mixins for BFF sessions and API keys.

**Files to create:**
- `identity/ent/mixin/bff_session.go`
- `identity/ent/mixin/api_key.go`

**Changes:**
1. Define BFF session mixin with all required fields
2. Define API key mixin with all required fields
3. Add indexes for common queries
4. Write godoc documentation

**Verification:**
- Mixins can be imported into test schema
- `go generate` runs without errors

### Phase 2: Encryption Utilities (Day 2)

Add encryption utilities for token-at-rest encryption.

**Files to create:**
- `session/bff/encryption.go`
- `session/bff/encryption_test.go`

**Changes:**
1. Add `Encryptor` struct with AES-256-GCM
2. Add `Encrypt(plaintext []byte) ([]byte, error)`
3. Add `Decrypt(ciphertext []byte) ([]byte, error)`
4. Write comprehensive tests

**Verification:**
```bash
go test ./session/bff/... -v -run TestEncrypt
```

### Phase 3: BFF Session Ent Store (Day 3-4)

Implement Ent-backed BFF session store.

**Files to create:**
- `session/bff/store_ent.go`
- `session/bff/store_ent_test.go`

**Changes:**
1. Define `EntClientInterface`
2. Implement `EntStore` struct
3. Implement all `Store` interface methods
4. Add automatic cleanup goroutine
5. Write unit tests with mock client

**Verification:**
```bash
go test ./session/bff/... -v -run TestEntStore
```

### Phase 4: API Key Ent Store (Day 4-5)

Implement Ent-backed API key store.

**Files to create:**
- `identity/apikey/store_ent.go`
- `identity/apikey/store_ent_test.go`

**Changes:**
1. Define `EntClientInterface`
2. Implement `EntStore` struct
3. Implement all `Store` interface methods
4. Write unit tests

**Verification:**
```bash
go test ./identity/apikey/... -v -run TestEntStore
```

### Phase 5: Example Integration (Day 5)

Create example showing how apps integrate.

**Files to create:**
- `_examples/ent-stores/main.go`
- `_examples/ent-stores/schema/bff_session.go`
- `_examples/ent-stores/schema/api_key.go`

**Changes:**
1. Minimal app showing mixin usage
2. Client wrapper implementation
3. Store initialization

### Phase 6: Documentation (Day 6)

Update documentation.

**Files to create/modify:**
- `docs/bff/ent-store.md`
- `docs/identity/api-key-store.md`
- Update README

## Rollout Plan

### Week 1: CoreForge Implementation
- Implement Phases 1-4
- Code review and merge
- Release as CoreForge v0.5.0-beta

### Week 2: App3 Integration
- Migrate App3 to use CoreForge stores
- Validate with production-like data
- Document integration patterns

### Week 3: General Availability
- Release CoreForge v0.5.0
- Update other apps
- Final documentation

## Backward Compatibility

| Scenario | Behavior |
|----------|----------|
| Apps using custom stores | Continue to work, no changes needed |
| Apps wanting to migrate | Can gradually adopt Ent stores |
| New apps | Use provided stores from day one |

## Dependencies

```
identity/ent/mixin (new)
├── entgo.io/ent (existing in CoreForge)
└── github.com/google/uuid (existing)

session/bff/store_ent.go (new)
├── session/bff/store.go (existing interface)
└── crypto/aes (stdlib)

identity/apikey/store_ent.go (new)
└── identity/apikey/service.go (existing interface)
```

## Testing Checklist

- [ ] Unit tests for encryption utilities
- [ ] Unit tests for BFF session store (mock client)
- [ ] Unit tests for API key store (mock client)
- [ ] Integration test with SQLite
- [ ] Example app compiles and runs
- [ ] Documentation is accurate
