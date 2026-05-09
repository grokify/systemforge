# Ent Schema Reference

Complete reference for all SystemForge Ent schemas.

## Identity Schemas

### User

**Table**: `cf_users`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `email` | string | unique, not empty | User email |
| `name` | string | not empty | Display name |
| `avatar_url` | string | optional, nillable | Profile picture |
| `password_hash` | string | optional, sensitive | Argon2id hash |
| `is_platform_admin` | bool | default: false | Cross-org admin |
| `active` | bool | default: true | Account status |
| `last_login_at` | time | optional, nillable | Last login |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**Edges**:

- `memberships` → Membership (O2M)
- `oauth_accounts` → OAuthAccount (O2M)
- `refresh_tokens` → RefreshToken (O2M)
- `api_keys` → APIKey (O2M)
- `oauth_apps` → OAuthApp (O2M)
- `oauth_tokens` → OAuthToken (O2M)
- `oauth_auth_codes` → OAuthAuthCode (O2M)
- `oauth_consents` → OAuthConsent (O2M)
- `created_service_accounts` → ServiceAccount (O2M)

---

### Organization

**Table**: `cf_organizations`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `name` | string | not empty | Display name |
| `slug` | string | unique, not empty | URL identifier |
| `logo_url` | string | optional, nillable | Logo URL |
| `settings` | JSON | optional | Custom config |
| `plan` | enum | default: free | Subscription tier |
| `active` | bool | default: true | Org status |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**Plan Values**: `free`, `starter`, `pro`, `enterprise`

**Edges**:

- `memberships` → Membership (O2M)
- `api_keys` → APIKey (O2M)
- `oauth_apps` → OAuthApp (O2M)
- `service_accounts` → ServiceAccount (O2M)

---

### Membership

**Table**: `cf_memberships`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `user_id` | UUID | FK, required | User reference |
| `organization_id` | UUID | FK, required | Org reference |
| `role` | string | not empty | Role name |
| `permissions` | JSON | optional | Fine-grained perms |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**Unique Constraint**: `(user_id, organization_id)`

**Edges**:

- `user` → User (M2O)
- `organization` → Organization (M2O)

---

### OAuthAccount

**Table**: `cf_oauth_accounts`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `user_id` | UUID | FK, required | User reference |
| `provider` | string | not empty | Provider name |
| `provider_user_id` | string | not empty | External ID |
| `access_token` | string | sensitive | Encrypted token |
| `refresh_token` | string | optional, sensitive | Encrypted token |
| `expires_at` | time | optional, nillable | Token expiry |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**Unique Constraint**: `(provider, provider_user_id)`

---

### APIKey

**Table**: `cf_api_keys`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `name` | string | not empty | Key name |
| `key_prefix` | string | max: 12 | First chars |
| `key_hash` | string | unique, sensitive | SHA256 hash |
| `user_id` | UUID | FK, optional | Owner user |
| `organization_id` | UUID | FK, optional | Owner org |
| `scopes` | JSON | default: [] | Allowed scopes |
| `expires_at` | time | optional, nillable | Expiration |
| `last_used_at` | time | optional, nillable | Last usage |
| `revoked` | bool | default: false | Revocation status |
| `created_at` | time | immutable | Creation time |

---

## OAuth Schemas

### OAuthApp

**Table**: `cf_oauth_apps`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `client_id` | string | unique, immutable | Public ID |
| `name` | string | not empty, max: 255 | App name |
| `description` | string | optional, max: 1000 | Description |
| `logo_url` | string | optional | App logo |
| `app_type` | enum | default: web | Client type |
| `owner_id` | UUID | FK, required | Creating user |
| `organization_id` | UUID | FK, optional | Owning org |
| `redirect_uris` | JSON | default: [] | Allowed URIs |
| `allowed_scopes` | JSON | default: [] | Allowed scopes |
| `allowed_grants` | JSON | default: [auth_code, refresh] | Grant types |
| `allowed_response_types` | JSON | default: [code] | Response types |
| `access_token_ttl` | int | default: 900 | Token TTL (sec) |
| `refresh_token_ttl` | int | default: 604800 | Refresh TTL (sec) |
| `refresh_token_rotation` | bool | default: true | Enable rotation |
| `first_party` | bool | default: false | Skip consent |
| `public` | bool | default: false | No secret |
| `active` | bool | default: true | App status |
| `revoked_at` | time | optional, nillable | Revocation time |
| `metadata` | JSON | optional | Custom data |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**App Types**: `web`, `spa`, `native`, `service`, `machine`

---

### OAuthAppSecret

**Table**: `cf_oauth_app_secrets`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `app_id` | UUID | FK, required | Parent app |
| `secret_hash` | string | sensitive, not empty | Argon2id hash |
| `secret_prefix` | string | max: 12 | First chars |
| `expires_at` | time | optional, nillable | Expiration |
| `last_used_at` | time | optional, nillable | Last usage |
| `revoked` | bool | default: false | Revocation status |
| `revoked_at` | time | optional, nillable | Revocation time |
| `created_at` | time | immutable | Creation time |

---

### OAuthToken

**Table**: `cf_oauth_tokens`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `app_id` | UUID | FK, required | Issuing app |
| `user_id` | UUID | FK, optional | Token owner |
| `service_account_id` | UUID | FK, optional | SA owner |
| `access_token_signature` | string | unique | Token hash |
| `refresh_token_signature` | string | unique, optional | Refresh hash |
| `family_id` | UUID | default: new | Rotation family |
| `scopes` | JSON | default: [] | Granted scopes |
| `audience` | JSON | default: [] | Token audience |
| `session_id` | string | optional | BFF session |
| `request_data` | text | optional | Fosite data |
| `access_expires_at` | time | required | Access expiry |
| `refresh_expires_at` | time | optional, nillable | Refresh expiry |
| `revoked` | bool | default: false | Revocation status |
| `revoked_at` | time | optional, nillable | Revocation time |
| `revoked_reason` | string | optional | Reason |
| `client_ip` | string | optional | Request IP |
| `user_agent` | string | optional | User agent |
| `last_used_at` | time | optional, nillable | Last usage |
| `created_at` | time | immutable | Creation time |

---

### OAuthAuthCode

**Table**: `cf_oauth_auth_codes`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `code_signature` | string | unique | Code hash |
| `app_id` | UUID | FK, required | Requesting app |
| `user_id` | UUID | FK, required | Authorizing user |
| `code_challenge` | string | optional | PKCE challenge |
| `code_challenge_method` | string | optional, default: S256 | PKCE method |
| `redirect_uri` | string | required | Redirect URI |
| `scopes` | JSON | default: [] | Requested scopes |
| `state` | string | optional | State param |
| `nonce` | string | optional | OIDC nonce |
| `request_data` | text | optional | Fosite data |
| `expires_at` | time | required | Code expiry |
| `used` | bool | default: false | Exchange status |
| `used_at` | time | optional, nillable | Exchange time |
| `client_ip` | string | optional | Request IP |
| `user_agent` | string | optional | User agent |
| `created_at` | time | immutable | Creation time |

---

### OAuthConsent

**Table**: `cf_oauth_consents`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `user_id` | UUID | FK, required | Consenting user |
| `app_id` | UUID | FK, required | Consented app |
| `scopes` | JSON | default: [] | Consented scopes |
| `granted` | bool | default: true | Active status |
| `granted_at` | time | required | Consent time |
| `last_used_at` | time | optional, nillable | Last use |
| `revoked` | bool | default: false | Revocation status |
| `revoked_at` | time | optional, nillable | Revocation time |
| `revoked_reason` | string | optional | Reason |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

**Unique Constraint**: `(user_id, app_id)`

---

### ServiceAccount

**Table**: `cf_service_accounts`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `name` | string | not empty, max: 255 | Display name |
| `description` | string | optional, max: 1000 | Description |
| `email` | string | unique, not empty | Unique identifier |
| `organization_id` | UUID | FK, required | Owning org |
| `created_by` | UUID | FK, required | Creating user |
| `allowed_scopes` | JSON | default: [] | Allowed scopes |
| `active` | bool | default: true | Account status |
| `last_used_at` | time | optional, nillable | Last usage |
| `created_at` | time | immutable | Creation time |
| `updated_at` | time | auto-update | Modification time |

---

### ServiceAccountKeyPair

**Table**: `cf_service_account_key_pairs`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | UUID | PK, immutable | Primary key |
| `service_account_id` | UUID | FK, required | Parent SA |
| `key_id` | string | not empty | JWK kid |
| `key_type` | enum | default: rsa | Key type |
| `algorithm` | enum | default: RS256 | JWT algorithm |
| `public_key_pem` | text | not empty | PEM public key |
| `expires_at` | time | optional, nillable | Key expiry |
| `active` | bool | default: true | Key status |
| `last_used_at` | time | optional, nillable | Last usage |
| `revoked` | bool | default: false | Revocation status |
| `revoked_at` | time | optional, nillable | Revocation time |
| `created_at` | time | immutable | Creation time |

**Key Types**: `rsa`, `ec`

**Algorithms**: `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`

**Unique Constraint**: `(service_account_id, key_id)`
