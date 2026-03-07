# CoreForge

**Batteries-included Go platform module for multi-tenant SaaS applications.**

CoreForge provides reusable identity, session, RBAC, and OAuth 2.0 functionality that you can integrate into your Go applications. Think of it as Django/Laravel for Go—pre-built components for the things every SaaS needs.

## Features

- **Identity Management**: Users, organizations, memberships with role-based access
- **OAuth 2.0 Server**: Full RFC-compliant implementation using Fosite
- **API Key Authentication**: Secure server-to-server access
- **Service Accounts**: JWT Bearer authentication for automation
- **Multi-Tenant Ready**: Organization-scoped resources with RLS support
- **Ent ORM**: Type-safe database schemas with migrations

## Quick Example

```go
package main

import (
    "github.com/grokify/coreforge/identity/ent"
    "github.com/grokify/coreforge/identity/oauth"
)

func main() {
    // Connect to database
    client, _ := ent.Open("postgres", "postgres://...")

    // Create OAuth provider
    cfg := oauth.DefaultConfig("https://api.example.com", []byte("secret"))
    provider, _ := oauth.NewProvider(client, cfg)

    // Create OAuth API (Huma/Chi)
    api, _ := oauth.NewAPI(provider)

    // Mount the API router
    http.Handle("/", api.Router())
    http.ListenAndServe(":8080", nil)
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                        │
├─────────────────────────────────────────────────────────────┤
│                        CoreForge                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Identity │  │  OAuth   │  │   RBAC   │  │ Feature  │    │
│  │  Module  │  │  Server  │  │  Module  │  │  Flags   │    │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
├─────────────────────────────────────────────────────────────┤
│                      Ent ORM Layer                           │
├─────────────────────────────────────────────────────────────┤
│                      PostgreSQL                              │
└─────────────────────────────────────────────────────────────┘
```

## Module Status

| Module | Status | Description |
|--------|--------|-------------|
| Identity | ✅ Ready | User, Org, Membership, OAuthAccount |
| OAuth 2.0 | ✅ Ready | Authorization Code, Client Credentials, Refresh Token |
| API Keys | ✅ Ready | Server-to-server authentication |
| Service Accounts | ✅ Ready | JWT Bearer authentication |
| RBAC | 🚧 Planned | Role-based access control |
| Feature Flags | 🚧 Planned | Feature flag engine |
| RLS Helpers | 🚧 Planned | PostgreSQL Row-Level Security |

## Why CoreForge?

Building a SaaS application means implementing the same foundational features over and over:

- User authentication and registration
- Organization/team management
- OAuth 2.0 for third-party integrations
- API key management for developers
- Role-based permissions

CoreForge provides battle-tested implementations of these features so you can focus on your application's unique value.

## Getting Started

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quickstart.md)
- [Configuration](getting-started/configuration.md)

## License

Apache 2.0
