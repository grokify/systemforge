# Integrating with Existing Apps

This guide covers integrating SystemForge into existing Go applications.

## Integration Patterns

### Pattern 1: Side-by-Side Tables

Keep existing tables, add SystemForge tables alongside:

```
existing tables          SystemForge tables
─────────────────       ─────────────────
users            ←──→   cf_users
organizations    ←──→   cf_organizations
user_orgs        ←──→   cf_memberships
```

### Pattern 2: Mixin Composition

Compose SystemForge mixins into your existing Ent schemas:

```go
// your-app/ent/schema/user.go
import cfmixin "github.com/grokify/systemforge/identity/ent/mixin"

type User struct {
    ent.Schema
}

func (User) Mixin() []ent.Mixin {
    return []ent.Mixin{
        cfmixin.UserBase{}, // SystemForge fields
    }
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        // Your app-specific fields
        field.String("username").Unique(),
        field.String("bio").Optional(),
    }
}
```

### Pattern 3: Extension Tables

Link your tables to SystemForge via foreign keys:

```go
type UserProfile struct {
    ent.Schema
}

func (UserProfile) Fields() []ent.Field {
    return []ent.Field{
        field.UUID("cf_user_id", uuid.UUID{}).Unique(),
        field.String("username").Unique(),
        field.String("bio").Optional(),
    }
}
```

## Step-by-Step Integration

### Step 1: Add Dependency

```bash
go get github.com/grokify/systemforge
```

### Step 2: Create SystemForge Tables

Run migrations to create `cf_*` tables:

```go
import "github.com/grokify/systemforge/identity/ent"

func migrate(ctx context.Context) error {
    cfClient, err := ent.Open("postgres", dsn)
    if err != nil {
        return err
    }

    return cfClient.Schema.Create(ctx)
}
```

### Step 3: Sync Existing Data

Create a migration to copy existing data:

```go
func syncUsers(ctx context.Context, appDB, cfDB *sql.DB) error {
    rows, err := appDB.QueryContext(ctx, `
        SELECT id, email, name, password_hash, created_at
        FROM users
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var u struct {
            ID           uuid.UUID
            Email        string
            Name         string
            PasswordHash string
            CreatedAt    time.Time
        }
        rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)

        _, err = cfDB.ExecContext(ctx, `
            INSERT INTO cf_users (id, email, name, password_hash, created_at, updated_at, active)
            VALUES ($1, $2, $3, $4, $5, $5, true)
            ON CONFLICT (id) DO UPDATE SET
                email = EXCLUDED.email,
                name = EXCLUDED.name,
                password_hash = EXCLUDED.password_hash
        `, u.ID, u.Email, u.Name, u.PasswordHash, u.CreatedAt)
        if err != nil {
            return err
        }
    }
    return nil
}
```

### Step 4: Set Up Dual-Write

Write to both old and new tables during transition:

```go
type UserService struct {
    appClient *appent.Client
    cfClient  *ent.Client
}

func (s *UserService) CreateUser(ctx context.Context, email, name string) error {
    // Write to SystemForge
    cfUser, err := s.cfClient.User.Create().
        SetEmail(email).
        SetName(name).
        Save(ctx)
    if err != nil {
        return err
    }

    // Write to app tables
    _, err = s.appClient.User.Create().
        SetID(cfUser.ID). // Use same ID
        SetEmail(email).
        SetName(name).
        Save(ctx)

    return err
}
```

### Step 5: Switch Reads

Gradually switch reads to SystemForge:

```go
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    // Read from SystemForge
    cfUser, err := s.cfClient.User.Get(ctx, id)
    if err != nil {
        return nil, err
    }

    // Optionally fetch app-specific data
    profile, _ := s.appClient.UserProfile.Query().
        Where(userprofile.CfUserIDEQ(id)).
        Only(ctx)

    return &User{
        User:    cfUser,
        Profile: profile,
    }, nil
}
```

### Step 6: Remove Old Tables

After validation, remove the old user tables:

```sql
-- Archive first
CREATE TABLE users_archive AS SELECT * FROM users;

-- Drop old table
DROP TABLE users;
```

## OAuth Integration

### Add OAuth to Existing Auth

```go
import (
    "github.com/grokify/systemforge/identity/oauth"
)

func setupAuth(entClient *ent.Client) {
    // Create OAuth provider
    cfg := oauth.DefaultConfig("https://api.example.com", []byte("secret"))
    provider, _ := oauth.NewProvider(entClient, cfg)

    // Create OAuth API (endpoints auto-registered)
    api, _ := oauth.NewAPI(provider)

    // Mount to existing router
    router.Mount("/", api.Router())
}
```

### Link Existing Sessions

```go
func linkSession(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    // Get existing session
    session := getExistingSession(r)

    // Find SystemForge user
    cfUser, _ := cfClient.User.Query().
        Where(user.EmailEQ(session.Email)).
        Only(ctx)

    // Store in context for OAuth
    ctx = WithUserID(ctx, cfUser.ID)

    // Continue to OAuth authorize
    oauthHandler.AuthorizeEndpoint(w, r.WithContext(ctx))
}
```

## Multi-Tenant Integration

### Map Existing Tenants

```go
func syncOrganizations(ctx context.Context) error {
    // Get existing tenants
    tenants, _ := appClient.Tenant.Query().All(ctx)

    for _, t := range tenants {
        _, err := cfClient.Organization.Create().
            SetID(t.ID).
            SetName(t.Name).
            SetSlug(t.Slug).
            SetPlan(mapPlan(t.Plan)).
            Save(ctx)
        if err != nil {
            return err
        }
    }
    return nil
}
```

### Map Existing Memberships

```go
func syncMemberships(ctx context.Context) error {
    members, _ := appClient.TenantMember.Query().All(ctx)

    for _, m := range members {
        _, err := cfClient.Membership.Create().
            SetUserID(m.UserID).
            SetOrganizationID(m.TenantID).
            SetRole(mapRole(m.Role)).
            Save(ctx)
        if err != nil {
            return err
        }
    }
    return nil
}
```

## Validation Checklist

- [ ] SystemForge tables created (`cf_*`)
- [ ] Existing data migrated
- [ ] Dual-write enabled
- [ ] Read paths switched
- [ ] OAuth endpoints working
- [ ] Multi-tenant queries working
- [ ] Old tables archived
- [ ] Dual-write disabled
- [ ] Old tables dropped
