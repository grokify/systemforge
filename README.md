# CoreForge

CoreForge is a batteries-included Go platform module providing reusable identity, session, RBAC, and feature flags for multi-tenant SaaS applications. Think of it as Django/Laravel-style conveniences for Go.

## Features

- **Identity** - User, Organization, Membership, OAuth account management with Ent schemas
- **Session** - JWT tokens, OAuth providers (GitHub, Google), middleware adapters
- **RBAC** - Role-based access control with optional Casbin integration
- **Feature Flags** - Feature flag engine with memory and PostgreSQL stores
- **RLS** - PostgreSQL Row-Level Security helpers

## Installation

```bash
go get github.com/grokify/coreforge
```

## Module Structure

```
github.com/grokify/coreforge/
├── identity/          # User, Organization, Membership, OAuth
├── session/           # JWT, OAuth, middleware
├── rbac/              # Role-based access control
├── featureflags/      # Feature flag engine
└── rls/               # PostgreSQL RLS helpers
```

## Quick Start

### Using Identity Schemas

CoreForge provides Ent schemas that can be used directly or composed via mixins.

#### Direct Schema Usage

Import and use CoreForge schemas directly in your Ent configuration:

```go
package main

import (
    "github.com/grokify/coreforge/identity/ent/schema"
)

// Your app uses CoreForge schemas directly
```

#### Mixin Composition (Recommended)

Compose CoreForge mixins into your own schemas for maximum flexibility:

```go
// your-app/ent/schema/user.go
package schema

import (
    "entgo.io/ent"
    cfmixin "github.com/grokify/coreforge/identity/ent/mixin"
)

type User struct {
    ent.Schema
}

func (User) Mixin() []ent.Mixin {
    return []ent.Mixin{
        cfmixin.UserBase{},  // CoreForge fields
    }
}

func (User) Fields() []ent.Field {
    return []ent.Field{
        // App-specific extensions
        field.String("username").Unique(),
    }
}
```

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Table prefix | `cf_` | Avoids conflicts, enables side-by-side migration |
| Role storage | String field | Apps define own vocabularies |
| OAuth pattern | Separate table | Supports multiple providers per user |
| Refresh tokens | Database-backed | Enables revocation, breach detection |
| Primary keys | UUID | Modern, distributed-friendly |

## Database Tables

CoreForge creates the following tables (all prefixed with `cf_`):

- `cf_users` - User accounts
- `cf_organizations` - Multi-tenant organizations
- `cf_memberships` - User-organization relationships with roles
- `cf_oauth_accounts` - OAuth provider connections
- `cf_refresh_tokens` - JWT refresh token tracking

## Migration Strategy

For existing apps, CoreForge supports side-by-side migration:

1. **Side-by-Side**: Create `cf_*` tables alongside existing tables
2. **Dual-Write**: Write to both old and new tables
3. **Cutover**: Switch reads to CoreForge tables
4. **Cleanup**: Remove old tables

## License

MIT License - see LICENSE file for details.
