# Users

Users represent individual accounts in the system.

## Schema

The `cf_users` table contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `email` | string | Unique email address |
| `name` | string | Display name |
| `avatar_url` | string | Profile picture URL |
| `password_hash` | string | Argon2id hashed password |
| `is_platform_admin` | bool | Cross-org admin access |
| `active` | bool | Account status |
| `last_login_at` | time | Last login timestamp |
| `created_at` | time | Account creation time |
| `updated_at` | time | Last modification time |

## Creating Users

### With Password

```go
import "github.com/grokify/systemforge/identity"

// Hash the password
hash, err := identity.HashPassword("secure-password")
if err != nil {
    return err
}

// Create the user
user, err := client.User.Create().
    SetEmail("user@example.com").
    SetName("John Doe").
    SetPasswordHash(hash).
    Save(ctx)
```

### OAuth-Only User

Users can be created without passwords if they authenticate via OAuth:

```go
user, err := client.User.Create().
    SetEmail("user@example.com").
    SetName("John Doe").
    // No password_hash - OAuth only
    Save(ctx)

// Link OAuth account
_, err = client.OAuthAccount.Create().
    SetUserID(user.ID).
    SetProvider("github").
    SetProviderUserID("12345").
    Save(ctx)
```

## Querying Users

### By Email

```go
user, err := client.User.Query().
    Where(user.EmailEQ("user@example.com")).
    Only(ctx)
```

### By Organization

```go
users, err := client.User.Query().
    Where(user.HasMembershipsWith(
        membership.OrganizationIDEQ(orgID),
    )).
    All(ctx)
```

### With Memberships

```go
user, err := client.User.Query().
    Where(user.IDEQ(userID)).
    WithMemberships(func(q *ent.MembershipQuery) {
        q.WithOrganization()
    }).
    Only(ctx)

for _, m := range user.Edges.Memberships {
    fmt.Printf("Org: %s, Role: %s\n", m.Edges.Organization.Name, m.Role)
}
```

## Updating Users

```go
user, err := client.User.UpdateOneID(userID).
    SetName("Jane Doe").
    SetAvatarURL("https://example.com/avatar.jpg").
    Save(ctx)
```

## Password Management

### Verify Password

```go
import "github.com/grokify/systemforge/identity"

user, _ := client.User.Query().
    Where(user.EmailEQ(email)).
    Only(ctx)

if user.PasswordHash == "" {
    // User has no password (OAuth only)
    return ErrNoPassword
}

if !identity.VerifyPassword(password, user.PasswordHash) {
    return ErrInvalidPassword
}
```

### Change Password

```go
newHash, _ := identity.HashPassword(newPassword)

_, err := client.User.UpdateOneID(userID).
    SetPasswordHash(newHash).
    Save(ctx)
```

## Deactivating Users

Soft-delete by deactivating:

```go
_, err := client.User.UpdateOneID(userID).
    SetActive(false).
    Save(ctx)
```

## Platform Admins

Platform admins have access across all organizations:

```go
// Make user a platform admin
_, err := client.User.UpdateOneID(userID).
    SetIsPlatformAdmin(true).
    Save(ctx)

// Check if user is platform admin
if user.IsPlatformAdmin {
    // Grant full access
}
```

## User Edges

Users have relationships to:

| Edge | Target | Description |
|------|--------|-------------|
| `memberships` | Membership | Organization memberships |
| `oauth_accounts` | OAuthAccount | Linked OAuth providers |
| `refresh_tokens` | RefreshToken | Active sessions |
| `api_keys` | APIKey | User's API keys |
| `oauth_apps` | OAuthApp | Apps created by user |
| `oauth_tokens` | OAuthToken | Tokens issued to user |
| `oauth_consents` | OAuthConsent | Consent grants |
