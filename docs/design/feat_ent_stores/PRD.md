# PRD: Ent Store Implementations

> **Status**: Draft
> **Target**: SystemForge v0.5.0

## Overview

Provide reference Ent (database) implementations for SystemForge's BFF session store and API key store interfaces. This enables apps to use PostgreSQL-backed storage with minimal boilerplate.

## Problem Statement

Currently, SystemForge provides:
- `session/bff.Store` interface with only `MemoryStore` implementation
- `identity/apikey.Store` interface with no implementations

Apps must write their own database implementations, leading to:
1. **Duplicated Effort**: Each app implements the same patterns
2. **Inconsistent Quality**: Varying levels of security and error handling
3. **Slower Adoption**: Barrier to entry for new apps
4. **Maintenance Burden**: Bugs fixed in one app aren't shared

## Goals

1. **Reference Implementations**: Production-ready Ent stores
2. **Ent Schema Mixins**: Reusable schema definitions
3. **App Flexibility**: Apps can customize or use as-is
4. **Security by Default**: Proper encryption, indexing, cleanup

## Non-Goals

- Redis store implementations (future iteration)
- Non-Ent ORMs (gorm, sqlx, etc.)
- Automatic schema migration (apps manage their own)

## User Stories

### App Developers

**US-1**: As a developer, I can add BFF sessions to my app with 3 lines of code.

**US-2**: As a developer, I can add API keys to my app without writing storage code.

**US-3**: As a developer, I can customize the schema if my app needs additional fields.

**US-4**: As a developer, I can use SystemForge mixins and generate Ent code.

### Platform Operators

**US-5**: As an operator, sessions are automatically cleaned up when expired.

**US-6**: As an operator, API keys are stored with hashed values (not plaintext).

**US-7**: As an operator, I can query sessions/keys by user for account management.

## Target Applications

| Application | BFF Sessions | API Keys |
|-------------|--------------|----------|
| App3 | Yes | Yes |
| App1 | Yes | Planned |
| Dashforge | Yes | Planned |

## Success Metrics

1. **Adoption**: All 3 apps using provided stores within 6 months
2. **Code Reduction**: 80% less auth storage code per app
3. **Security**: Zero storage-related auth vulnerabilities

## Deliverables

### 1. Ent Schema Mixins

```
identity/ent/mixin/
├── bff_session.go    # BFF session fields
└── api_key.go        # API key fields
```

### 2. Reference Implementations

```
session/bff/
└── store_ent.go      # Ent-backed BFF session store

identity/apikey/
└── store_ent.go      # Ent-backed API key store
```

### 3. Example App Integration

Documentation showing:
- How to import mixins into app schema
- How to wire up stores in app
- How to customize if needed

## Dependencies

- `entgo.io/ent` (Ent ORM)
- PostgreSQL (primary target)
- SQLite (for testing)

## Risks

| Risk | Mitigation |
|------|------------|
| Ent version incompatibility | Pin to stable Ent version, test with multiple versions |
| Schema migration conflicts | Document migration patterns, provide upgrade scripts |
| Performance at scale | Benchmark with realistic data, optimize indexes |
