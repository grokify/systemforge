# Observability

CoreForge integrates with [omniobserve](https://github.com/plexusone/omniobserve) to provide vendor-agnostic observability including metrics, traces, and logs with support for multiple backends.

## Supported Backends

- **OTLP** - OpenTelemetry Protocol (vendor-agnostic)
- **Datadog**
- **New Relic**
- **Dynatrace**

## Quick Start

### 1. Import the Provider

Import the observability package and the provider you want to use:

```go
import (
    "github.com/grokify/coreforge/observability"
    _ "github.com/plexusone/omniobserve/observops/otlp"     // OTLP
    // or
    _ "github.com/plexusone/omniobserve/observops/datadog"  // Datadog
    // or
    _ "github.com/plexusone/omniobserve/observops/newrelic" // New Relic
)
```

### 2. Create Observability Instance

```go
obs, err := observability.New(observability.Config{
    Provider:       "otlp",
    ServiceName:    "my-app",
    ServiceVersion: "1.0.0",
    Endpoint:       "localhost:4317",
})
if err != nil {
    log.Fatal(err)
}
defer obs.Shutdown(context.Background())
```

### 3. Integrate with CoreForge Components

#### CoreAuth (OAuth Server)

```go
server, err := coreauth.NewEmbedded(cfg,
    coreauth.WithObservability(obs),
)
```

#### Rate Limiting

```go
limiter := ratelimit.New(storage,
    ratelimit.WithObservability(obs),
)
router.Use(limiter.Middleware())
```

#### JWT Middleware

```go
router.Use(middleware.HTTPAuthWithObservability(jwtService, obs))
```

#### API Key Middleware

```go
config := middleware.DefaultAPIKeyMiddlewareConfig()
config.Service = apikeyService
config.Observability = obs
router.Use(middleware.APIKeyMiddleware(config))
```

#### HTTP Request Tracing

```go
router.Use(obs.Middleware())
```

#### slog Integration

```go
// Use observability-integrated logging
handler := obs.SlogHandler(
    observops.WithSlogLocalHandler(slog.NewJSONHandler(os.Stdout, nil)),
)
slog.SetDefault(slog.New(handler))
```

## Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `OBSERVABILITY_PROVIDER` | Backend provider | `otlp`, `datadog`, `newrelic`, `dynatrace` |
| `OBSERVABILITY_ENDPOINT` | Backend endpoint | `localhost:4317` |
| `OBSERVABILITY_API_KEY` | API key (if required) | |
| `OBSERVABILITY_SERVICE_NAME` | Service name | `my-app` |
| `OBSERVABILITY_SERVICE_VERSION` | Service version | `1.0.0` |
| `OBSERVABILITY_DISABLED` | Disable observability | `true`, `false` |
| `OBSERVABILITY_INSECURE` | Disable TLS | `true`, `false` |
| `OBSERVABILITY_DEBUG` | Enable debug logging | `true`, `false` |

### Configuration from Environment

```go
obs, err := observability.New(observability.ConfigFromEnv())
```

## Metrics

### CoreAuth Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coreforge.coreauth.auth_requests_total` | Counter | grant_type, client_id, status | Authentication requests |
| `coreforge.coreauth.auth_latency_ms` | Histogram | grant_type, endpoint | Request latency |
| `coreforge.coreauth.tokens_issued_total` | Counter | grant_type, client_id | Tokens issued |
| `coreforge.coreauth.token_validations_total` | Counter | result | Token validations |
| `coreforge.coreauth.sessions_active` | Gauge | | Active sessions |

### Rate Limiting Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coreforge.coreapi.ratelimit_requests_total` | Counter | policy_id, client_id, allowed | Rate limit checks |
| `coreforge.coreapi.ratelimit_quota_usage` | Gauge | policy_id, client_id, window | Current usage ratio |

### Session Middleware Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `coreforge.session.jwt_validations_total` | Counter | result | JWT validations |
| `coreforge.session.jwt_validation_latency_ms` | Histogram | | Validation latency |
| `coreforge.session.apikey_validations_total` | Counter | result | API key validations |

## Traces

### Span Names

| Span | Description |
|------|-------------|
| `coreforge.coreauth.authorize` | OAuth authorization endpoint |
| `coreforge.coreauth.token` | OAuth token endpoint |
| `coreforge.coreauth.introspect` | Token introspection |
| `coreforge.coreauth.revoke` | Token revocation |
| `coreforge.http.request` | HTTP request |
| `coreforge.session.jwt_validation` | JWT validation |
| `coreforge.session.apikey_validation` | API key validation |
| `coreforge.ratelimit.check` | Rate limit check |

## Testing with Jaeger

Run Jaeger locally to test OTLP integration:

```bash
docker run -d --name jaeger \
  -p 16686:16686 -p 4317:4317 \
  jaegertracing/all-in-one:latest
```

Configure your app:

```bash
OBSERVABILITY_PROVIDER=otlp \
OBSERVABILITY_ENDPOINT=localhost:4317 \
OBSERVABILITY_INSECURE=true \
go run ./cmd/server
```

View traces at http://localhost:16686

## Recording Custom Metrics

Use the underlying observops.Provider for custom metrics:

```go
meter := obs.Meter()

counter, _ := meter.Counter("my_custom_counter",
    observops.WithDescription("My custom counter"),
)

counter.Add(ctx, 1, observops.WithAttributes(
    observops.Attribute("key", "value"),
))
```

## Creating Custom Spans

```go
ctx, span := obs.StartSpan(ctx, "my.custom.operation",
    observops.WithSpanKind(observops.SpanKindInternal),
    observops.WithSpanAttributes(
        observops.Attribute("user.id", userID),
    ),
)
defer span.End()

// ... your code here ...

if err != nil {
    span.RecordError(err)
    span.SetStatus(observops.StatusCodeError, err.Error())
}
```

## Disabling Observability

To disable observability entirely:

```go
obs, _ := observability.New(observability.Config{
    Disabled: true,
})
```

Or via environment:

```bash
OBSERVABILITY_DISABLED=true
```

When disabled, all operations are no-ops with minimal overhead.
