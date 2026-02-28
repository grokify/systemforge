# Memberships

Memberships link users to organizations with specific roles.

## Schema

The `cf_memberships` table contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | Foreign key to user |
| `organization_id` | UUID | Foreign key to organization |
| `role` | string | Role within organization |
| `permissions` | JSON | Fine-grained permissions |
| `created_at` | time | Membership start |
| `updated_at` | time | Last modification |

**Unique Constraint**: One membership per user per organization.

## Roles

Roles are stored as strings, allowing apps to define their own vocabulary:

```go
// SaaS app roles
const (
    RoleOwner  = "owner"
    RoleAdmin  = "admin"
    RoleMember = "member"
    RoleGuest  = "guest"
)

// LMS app roles
const (
    RoleInstructor = "instructor"
    RoleStudent    = "student"
    RoleTA         = "ta"
)
```

## Creating Memberships

### Add User to Organization

```go
membership, err := client.Membership.Create().
    SetUserID(userID).
    SetOrganizationID(orgID).
    SetRole("member").
    Save(ctx)
```

### With Permissions

```go
membership, err := client.Membership.Create().
    SetUserID(userID).
    SetOrganizationID(orgID).
    SetRole("member").
    SetPermissions(map[string]any{
        "can_invite": true,
        "can_export": false,
        "max_projects": 5,
    }).
    Save(ctx)
```

## Querying Memberships

### User's Role in Organization

```go
membership, err := client.Membership.Query().
    Where(
        membership.UserIDEQ(userID),
        membership.OrganizationIDEQ(orgID),
    ).
    Only(ctx)

fmt.Printf("Role: %s\n", membership.Role)
```

### All Admins in Organization

```go
admins, err := client.Membership.Query().
    Where(
        membership.OrganizationIDEQ(orgID),
        membership.RoleIn("owner", "admin"),
    ).
    WithUser().
    All(ctx)

for _, m := range admins {
    fmt.Printf("Admin: %s\n", m.Edges.User.Email)
}
```

### User's Organizations with Roles

```go
memberships, err := client.Membership.Query().
    Where(membership.UserIDEQ(userID)).
    WithOrganization().
    All(ctx)

for _, m := range memberships {
    fmt.Printf("Org: %s, Role: %s\n", m.Edges.Organization.Name, m.Role)
}
```

## Updating Memberships

### Change Role

```go
_, err := client.Membership.Update().
    Where(
        membership.UserIDEQ(userID),
        membership.OrganizationIDEQ(orgID),
    ).
    SetRole("admin").
    Save(ctx)
```

### Update Permissions

```go
_, err := client.Membership.Update().
    Where(
        membership.UserIDEQ(userID),
        membership.OrganizationIDEQ(orgID),
    ).
    SetPermissions(map[string]any{
        "can_invite": true,
        "can_export": true,
    }).
    Save(ctx)
```

## Removing Memberships

```go
_, err := client.Membership.Delete().
    Where(
        membership.UserIDEQ(userID),
        membership.OrganizationIDEQ(orgID),
    ).
    Exec(ctx)
```

## Role Hierarchy

Implement role hierarchy in your application:

```go
var roleHierarchy = map[string]int{
    "owner":  100,
    "admin":  80,
    "member": 50,
    "guest":  10,
}

func HasMinRole(membership *ent.Membership, requiredRole string) bool {
    userLevel := roleHierarchy[membership.Role]
    requiredLevel := roleHierarchy[requiredRole]
    return userLevel >= requiredLevel
}

// Usage
if HasMinRole(membership, "admin") {
    // User is admin or higher
}
```

## Permission Checks

### Simple Role Check

```go
func CanManageUsers(membership *ent.Membership) bool {
    return membership.Role == "owner" || membership.Role == "admin"
}
```

### Fine-Grained Permissions

```go
func HasPermission(membership *ent.Membership, permission string) bool {
    // Check role-based defaults
    if membership.Role == "owner" {
        return true
    }

    // Check explicit permissions
    if membership.Permissions != nil {
        if val, ok := membership.Permissions[permission].(bool); ok {
            return val
        }
    }

    return false
}

// Usage
if HasPermission(membership, "can_invite") {
    // Allow invite
}
```

## Authorization Middleware

```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            membership := MembershipFromContext(r.Context())
            if membership == nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            for _, role := range roles {
                if membership.Role == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }

            http.Error(w, "forbidden", http.StatusForbidden)
        })
    }
}

// Usage
r.With(RequireRole("owner", "admin")).Get("/settings", settingsHandler)
```
