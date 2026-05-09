# API Keys

API keys provide server-to-server authentication for programmatic access.

## Quick Start

SystemForge provides a complete API key management system:

```go
import (
    "github.com/grokify/systemforge/identity/apikey"
)

// Create service with your store implementation
service := apikey.NewService(apikey.ServiceConfig{
    Store:         myStore,               // Your store implementation
    Prefix:        "myapp",               // Key prefix (default: "cf")
    AllowedScopes: []string{"read:*", "write:*"},
})

// Generate a new key
result, err := service.Create(ctx, apikey.CreateKeyRequest{
    Name:     "CI/CD Pipeline",
    OwnerID:  userID,
    Scopes:   []string{"read:deployments", "write:deployments"},
})
// result.Key = "myapp_live_abc123..._xyz789..." (show once)
// result.APIKey = metadata (store in UI)

// Validate incoming key
apiKey, err := service.Validate(ctx, rawKey)
if err != nil {
    // Invalid, expired, or revoked
}

// Check scope
if !apiKey.HasScope("write:deployments") {
    // Forbidden
}
```

## Schema

The `cf_api_keys` table contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `name` | string | Key name/description |
| `key_prefix` | string | First 8 chars for identification |
| `key_hash` | string | SHA256 hash of the key |
| `user_id` | UUID | Owner user (optional) |
| `organization_id` | UUID | Owning organization (optional) |
| `scopes` | JSON | Allowed scopes |
| `expires_at` | time | Expiration (optional) |
| `last_used_at` | time | Last usage timestamp |
| `revoked` | bool | Revocation status |
| `created_at` | time | Creation timestamp |

## Key Format

API keys use the format: `cf_<base62_encoded_random_bytes>`

Example: `cf_7K3mN9pQrS2tUvW4xYz6`

## Creating API Keys

### Generate Key

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"

    "github.com/grokify/systemforge/identity"
)

// Generate a secure random key
keyBytes := make([]byte, 32)
rand.Read(keyBytes)
rawKey := "cf_" + identity.Base62Encode(keyBytes)

// Hash for storage
hash := sha256.Sum256([]byte(rawKey))
keyHash := hex.EncodeToString(hash[:])

// Store in database
apiKey, err := client.APIKey.Create().
    SetName("CI/CD Pipeline").
    SetKeyPrefix(rawKey[:11]). // "cf_" + first 8 chars
    SetKeyHash(keyHash).
    SetOrganizationID(orgID).
    SetScopes([]string{"read:deployments", "write:deployments"}).
    Save(ctx)

// Return the raw key to user (only time it's visible)
return rawKey
```

### With Expiration

```go
apiKey, err := client.APIKey.Create().
    SetName("Temporary Access").
    SetKeyPrefix(prefix).
    SetKeyHash(hash).
    SetUserID(userID).
    SetExpiresAt(time.Now().Add(30 * 24 * time.Hour)). // 30 days
    Save(ctx)
```

## Validating API Keys

```go
func ValidateAPIKey(ctx context.Context, client *ent.Client, rawKey string) (*ent.APIKey, error) {
    // Check format
    if !strings.HasPrefix(rawKey, "cf_") {
        return nil, ErrInvalidKeyFormat
    }

    // Hash the key
    hash := sha256.Sum256([]byte(rawKey))
    keyHash := hex.EncodeToString(hash[:])

    // Find the key
    apiKey, err := client.APIKey.Query().
        Where(
            apikey.KeyHashEQ(keyHash),
            apikey.RevokedEQ(false),
        ).
        WithUser().
        WithOrganization().
        Only(ctx)
    if err != nil {
        return nil, ErrInvalidKey
    }

    // Check expiration
    if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
        return nil, ErrKeyExpired
    }

    // Update last used
    client.APIKey.UpdateOneID(apiKey.ID).
        SetLastUsedAt(time.Now()).
        Save(ctx)

    return apiKey, nil
}
```

## API Key Middleware

```go
func APIKeyAuth(client *ent.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract from header
            authHeader := r.Header.Get("Authorization")
            if !strings.HasPrefix(authHeader, "Bearer cf_") {
                http.Error(w, "invalid authorization", http.StatusUnauthorized)
                return
            }

            rawKey := strings.TrimPrefix(authHeader, "Bearer ")

            // Validate
            apiKey, err := ValidateAPIKey(r.Context(), client, rawKey)
            if err != nil {
                http.Error(w, "invalid api key", http.StatusUnauthorized)
                return
            }

            // Add to context
            ctx := WithAPIKey(r.Context(), apiKey)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## Scope Checking

```go
func HasScope(apiKey *ent.APIKey, required string) bool {
    for _, scope := range apiKey.Scopes {
        if scope == required {
            return true
        }
        // Check wildcard
        if strings.HasSuffix(scope, ":*") {
            prefix := strings.TrimSuffix(scope, "*")
            if strings.HasPrefix(required, prefix) {
                return true
            }
        }
    }
    return false
}

// Usage
if !HasScope(apiKey, "write:deployments") {
    return ErrForbidden
}
```

## Managing API Keys

### List Keys

```go
keys, err := client.APIKey.Query().
    Where(
        apikey.OrganizationIDEQ(orgID),
        apikey.RevokedEQ(false),
    ).
    Order(ent.Desc(apikey.FieldCreatedAt)).
    All(ctx)
```

### Revoke Key

```go
_, err := client.APIKey.UpdateOneID(keyID).
    SetRevoked(true).
    Save(ctx)
```

### Delete Key

```go
err := client.APIKey.DeleteOneID(keyID).Exec(ctx)
```

## Security Best Practices

1. **Never log API keys** - Only log the prefix for debugging
2. **Set expirations** - Use short-lived keys when possible
3. **Scope narrowly** - Grant minimum required permissions
4. **Monitor usage** - Track `last_used_at` for anomaly detection
5. **Allow revocation** - Provide UI for users to revoke keys
6. **Rate limit** - Protect against brute force attacks

## Usage Example

```bash
# Using an API key
curl https://api.example.com/v1/deployments \
  -H "Authorization: Bearer cf_7K3mN9pQrS2tUvW4xYz6"
```

## Ent Store Implementation

SystemForge provides an Ent-backed store implementation for production use.

### Step 1: Add Schema Using Mixin

```go
// internal/ent/schema/api_key.go

package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "github.com/grokify/systemforge/identity/ent/mixin"
)

// APIKey holds the schema definition.
type APIKey struct {
    ent.Schema
}

// Mixin of the APIKey.
func (APIKey) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixin.APIKey{},
    }
}

// Edges of the APIKey.
func (APIKey) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("owner", User.Type).
            Ref("api_keys").
            Field("owner_id").
            Required().
            Unique(),
    }
}
```

### Step 2: Implement Client Interface

```go
// internal/auth/apikey_store.go

package auth

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/grokify/systemforge/identity/apikey"
    "myapp/internal/ent"
    entapikey "myapp/internal/ent/apikey"
)

// APIKeyClientWrapper implements apikey.EntClientInterface.
type APIKeyClientWrapper struct {
    client *ent.Client
}

func NewAPIKeyClientWrapper(client *ent.Client) *APIKeyClientWrapper {
    return &APIKeyClientWrapper{client: client}
}

func (w *APIKeyClientWrapper) CreateAPIKey(ctx context.Context, key *apikey.APIKey, keyHash string) error {
    create := w.client.APIKey.Create().
        SetID(key.ID).
        SetName(key.Name).
        SetPrefix(key.Prefix).
        SetKeyHash(keyHash).
        SetOwnerID(key.OwnerID).
        SetScopes(key.Scopes).
        SetEnvironment(entapikey.Environment(key.Environment)).
        SetCreatedAt(key.CreatedAt).
        SetUpdatedAt(key.UpdatedAt)

    if key.OrganizationID != nil {
        create = create.SetOrganizationID(*key.OrganizationID)
    }
    if key.Description != "" {
        create = create.SetDescription(key.Description)
    }
    if key.ExpiresAt != nil {
        create = create.SetExpiresAt(*key.ExpiresAt)
    }
    if key.Metadata != nil {
        create = create.SetMetadata(key.Metadata)
    }

    _, err := create.Save(ctx)
    return err
}

func (w *APIKeyClientWrapper) GetAPIKeyByPrefix(ctx context.Context, prefix string) (*apikey.APIKey, string, error) {
    k, err := w.client.APIKey.Query().
        Where(entapikey.PrefixEQ(prefix)).
        Only(ctx)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, "", apikey.ErrKeyNotFound
        }
        return nil, "", err
    }
    return toAPIKey(k), k.KeyHash, nil
}

func (w *APIKeyClientWrapper) GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
    k, err := w.client.APIKey.Get(ctx, id)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, apikey.ErrKeyNotFound
        }
        return nil, err
    }
    return toAPIKey(k), nil
}

func (w *APIKeyClientWrapper) ListAPIKeysByOwner(ctx context.Context, ownerID uuid.UUID) ([]*apikey.APIKey, error) {
    keys, err := w.client.APIKey.Query().
        Where(entapikey.OwnerIDEQ(ownerID)).
        Order(ent.Desc(entapikey.FieldCreatedAt)).
        All(ctx)
    if err != nil {
        return nil, err
    }

    result := make([]*apikey.APIKey, len(keys))
    for i, k := range keys {
        result[i] = toAPIKey(k)
    }
    return result, nil
}

func (w *APIKeyClientWrapper) ListAPIKeysByOrganization(ctx context.Context, orgID uuid.UUID) ([]*apikey.APIKey, error) {
    keys, err := w.client.APIKey.Query().
        Where(entapikey.OrganizationIDEQ(orgID)).
        Order(ent.Desc(entapikey.FieldCreatedAt)).
        All(ctx)
    if err != nil {
        return nil, err
    }

    result := make([]*apikey.APIKey, len(keys))
    for i, k := range keys {
        result[i] = toAPIKey(k)
    }
    return result, nil
}

func (w *APIKeyClientWrapper) UpdateAPIKey(ctx context.Context, key *apikey.APIKey) error {
    update := w.client.APIKey.UpdateOneID(key.ID).
        SetName(key.Name).
        SetScopes(key.Scopes).
        SetRevoked(key.Revoked).
        SetUpdatedAt(key.UpdatedAt)

    if key.Description != "" {
        update = update.SetDescription(key.Description)
    }
    if key.RevokedAt != nil {
        update = update.SetRevokedAt(*key.RevokedAt)
    }
    if key.RevokedReason != "" {
        update = update.SetRevokedReason(key.RevokedReason)
    }
    if key.Metadata != nil {
        update = update.SetMetadata(key.Metadata)
    }

    _, err := update.Save(ctx)
    return err
}

func (w *APIKeyClientWrapper) DeleteAPIKey(ctx context.Context, id uuid.UUID) error {
    return w.client.APIKey.DeleteOneID(id).Exec(ctx)
}

func (w *APIKeyClientWrapper) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID, ip string) error {
    now := time.Now()
    _, err := w.client.APIKey.UpdateOneID(id).
        SetLastUsedAt(now).
        SetLastUsedIP(ip).
        Save(ctx)
    return err
}

func toAPIKey(k *ent.APIKey) *apikey.APIKey {
    return &apikey.APIKey{
        ID:             k.ID,
        Name:           k.Name,
        Prefix:         k.Prefix,
        OwnerID:        k.OwnerID,
        OrganizationID: k.OrganizationID,
        Scopes:         k.Scopes,
        Description:    k.Description,
        Environment:    apikey.Environment(k.Environment),
        ExpiresAt:      k.ExpiresAt,
        LastUsedAt:     k.LastUsedAt,
        LastUsedIP:     k.LastUsedIP,
        Revoked:        k.Revoked,
        RevokedAt:      k.RevokedAt,
        RevokedReason:  k.RevokedReason,
        Metadata:       k.Metadata,
        CreatedAt:      k.CreatedAt,
        UpdatedAt:      k.UpdatedAt,
    }
}
```

### Step 3: Create the Store and Service

```go
// main.go or wire.go

import (
    "github.com/grokify/systemforge/identity/apikey"
    "myapp/internal/auth"
)

// Create the Ent store
store, err := apikey.NewEntStore(apikey.EntStoreConfig{
    Client: auth.NewAPIKeyClientWrapper(entClient),
})
if err != nil {
    log.Fatal(err)
}

// Create the service
service := apikey.NewService(apikey.ServiceConfig{
    Store:         store,
    Prefix:        "myapp",
    AllowedScopes: []string{
        "read:users",
        "write:users",
        "read:projects",
        "write:projects",
    },
    MaxKeysPerUser: 10,
})
```

### Mixin Fields

The `mixin.APIKey` provides these fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `name` | string | User-provided name |
| `prefix` | string | Visible prefix (e.g., `cf_live_xxxx`) |
| `key_hash` | string | SHA-256 hash (sensitive) |
| `owner_id` | UUID | Owner user |
| `organization_id` | UUID | Organization scope (optional) |
| `scopes` | []string | Granted permissions |
| `description` | string | User-provided note (optional) |
| `environment` | enum | `live` or `test` |
| `expires_at` | time | Expiration (optional) |
| `last_used_at` | time | Last usage (optional) |
| `last_used_ip` | string | Last IP (optional) |
| `revoked` | bool | Revocation status |
| `revoked_at` | time | Revocation time (optional) |
| `revoked_reason` | string | Revocation reason (optional) |
| `metadata` | JSON | Additional data (optional) |
| `created_at` | time | Creation timestamp |
| `updated_at` | time | Update timestamp |

### Indexes

The mixin creates indexes for common queries:

```go
index.Fields("owner_id")                // List by owner
index.Fields("organization_id")         // List by organization
index.Fields("prefix")                  // Lookup by prefix
index.Fields("key_hash").Unique()       // Validate by hash
index.Fields("environment")             // Filter by environment
```

## Service Methods

The `apikey.Service` provides these methods:

```go
// Create a new API key (returns full key once)
result, err := service.Create(ctx, CreateKeyRequest{
    Name:           "My Key",
    OwnerID:        userID,
    OrganizationID: &orgID,
    Scopes:         []string{"read:*"},
    Description:    "For CI/CD",
    Environment:    apikey.EnvLive,
    ExpiresIn:      ptr(90 * 24 * time.Hour),
    Metadata:       map[string]string{"team": "platform"},
})

// Validate a key
apiKey, err := service.Validate(ctx, rawKey)

// Validate with scope check
apiKey, err := service.ValidateWithScope(ctx, rawKey, "write:projects")

// Get key by ID
apiKey, err := service.Get(ctx, keyID)

// List keys for a user
keys, err := service.List(ctx, ownerID)

// List keys for an organization
keys, err := service.ListByOrganization(ctx, orgID)

// Revoke a key
err := service.Revoke(ctx, keyID, "Compromised")

// Delete a key
err := service.Delete(ctx, keyID)

// Record usage (call after validation)
err := service.RecordUsage(ctx, keyID, clientIP)
```

## Middleware Integration

```go
import (
    "github.com/grokify/systemforge/session/middleware"
)

// Use the built-in middleware
apiAuth := middleware.APIKeyAuth(middleware.APIKeyConfig{
    Service:      service,
    RecordUsage:  true,
    ContextKey:   "api_key",
})

// Apply to routes
r.Route("/api/v1", func(r chi.Router) {
    r.Use(apiAuth)
    r.Get("/projects", listProjects)
})

// Access in handler
func listProjects(w http.ResponseWriter, r *http.Request) {
    principal := middleware.GetPrincipal(r.Context())
    // principal.ID = owner_id
    // principal.Scopes = granted scopes
}
```
