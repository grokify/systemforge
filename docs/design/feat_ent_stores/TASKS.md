# Tasks: Ent Store Implementations

> **Status**: Draft
> **Target**: CoreForge v0.5.0

## Priority Levels

- **P0**: Critical path, blocks other work
- **P1**: High priority, core functionality
- **P2**: Important, enhances security/UX
- **P3**: Nice to have, future iteration

---

## Phase 1: Ent Schema Mixins (P0)

### Task 1.1: Create BFF Session Mixin
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `identity/ent/mixin/bff_session.go`

**Implementation**:
```go
package mixin

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "entgo.io/ent/schema/mixin"
    "github.com/google/uuid"
)

// BFFSession provides common fields for BFF session entities.
type BFFSession struct {
    mixin.Schema
}

// Fields of the BFFSession mixin.
func (BFFSession) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").NotEmpty().Unique().Immutable(),
        field.UUID("user_id", uuid.UUID{}),
        field.UUID("organization_id", uuid.UUID{}).Optional().Nillable(),
        field.Bytes("access_token_encrypted").Sensitive(),
        field.Bytes("refresh_token_encrypted").Sensitive(),
        field.Time("access_token_expires_at"),
        field.Time("refresh_token_expires_at"),
        field.Bytes("dpop_key_pair_encrypted").Optional().Sensitive(),
        field.String("dpop_thumbprint").Optional().MaxLen(64),
        field.String("ip_address").Optional().MaxLen(45),
        field.String("user_agent").Optional().MaxLen(500),
        field.JSON("metadata", map[string]string{}).Optional(),
        field.Time("last_accessed_at").Default(time.Now),
        field.Time("expires_at"),
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

// Indexes of the BFFSession mixin.
func (BFFSession) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("user_id"),
        index.Fields("expires_at"),
        index.Fields("organization_id"),
    }
}
```

**Acceptance Criteria**:
- [ ] Mixin compiles without errors
- [ ] All fields from bff.Session are represented
- [ ] Indexes support common queries (by user, by expiry)
- [ ] Sensitive fields marked with `.Sensitive()`

### Task 1.2: Create API Key Mixin
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `identity/ent/mixin/api_key.go`

**Implementation**:
```go
package mixin

import (
    "time"

    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
    "entgo.io/ent/schema/mixin"
    "github.com/google/uuid"
)

// APIKey provides common fields for API key entities.
type APIKey struct {
    mixin.Schema
}

// Fields of the APIKey mixin.
func (APIKey) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("id", uuid.UUID{}).Default(uuid.New).Immutable(),
        field.String("name").NotEmpty().MaxLen(100),
        field.String("prefix").NotEmpty().MaxLen(20),
        field.String("key_hash").NotEmpty().MaxLen(64).Sensitive(),
        field.UUID("owner_id", uuid.UUID{}),
        field.UUID("organization_id", uuid.UUID{}).Optional().Nillable(),
        field.Strings("scopes").Default([]string{}),
        field.String("description").Optional().MaxLen(500),
        field.Enum("environment").Values("live", "test").Default("live"),
        field.Time("expires_at").Optional().Nillable(),
        field.Time("last_used_at").Optional().Nillable(),
        field.String("last_used_ip").Optional().MaxLen(45),
        field.Bool("revoked").Default(false),
        field.Time("revoked_at").Optional().Nillable(),
        field.String("revoked_reason").Optional().MaxLen(500),
        field.JSON("metadata", map[string]string{}).Optional(),
        field.Time("created_at").Default(time.Now).Immutable(),
        field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
    }
}

// Indexes of the APIKey mixin.
func (APIKey) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("owner_id"),
        index.Fields("organization_id"),
        index.Fields("prefix"),
        index.Fields("key_hash").Unique(),
        index.Fields("environment"),
    }
}
```

**Acceptance Criteria**:
- [ ] Mixin compiles without errors
- [ ] All fields from apikey.APIKey are represented
- [ ] Unique index on key_hash
- [ ] Sensitive field marked for key_hash

---

## Phase 2: Encryption Utilities (P0)

### Task 2.1: Create Encryptor Struct
**Priority**: P0
**Estimate**: 2 hours
**Files**:
- `session/bff/encryption.go`
- `session/bff/encryption_test.go`

**Implementation**:
```go
package bff

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "errors"
    "io"
)

var (
    ErrInvalidKeySize    = errors.New("encryption key must be 32 bytes for AES-256")
    ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// Encryptor provides AES-256-GCM encryption for tokens at rest.
type Encryptor struct {
    gcm cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given 32-byte key.
func NewEncryptor(key []byte) (*Encryptor, error) {
    if len(key) != 32 {
        return nil, ErrInvalidKeySize
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, e.gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
    if len(ciphertext) < e.gcm.NonceSize() {
        return nil, ErrCiphertextTooShort
    }

    nonce, ciphertext := ciphertext[:e.gcm.NonceSize()], ciphertext[e.gcm.NonceSize():]
    return e.gcm.Open(nil, nonce, ciphertext, nil)
}
```

**Acceptance Criteria**:
- [ ] Encrypt/Decrypt round-trip succeeds
- [ ] Invalid key size returns error
- [ ] Corrupted ciphertext returns error
- [ ] Different encryptions of same plaintext produce different ciphertext (nonce)
- [ ] 100% test coverage

---

## Phase 3: BFF Session Ent Store (P0)

### Task 3.1: Define EntClientInterface
**Priority**: P0
**Estimate**: 1 hour
**Files**:
- `session/bff/store_ent.go`

**Implementation**:
```go
// EntClientInterface abstracts the Ent client to avoid import cycles.
// Apps implement this interface wrapping their generated Ent client.
type EntClientInterface interface {
    CreateBFFSession(ctx context.Context, session *Session) error
    GetBFFSession(ctx context.Context, id string) (*Session, error)
    UpdateBFFSession(ctx context.Context, session *Session) error
    DeleteBFFSession(ctx context.Context, id string) error
    DeleteBFFSessionsByUserID(ctx context.Context, userID string) (int, error)
    TouchBFFSession(ctx context.Context, id string) error
    CleanupExpiredBFFSessions(ctx context.Context, limit int) (int, error)
}
```

**Acceptance Criteria**:
- [ ] Interface covers all Store operations
- [ ] No Ent imports in CoreForge
- [ ] Clear godoc for each method

### Task 3.2: Implement EntStore
**Priority**: P0
**Estimate**: 4 hours
**Files**:
- `session/bff/store_ent.go`

**Implementation**:
```go
// EntStoreConfig configures the Ent-backed session store.
type EntStoreConfig struct {
    Client           EntClientInterface
    EncryptionKey    []byte
    CleanupInterval  time.Duration
    CleanupBatchSize int
}

// EntStore implements Store using Ent.
type EntStore struct {
    config    EntStoreConfig
    encryptor *Encryptor
    cleanup   *time.Ticker
    done      chan struct{}
}

// NewEntStore creates a new Ent-backed session store.
func NewEntStore(config EntStoreConfig) (*EntStore, error) {
    if config.Client == nil {
        return nil, errors.New("client is required")
    }
    if len(config.EncryptionKey) != 32 {
        return nil, ErrInvalidKeySize
    }

    encryptor, err := NewEncryptor(config.EncryptionKey)
    if err != nil {
        return nil, fmt.Errorf("creating encryptor: %w", err)
    }

    store := &EntStore{
        config:    config,
        encryptor: encryptor,
        done:      make(chan struct{}),
    }

    if config.CleanupInterval > 0 {
        store.startCleanup()
    }

    return store, nil
}

// Close stops the cleanup goroutine.
func (s *EntStore) Close() error {
    if s.cleanup != nil {
        s.cleanup.Stop()
        close(s.done)
    }
    return nil
}
```

**Acceptance Criteria**:
- [ ] All Store interface methods implemented
- [ ] Tokens encrypted before storage
- [ ] Tokens decrypted after retrieval
- [ ] Automatic cleanup goroutine starts if configured
- [ ] Close() stops cleanup gracefully

### Task 3.3: Write EntStore Tests
**Priority**: P0
**Estimate**: 3 hours
**Files**:
- `session/bff/store_ent_test.go`

**Acceptance Criteria**:
- [ ] Mock client for testing
- [ ] Test Create/Get/Update/Delete
- [ ] Test encryption/decryption
- [ ] Test cleanup functionality
- [ ] Test error cases

---

## Phase 4: API Key Ent Store (P1)

### Task 4.1: Define EntClientInterface
**Priority**: P1
**Estimate**: 1 hour
**Files**:
- `identity/apikey/store_ent.go`

**Implementation**:
```go
// EntClientInterface abstracts the Ent client for API keys.
type EntClientInterface interface {
    CreateAPIKey(ctx context.Context, key *APIKey, keyHash string) error
    GetAPIKeyByPrefix(ctx context.Context, prefix string) (*APIKey, string, error)
    GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*APIKey, error)
    ListAPIKeysByOwner(ctx context.Context, ownerID uuid.UUID) ([]*APIKey, error)
    ListAPIKeysByOrganization(ctx context.Context, orgID uuid.UUID) ([]*APIKey, error)
    UpdateAPIKey(ctx context.Context, key *APIKey) error
    DeleteAPIKey(ctx context.Context, id uuid.UUID) error
    UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID, ip string) error
}
```

**Acceptance Criteria**:
- [ ] Interface covers all Store operations
- [ ] Returns key hash for validation
- [ ] Clear godoc for each method

### Task 4.2: Implement EntStore
**Priority**: P1
**Estimate**: 3 hours
**Files**:
- `identity/apikey/store_ent.go`

**Implementation**:
```go
// EntStoreConfig configures the Ent-backed API key store.
type EntStoreConfig struct {
    Client EntClientInterface
}

// EntStore implements Store using Ent.
type EntStore struct {
    config EntStoreConfig
}

// NewEntStore creates a new Ent-backed API key store.
func NewEntStore(config EntStoreConfig) (*EntStore, error) {
    if config.Client == nil {
        return nil, errors.New("client is required")
    }
    return &EntStore{config: config}, nil
}

// Create stores a new API key.
func (s *EntStore) Create(ctx context.Context, key *APIKey, keyHash string) error {
    return s.config.Client.CreateAPIKey(ctx, key, keyHash)
}

// GetByPrefix retrieves an API key by its prefix.
func (s *EntStore) GetByPrefix(ctx context.Context, prefix string) (*APIKey, string, error) {
    return s.config.Client.GetAPIKeyByPrefix(ctx, prefix)
}

// ... other methods
```

**Acceptance Criteria**:
- [ ] All Store interface methods implemented
- [ ] Hash-based lookup for validation
- [ ] Last used tracking
- [ ] Revocation support

### Task 4.3: Write EntStore Tests
**Priority**: P1
**Estimate**: 2 hours
**Files**:
- `identity/apikey/store_ent_test.go`

**Acceptance Criteria**:
- [ ] Mock client for testing
- [ ] Test Create/Get/List/Delete
- [ ] Test GetByPrefix returns hash
- [ ] Test error cases

---

## Phase 5: Example Integration (P2)

### Task 5.1: Create Example App
**Priority**: P2
**Estimate**: 3 hours
**Files**:
- `_examples/ent-stores/main.go`
- `_examples/ent-stores/ent/schema/bff_session.go`
- `_examples/ent-stores/ent/schema/api_key.go`

**Acceptance Criteria**:
- [ ] Shows mixin usage
- [ ] Shows client wrapper implementation
- [ ] Shows store initialization
- [ ] Compiles and runs

---

## Phase 6: Documentation (P2)

### Task 6.1: BFF Ent Store Documentation
**Priority**: P2
**Estimate**: 2 hours
**Files**:
- `docs/bff/ent-store.md`

**Content**:
- How to add schema to app
- How to implement client wrapper
- How to wire up store
- Configuration options

### Task 6.2: API Key Store Documentation
**Priority**: P2
**Estimate**: 1 hour
**Files**:
- `docs/identity/api-key-store.md`

**Content**:
- How to add schema to app
- How to implement client wrapper
- How to wire up store

### Task 6.3: Update README
**Priority**: P2
**Estimate**: 1 hour
**Files**:
- `README.md`

**Content**:
- Add Ent stores to features list
- Link to documentation

---

## Summary

| Phase | Tasks | Estimate | Priority |
|-------|-------|----------|----------|
| Phase 1 | 2 | 4 hours | P0 |
| Phase 2 | 1 | 2 hours | P0 |
| Phase 3 | 3 | 8 hours | P0 |
| Phase 4 | 3 | 6 hours | P1 |
| Phase 5 | 1 | 3 hours | P2 |
| Phase 6 | 3 | 4 hours | P2 |
| **Total** | **13** | **~27 hours** | |

## Dependencies

```
Phase 1 (Mixins) ──┬──> Phase 3 (BFF Store)
                   │
Phase 2 (Encrypt) ─┘

Phase 1 (Mixins) ────> Phase 4 (API Key Store)

Phase 3, 4 ──────────> Phase 5 (Example)

Phase 3, 4, 5 ───────> Phase 6 (Docs)
```
