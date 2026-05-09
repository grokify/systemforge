# CoreForge Roadmap

**Last Updated:** 2026-04-28

## Vision

CoreForge is a batteries-included Go platform module providing reusable identity, session, authorization, and SaaS infrastructure for multi-tenant applications. The goal is Django/Laravel-style conveniences for Go.

## Current State

### Production-Ready

| Module | Status | Description |
|--------|--------|-------------|
| Identity/Auth | Complete | OAuth2, JWT, API Keys, SCIM 2.0, DPoP |
| Session | Complete | JWT service, BFF pattern, middleware |
| Authorization | Complete | RBAC/ReBAC with SpiceDB, simple provider |
| Marketplace | Complete | Listings, licenses, Stripe integration |
| Multi-App | Complete | Schema isolation, routing, shared caching |
| Row-Level Security | Complete | PostgreSQL RLS, tenant isolation |
| Observability | Partial | Metrics, traces, logs via omniobserve |
| Feature Flags | Basic | In-memory store only |
| Rate Limiting | Partial | In-memory only, needs Redis |

## Roadmap

### Phase 1: Security Hardening (High Priority)

#### 1.1 Multi-Factor Authentication (MFA)

Enable second-factor authentication for enhanced security.

- **TOTP Support**: Time-based one-time passwords (RFC 6238)
- **Recovery Codes**: Backup codes for account recovery
- **MFA Enrollment Flow**: User-friendly setup process
- **MFA Verification Middleware**: Enforce MFA on sensitive operations

#### 1.2 Account Security

Protect against common attack vectors.

- **Account Lockout**: Brute-force protection with configurable thresholds
- **Suspicious Activity Detection**: Anomaly detection for logins
- **Session Invalidation**: Cascade logout across all devices
- **Password Policies**: Configurable strength requirements

#### 1.3 Enterprise SSO (Future)

- **SAML 2.0 Federation**: Enterprise identity provider integration
- **OIDC Federation**: OpenID Connect provider support
- **WebAuthn/Passkeys**: Passwordless authentication

### Phase 2: Infrastructure (Medium Priority)

#### 2.1 Distributed Rate Limiting

Enable rate limiting across multiple instances.

- **Redis Backend**: Distributed rate limit storage
- **Custom Quota Management**: Per-tier rate limits
- **Usage Alerts**: Threshold warnings and notifications
- **Quota Dashboard**: Usage visibility

#### 2.2 Redis Feature Flags

Enable feature flag coordination across instances.

- **Redis Store**: Distributed flag storage
- **Flag Rollout Scheduling**: Timed rollouts
- **Gradual Rollouts**: Percentage-based deployment
- **A/B Testing Framework**: Experiment support

#### 2.3 Webhook System

General-purpose outbound webhook infrastructure.

- **Webhook Registration**: Create/update/delete webhooks
- **Event Subscriptions**: Subscribe to specific events
- **Retry Logic**: Exponential backoff with dead letter queue
- **Webhook Signatures**: HMAC verification for security
- **Delivery Logs**: Audit trail for debugging

### Phase 3: Analytics & Correlation (In Progress)

#### 3.1 ProductGraph Integration

Backend-frontend telemetry correlation.

- **Correlation Middleware**: X-Session-ID, X-Request-ID extraction
- **Event Forwarding**: Backend events to ProductGraph
- **Journey Tracking**: Backend-driven flow completion
- **Error Aggregation**: Unified frontend/backend error tracking

#### 3.2 Marketplace Observability

Business metrics for marketplace operations.

- **Checkout Metrics**: Conversion tracking
- **License Metrics**: Grant/revoke tracking
- **Webhook Metrics**: Delivery success rates
- **Revenue Dashboards**: Business intelligence

### Phase 4: Compliance & Privacy (Medium Priority)

#### 4.1 GDPR Compliance

Data privacy and user rights support.

- **Data Export**: User data export (JSON/CSV)
- **Data Deletion**: Account termination with cascade delete
- **Consent Management**: Track and honor user consent
- **Data Residency**: Region-specific storage policies

#### 4.2 Audit & Compliance

Enterprise audit requirements.

- **Audit Log Search**: Query and filter audit events
- **Audit Log Export**: Compliance reporting
- **Retention Policies**: Configurable log retention
- **Compliance Reports**: SOC2, HIPAA templates

### Phase 5: Operations & Admin (Lower Priority)

#### 5.1 Admin Features

Support and debugging capabilities.

- **Admin Impersonation**: Debug user sessions safely
- **Bulk Operations**: Mass user import/export
- **Resource Quotas**: Per-org resource limits
- **Usage Reports**: Organization usage summaries

#### 5.2 Notifications

Communication infrastructure.

- **Email Service**: Transactional email with templates
- **Notification Preferences**: User communication settings
- **Email Verification**: Account verification flow
- **Password Reset**: Secure reset flow

#### 5.3 API Management

Enhanced API key capabilities.

- **API Key Rotation**: Scheduled key rotation
- **Per-Key Rate Limits**: Custom limits per key
- **Usage Metering**: Call counting for billing
- **Key Scoping**: Fine-grained permission scoping

### Phase 6: Documentation (Ongoing)

#### 6.1 GoDoc Coverage

~600 exported items need documentation.

- **Public API Types**: Configuration, middleware, services
- **Authorization Module**: Security-critical documentation
- **Session Components**: JWT, BFF, DPoP
- **Examples**: Example code for complex APIs

#### 6.2 Guides & Examples

Developer experience improvements.

- **Integration Guides**: Common SaaS patterns
- **Example Reference App**: Best practices demonstration
- **Migration Guides**: Upgrade paths
- **Troubleshooting**: Common issues and solutions

## Backlog (Future Consideration)

| Feature | Description | Priority |
|---------|-------------|----------|
| Cedar Policy Language | Alternative to SpiceDB for authorization | P3 |
| OPA Integration | Complex policy evaluation | P3 |
| Mobile Native Auth | iOS/Android SDK support | P3 |
| GraphQL Support | GraphQL API layer | P3 |
| Event Sourcing | Audit trail via events | P3 |
| Multi-Region | Active-active deployment | P3 |

## Dependencies

| Feature | Depends On |
|---------|------------|
| ProductGraph Integration | omniobserve, omnidxi |
| Redis Rate Limiting | Redis infrastructure |
| Redis Feature Flags | Redis infrastructure |
| SAML/OIDC | Enterprise customer demand |
| Webhook System | None (standalone) |

## Success Metrics

| Metric | Target |
|--------|--------|
| Test Coverage | >80% for new code |
| GoDoc Coverage | 100% exported items documented |
| API Stability | Semantic versioning, no breaking changes in minor releases |
| Security | Zero critical vulnerabilities |

## Related Documents

- [TASKS.md](TASKS.md) - Detailed implementation tasks
- [CHANGELOG.md](CHANGELOG.md) - Release history
- [docs/design/](docs/design/) - Feature design documents
