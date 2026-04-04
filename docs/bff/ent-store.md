# Ent-Backed BFF Session Store

CoreForge provides an Ent-backed session store for production BFF deployments. This stores sessions in your database with encrypted tokens at rest.

## Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CoreForge (Library)                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│   session/bff/store.go          session/bff/store_ent.go                    │
│                                                                              │
│   type Store interface {         type EntStore struct {                      │
│     Create(...)                    config    EntStoreConfig                  │
│     Get(...)                       encryptor *Encryptor                      │
│     Update(...)                  }                                           │
│     Delete(...)                                                              │
│   }                              func NewEntStore(...) (*EntStore, error)    │
└────────────────────────────────────────────────────────┬────────────────────┘
                                                         │
                                                         │ implements
                                                         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        App-Level Implementation                              │
├─────────────────────────────────────────────────────────────────────────────┤
│   internal/ent/schema/bff_session.go    internal/auth/bff_store.go          │
│                                                                              │
│   Uses mixin:                           Implements EntClientInterface:       │
│   coreforge/identity/ent/mixin          wraps generated Ent client           │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **Encrypted tokens** - AES-256-GCM encryption for tokens at rest
- **Automatic cleanup** - Background goroutine removes expired sessions
- **Database agnostic** - Works with PostgreSQL, MySQL, SQLite
- **Reusable schema** - Ent mixin provides consistent field definitions

## Setup

### Step 1: Add Schema to Your App

Create an Ent schema using the CoreForge mixin:

```go
// internal/ent/schema/bff_session.go

package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "github.com/grokify/coreforge/identity/ent/mixin"
)

// BFFSession holds the schema definition.
type BFFSession struct {
    ent.Schema
}

// Mixin of the BFFSession.
func (BFFSession) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixin.BFFSession{},
    }
}

// Edges of the BFFSession (optional, for relations).
func (BFFSession) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).
            Ref("bff_sessions").
            Field("user_id").
            Required().
            Unique(),
    }
}
```

Then run `go generate ./ent` to generate the Ent code.

### Step 2: Implement Client Interface

Create a wrapper that implements `bff.EntClientInterface`:

```go
// internal/auth/bff_store.go

package auth

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/grokify/coreforge/session/bff"
    "myapp/internal/ent"
    "myapp/internal/ent/bffsession"
)

// EntClientWrapper implements bff.EntClientInterface.
type EntClientWrapper struct {
    client *ent.Client
}

// NewEntClientWrapper creates a new wrapper.
func NewEntClientWrapper(client *ent.Client) *EntClientWrapper {
    return &EntClientWrapper{client: client}
}

func (w *EntClientWrapper) CreateBFFSession(ctx context.Context, session *bff.Session) error {
    create := w.client.BFFSession.Create().
        SetID(session.ID).
        SetUserID(session.UserID).
        SetAccessTokenEncrypted(session.EncryptedAccessToken()).
        SetRefreshTokenEncrypted(session.EncryptedRefreshToken()).
        SetAccessTokenExpiresAt(session.AccessTokenExpiresAt).
        SetRefreshTokenExpiresAt(session.RefreshTokenExpiresAt).
        SetExpiresAt(session.ExpiresAt).
        SetCreatedAt(session.CreatedAt).
        SetUpdatedAt(session.UpdatedAt).
        SetLastAccessedAt(session.LastAccessedAt)

    if session.OrganizationID != nil {
        create = create.SetOrganizationID(*session.OrganizationID)
    }
    if session.IPAddress != "" {
        create = create.SetIPAddress(session.IPAddress)
    }
    if session.UserAgent != "" {
        create = create.SetUserAgent(session.UserAgent)
    }
    if len(session.EncryptedDPoPKeyPair()) > 0 {
        create = create.SetDpopKeyPairEncrypted(session.EncryptedDPoPKeyPair())
    }
    if session.DPoPThumbprint != "" {
        create = create.SetDpopThumbprint(session.DPoPThumbprint)
    }
    if session.Metadata != nil {
        create = create.SetMetadata(session.Metadata)
    }

    _, err := create.Save(ctx)
    return err
}

func (w *EntClientWrapper) GetBFFSession(ctx context.Context, id string) (*bff.Session, error) {
    s, err := w.client.BFFSession.Query().
        Where(bffsession.IDEQ(id)).
        Only(ctx)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, bff.ErrSessionNotFound
        }
        return nil, err
    }

    session := &bff.Session{
        ID:                    s.ID,
        UserID:                s.UserID,
        OrganizationID:        s.OrganizationID,
        AccessTokenExpiresAt:  s.AccessTokenExpiresAt,
        RefreshTokenExpiresAt: s.RefreshTokenExpiresAt,
        DPoPThumbprint:        s.DpopThumbprint,
        IPAddress:             s.IPAddress,
        UserAgent:             s.UserAgent,
        Metadata:              s.Metadata,
        CreatedAt:             s.CreatedAt,
        UpdatedAt:             s.UpdatedAt,
        LastAccessedAt:        s.LastAccessedAt,
        ExpiresAt:             s.ExpiresAt,
    }

    // Set encrypted fields for decryption by EntStore
    session.SetEncryptedTokens(
        s.AccessTokenEncrypted,
        s.RefreshTokenEncrypted,
        s.DpopKeyPairEncrypted,
    )

    return session, nil
}

func (w *EntClientWrapper) UpdateBFFSession(ctx context.Context, session *bff.Session) error {
    update := w.client.BFFSession.UpdateOneID(session.ID).
        SetAccessTokenEncrypted(session.EncryptedAccessToken()).
        SetRefreshTokenEncrypted(session.EncryptedRefreshToken()).
        SetAccessTokenExpiresAt(session.AccessTokenExpiresAt).
        SetRefreshTokenExpiresAt(session.RefreshTokenExpiresAt).
        SetExpiresAt(session.ExpiresAt).
        SetUpdatedAt(session.UpdatedAt).
        SetLastAccessedAt(session.LastAccessedAt)

    if len(session.EncryptedDPoPKeyPair()) > 0 {
        update = update.SetDpopKeyPairEncrypted(session.EncryptedDPoPKeyPair())
    }
    if session.DPoPThumbprint != "" {
        update = update.SetDpopThumbprint(session.DPoPThumbprint)
    }
    if session.Metadata != nil {
        update = update.SetMetadata(session.Metadata)
    }

    _, err := update.Save(ctx)
    return err
}

func (w *EntClientWrapper) DeleteBFFSession(ctx context.Context, id string) error {
    return w.client.BFFSession.DeleteOneID(id).Exec(ctx)
}

func (w *EntClientWrapper) DeleteBFFSessionsByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
    return w.client.BFFSession.Delete().
        Where(bffsession.UserIDEQ(userID)).
        Exec(ctx)
}

func (w *EntClientWrapper) TouchBFFSession(ctx context.Context, id string) error {
    _, err := w.client.BFFSession.UpdateOneID(id).
        SetLastAccessedAt(time.Now()).
        Save(ctx)
    return err
}

func (w *EntClientWrapper) CleanupExpiredBFFSessions(ctx context.Context, limit int) (int, error) {
    return w.client.BFFSession.Delete().
        Where(bffsession.ExpiresAtLT(time.Now())).
        Exec(ctx)
}
```

### Step 3: Wire Up the Store

```go
// main.go or wire.go

import (
    "github.com/grokify/coreforge/session/bff"
    "myapp/internal/auth"
)

// Create the Ent store
store, err := bff.NewEntStore(bff.EntStoreConfig{
    Client:           auth.NewEntClientWrapper(entClient),
    EncryptionKey:    []byte(cfg.SessionEncryptionKey), // 32 bytes
    CleanupInterval:  5 * time.Minute,
    CleanupBatchSize: 100,
})
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Use with BFF handler
bffHandler, err := bff.NewHandler(bff.HandlerConfig{
    Store:          store,
    AllowedOrigins: cfg.AllowedOrigins,
    // ...
})
```

## Configuration

### EntStoreConfig

```go
type EntStoreConfig struct {
    // Required: Ent client wrapper
    Client EntClientInterface

    // Required: 32-byte encryption key for AES-256
    EncryptionKey []byte

    // Optional: cleanup interval (0 = disabled)
    CleanupInterval time.Duration

    // Optional: max sessions deleted per cleanup (default: 100)
    CleanupBatchSize int
}
```

### Encryption Key

The encryption key must be exactly 32 bytes for AES-256-GCM:

```go
// From environment variable (base64 encoded)
key, _ := base64.StdEncoding.DecodeString(os.Getenv("SESSION_ENCRYPTION_KEY"))

// Or generate one for development
key := make([]byte, 32)
rand.Read(key)
fmt.Println("Key:", base64.StdEncoding.EncodeToString(key))
```

## Mixin Fields

The `mixin.BFFSession` provides these fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Session ID (unique, immutable) |
| `user_id` | UUID | Owner user |
| `organization_id` | UUID | Organization context (optional) |
| `access_token_encrypted` | []byte | Encrypted access token |
| `refresh_token_encrypted` | []byte | Encrypted refresh token |
| `access_token_expires_at` | time | Access token expiry |
| `refresh_token_expires_at` | time | Refresh token expiry |
| `dpop_key_pair_encrypted` | []byte | Encrypted DPoP key pair (optional) |
| `dpop_thumbprint` | string | DPoP JWK thumbprint (optional) |
| `ip_address` | string | Client IP (optional) |
| `user_agent` | string | Client User-Agent (optional) |
| `metadata` | JSON | Additional metadata (optional) |
| `last_accessed_at` | time | Last access timestamp |
| `expires_at` | time | Session expiry |
| `created_at` | time | Creation timestamp |
| `updated_at` | time | Update timestamp |

## Indexes

The mixin creates indexes for common queries:

```go
index.Fields("user_id")        // Find sessions by user
index.Fields("expires_at")     // Cleanup expired sessions
index.Fields("organization_id") // Filter by organization
```

## Encryption

Tokens are encrypted using AES-256-GCM before storage:

```go
// Encryption happens automatically in Create/Update
store.Create(ctx, session)
// session.AccessToken and RefreshToken are encrypted

// Decryption happens automatically in Get
session, _ := store.Get(ctx, id)
// session.AccessToken and RefreshToken are decrypted
```

The `Encryptor` is also available for direct use:

```go
encryptor, _ := bff.NewEncryptor(key)

// Encrypt
ciphertext, _ := encryptor.Encrypt([]byte("secret"))
ciphertext, _ := encryptor.EncryptString("secret")

// Decrypt
plaintext, _ := encryptor.Decrypt(ciphertext)
str, _ := encryptor.DecryptString(ciphertext)
```

## Automatic Cleanup

When `CleanupInterval` is set, expired sessions are automatically deleted:

```go
store, _ := bff.NewEntStore(bff.EntStoreConfig{
    // ...
    CleanupInterval:  5 * time.Minute,
    CleanupBatchSize: 100, // Delete at most 100 per run
})

// Cleanup runs in background goroutine
// Stop it on shutdown:
store.Close()
```

Manual cleanup is also available:

```go
deleted, err := store.Cleanup(ctx)
fmt.Printf("Deleted %d expired sessions\n", deleted)
```

## Database Migration

Example SQL migration (PostgreSQL):

```sql
CREATE TABLE bff_sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id UUID NOT NULL,
    organization_id UUID,
    access_token_encrypted BYTEA NOT NULL,
    refresh_token_encrypted BYTEA NOT NULL,
    access_token_expires_at TIMESTAMP NOT NULL,
    refresh_token_expires_at TIMESTAMP NOT NULL,
    dpop_key_pair_encrypted BYTEA,
    dpop_thumbprint VARCHAR(64),
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    metadata JSONB,
    last_accessed_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_bff_sessions_user_id ON bff_sessions(user_id);
CREATE INDEX idx_bff_sessions_expires_at ON bff_sessions(expires_at);
CREATE INDEX idx_bff_sessions_organization_id ON bff_sessions(organization_id);
```

## Security Considerations

1. **Protect the encryption key** - Store in secrets manager, not in code
2. **Rotate keys periodically** - Implement key rotation strategy
3. **Use HTTPS** - Encrypted storage doesn't help if transport is insecure
4. **Monitor session counts** - Alert on unusual session creation rates
5. **Audit access** - Log session creation/deletion for security review
