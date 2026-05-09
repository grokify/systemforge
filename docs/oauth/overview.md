# OAuth 2.0 Overview

SystemForge includes a full OAuth 2.0 server implementation using [Fosite](https://github.com/ory/fosite).

## Supported Grant Types

| Grant Type | Use Case | PKCE Required |
|------------|----------|---------------|
| Authorization Code | Web apps, SPAs, mobile apps | Yes (public clients) |
| Client Credentials | Server-to-server | No |
| Refresh Token | Token renewal | N/A |
| JWT Bearer (RFC 7523) | Service accounts | No |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      OAuth Provider                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                     Fosite Core                       │   │
│  └──────────────────────────────────────────────────────┘   │
│       │              │              │              │         │
│  ┌────┴────┐   ┌────┴────┐   ┌────┴────┐   ┌────┴────┐    │
│  │ AuthZ   │   │ Token   │   │ Intro-  │   │ Revoke  │    │
│  │ Handler │   │ Handler │   │ spect   │   │ Handler │    │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘    │
│                        │                                     │
│  ┌─────────────────────┴─────────────────────────────────┐  │
│  │                   Ent Storage Adapter                  │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      PostgreSQL                              │
│  cf_oauth_apps │ cf_oauth_tokens │ cf_oauth_auth_codes      │
└─────────────────────────────────────────────────────────────┘
```

## Database Schema

| Table | Description |
|-------|-------------|
| `cf_oauth_apps` | OAuth client applications |
| `cf_oauth_app_secrets` | Client secrets (hashed) |
| `cf_oauth_tokens` | Access and refresh tokens |
| `cf_oauth_auth_codes` | Authorization codes |
| `cf_oauth_consents` | User consent records |
| `cf_service_accounts` | Service accounts |
| `cf_service_account_key_pairs` | SA key pairs |

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/oauth/authorize` | GET/POST | Authorization endpoint |
| `/oauth/token` | POST | Token endpoint |
| `/oauth/introspect` | POST | Token introspection |
| `/oauth/revoke` | POST | Token revocation |
| `/.well-known/openid-configuration` | GET | OIDC discovery |
| `/.well-known/jwks.json` | GET | JSON Web Key Set |

## Quick Setup

```go
import (
    "github.com/grokify/systemforge/identity/ent"
    "github.com/grokify/systemforge/identity/oauth"
)

// Create provider
cfg := oauth.DefaultConfig("https://api.example.com", []byte("secret"))
provider, _ := oauth.NewProvider(entClient, cfg)

// Create API (all endpoints auto-registered)
api, _ := oauth.NewAPI(provider)

// Mount router (includes all OAuth and discovery endpoints)
http.Handle("/", api.Router())
```

## Security Features

### PKCE Enforcement

PKCE is required for public clients (SPAs, mobile apps):

```go
fositeConfig := &fosite.Config{
    EnforcePKCE:                 true,
    EnforcePKCEForPublicClients: true,
}
```

### Token Signatures

Tokens are stored as SHA256 signatures, not raw values:

```go
signature := sha256.Sum256([]byte(token))
// Only the signature is stored in the database
```

### Refresh Token Rotation

Each refresh token use generates a new token:

```go
app, _ := client.OAuthApp.Create().
    SetRefreshTokenRotation(true).
    // ...
```

### Secret Hashing

Client secrets use Argon2id:

```go
hash := argon2.IDKey(secret, salt, 3, 64*1024, 4, 32)
```

## Client Types

| Type | Secret | PKCE | Use Case |
|------|--------|------|----------|
| `web` | Required | Optional | Traditional web apps |
| `spa` | None | Required | Single-page apps |
| `native` | None | Required | Mobile/desktop apps |
| `service` | Required | No | Server-to-server |
| `machine` | Required | No | Automated systems |

## Next Steps

- [OAuth Apps](apps.md) - Creating and managing OAuth apps
- [Authorization Code](authorization-code.md) - User authorization flow
- [Client Credentials](client-credentials.md) - Server-to-server auth
- [Service Accounts](service-accounts.md) - JWT Bearer authentication
