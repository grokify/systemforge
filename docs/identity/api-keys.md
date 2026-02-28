# API Keys

API keys provide server-to-server authentication for programmatic access.

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

    "github.com/grokify/coreforge/identity"
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
