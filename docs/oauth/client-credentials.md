# Client Credentials Grant

The Client Credentials grant is used for server-to-server authentication where no user is involved.

## Use Cases

- Backend services accessing APIs
- Scheduled jobs and cron tasks
- CI/CD pipelines
- Microservice communication
- Data synchronization jobs

## Flow Overview

```
┌──────────┐                              ┌──────────┐
│  Client  │  POST /oauth/token           │  Token   │
│ (Server) │ ────────────────────────────▶│ Endpoint │
│          │  client_id + client_secret   │          │
│          │◀─────────────────────────────│          │
└──────────┘  Access Token                └──────────┘
```

## Setup

### Create Service App

```go
app, err := client.OAuthApp.Create().
    SetClientID("my-backend-service").
    SetName("Backend Service").
    SetAppType("service").
    SetPublic(false).
    SetAllowedScopes([]string{"api:read", "api:write"}).
    SetAllowedGrants([]string{"client_credentials"}).
    SetAccessTokenTTL(3600). // 1 hour
    SetOwnerID(systemUserID).
    Save(ctx)

// Generate secret
secret := generateSecret()
hash := hashSecret(secret)

_, err = client.OAuthAppSecret.Create().
    SetAppID(app.ID).
    SetSecretHash(hash).
    SetSecretPrefix(secret[:8]).
    Save(ctx)
```

## Token Request

### Using Basic Authentication

```bash
curl -X POST https://api.example.com/oauth/token \
  -u "my-backend-service:client_secret_here" \
  -d "grant_type=client_credentials" \
  -d "scope=api:read api:write"
```

### Using POST Body

```bash
curl -X POST https://api.example.com/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=my-backend-service" \
  -d "client_secret=client_secret_here" \
  -d "scope=api:read api:write"
```

## Token Response

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api:read api:write"
}
```

**Note**: No refresh token is issued. Request a new access token when needed.

## Go Client Example

```go
package main

import (
    "context"
    "net/http"
    "net/url"
    "encoding/json"

    "golang.org/x/oauth2/clientcredentials"
)

func main() {
    // Configure client credentials
    config := &clientcredentials.Config{
        ClientID:     "my-backend-service",
        ClientSecret: "client_secret_here",
        TokenURL:     "https://api.example.com/oauth/token",
        Scopes:       []string{"api:read", "api:write"},
    }

    // Get HTTP client with automatic token management
    client := config.Client(context.Background())

    // Make authenticated requests
    resp, err := client.Get("https://api.example.com/v1/data")
    // ...
}
```

## Python Client Example

```python
import requests
from requests.auth import HTTPBasicAuth

# Get token
response = requests.post(
    'https://api.example.com/oauth/token',
    auth=HTTPBasicAuth('my-backend-service', 'client_secret_here'),
    data={
        'grant_type': 'client_credentials',
        'scope': 'api:read api:write'
    }
)

token = response.json()['access_token']

# Use token
headers = {'Authorization': f'Bearer {token}'}
api_response = requests.get('https://api.example.com/v1/data', headers=headers)
```

## Token Caching

Cache tokens to avoid unnecessary token requests:

```go
type TokenCache struct {
    token     string
    expiresAt time.Time
    mu        sync.RWMutex
}

func (c *TokenCache) GetToken(ctx context.Context, config *clientcredentials.Config) (string, error) {
    c.mu.RLock()
    if c.token != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
        defer c.mu.RUnlock()
        return c.token, nil
    }
    c.mu.RUnlock()

    // Fetch new token
    c.mu.Lock()
    defer c.mu.Unlock()

    token, err := config.Token(ctx)
    if err != nil {
        return "", err
    }

    c.token = token.AccessToken
    c.expiresAt = token.Expiry
    return c.token, nil
}
```

## Security Considerations

### Secret Storage

Store client secrets securely:

```go
// Environment variable
secret := os.Getenv("OAUTH_CLIENT_SECRET")

// Secrets manager (AWS, GCP, Azure)
secret, err := secretsManager.GetSecret("my-service/oauth-secret")

// Vault
secret, err := vaultClient.Read("secret/data/my-service")
```

### Secret Rotation

Rotate secrets periodically:

1. Generate new secret
2. Add to app (both secrets valid)
3. Update all clients to use new secret
4. Revoke old secret after confirmation

### Scope Limitation

Grant minimum required scopes:

```go
// Bad: overly broad
SetAllowedScopes([]string{"admin:all"})

// Good: specific scopes
SetAllowedScopes([]string{"read:orders", "write:shipments"})
```

### IP Allowlisting

Consider restricting by IP for sensitive services:

```go
func validateClientIP(r *http.Request, allowedIPs []string) bool {
    clientIP := r.RemoteAddr
    for _, allowed := range allowedIPs {
        if clientIP == allowed {
            return true
        }
    }
    return false
}
```

## Difference from API Keys

| Feature | Client Credentials | API Keys |
|---------|-------------------|----------|
| Token lifetime | Short (hours) | Long (months) |
| Revocation | Per-token | Per-key |
| Rotation | Automatic via expiry | Manual |
| Standards | OAuth 2.0 | Custom |
| Scopes | Per-request | Per-key |
