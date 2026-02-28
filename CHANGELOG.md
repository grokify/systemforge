# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-28

Initial release of CoreForge, a batteries-included Go platform module for multi-tenant SaaS applications.

### Added

#### Identity Module

- User accounts with email and Argon2id password hashing
- Organizations for multi-tenant applications
- Memberships with flexible role-based relationships
- OAuth account linking (GitHub, Google)
- API key service for machine-to-machine authentication
- OAuth 2.0 server using Fosite with:
  - Authorization Code + PKCE
  - Client Credentials
  - Refresh Token with rotation
  - JWT Bearer (RFC 7523)

#### Session Management

- JWT service supporting HS256/RS256/ES256 algorithms
- DPoP (RFC 9449) proof-of-possession token binding
- Backend for Frontend (BFF) pattern with server-side sessions
- GitHub and Google OAuth social login handlers
- Authentication middleware for Chi router and stdlib

#### Authorization

- Role-based access control (RBAC) with organization-scoped permissions
- Casbin provider for advanced policy rules
- Simple provider for lightweight permission checking
- HTTP middleware for route protection

#### Feature Flags

- Feature flag engine with boolean, percentage, and user list flags
- Organization-scoped flag evaluation
- In-memory store for development

#### Row-Level Security

- PostgreSQL RLS policy generation helpers
- Tenant isolation for multi-tenant data separation
- Ent integration with transaction helpers

#### Documentation

- Comprehensive MkDocs documentation site
- Getting started guides and API reference
- PRD and TRD design documents
- Migration guide for existing applications

### Technical Details

- All tables use `cf_` prefix for side-by-side migration
- UUID primary keys throughout
- Ent ORM for type-safe database access
- Reusable mixins for common patterns

[0.1.0]: https://github.com/grokify/coreforge/releases/tag/v0.1.0
