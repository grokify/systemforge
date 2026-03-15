package observability

// Metric name constants following OpenTelemetry semantic conventions.
// All metrics are prefixed with "coreforge." for namespacing.
const (
	// MetricPrefix is the prefix for all CoreForge metrics.
	MetricPrefix = "coreforge."

	// CoreAuth metrics

	// MetricAuthRequests counts total authentication/authorization requests.
	// Labels: grant_type, client_id, status
	MetricAuthRequests = MetricPrefix + "coreauth.auth_requests_total"

	// MetricAuthLatency records authentication latency in milliseconds.
	// Labels: grant_type, endpoint
	MetricAuthLatency = MetricPrefix + "coreauth.auth_latency_ms"

	// MetricTokensIssued counts total tokens issued.
	// Labels: grant_type, client_id
	MetricTokensIssued = MetricPrefix + "coreauth.tokens_issued_total"

	// MetricTokenValidations counts total token validations.
	// Labels: result (valid, invalid, expired)
	MetricTokenValidations = MetricPrefix + "coreauth.token_validations_total"

	// MetricSessionsActive records currently active sessions.
	MetricSessionsActive = MetricPrefix + "coreauth.sessions_active"

	// Rate limiting metrics

	// MetricRateLimitRequests counts rate limit checks.
	// Labels: policy_id, client_id, allowed
	MetricRateLimitRequests = MetricPrefix + "coreapi.ratelimit_requests_total"

	// MetricRateLimitUsage records current quota usage ratio.
	// Labels: policy_id, client_id, window
	MetricRateLimitUsage = MetricPrefix + "coreapi.ratelimit_quota_usage"

	// Session middleware metrics

	// MetricJWTValidations counts JWT validations.
	// Labels: result (valid, invalid, expired, missing)
	MetricJWTValidations = MetricPrefix + "session.jwt_validations_total"

	// MetricJWTLatency records JWT validation latency in milliseconds.
	MetricJWTLatency = MetricPrefix + "session.jwt_validation_latency_ms"

	// MetricAPIKeyValidations counts API key validations.
	// Labels: result (valid, invalid, expired, revoked)
	MetricAPIKeyValidations = MetricPrefix + "session.apikey_validations_total"
)

// Span name constants for tracing.
const (
	// SpanPrefix is the prefix for all CoreForge spans.
	SpanPrefix = "coreforge."

	// CoreAuth spans

	// SpanAuthorize is the span for the authorization endpoint.
	SpanAuthorize = SpanPrefix + "coreauth.authorize"

	// SpanToken is the span for the token endpoint.
	SpanToken = SpanPrefix + "coreauth.token"

	// SpanIntrospect is the span for token introspection.
	SpanIntrospect = SpanPrefix + "coreauth.introspect"

	// SpanRevoke is the span for token revocation.
	SpanRevoke = SpanPrefix + "coreauth.revoke"

	// HTTP middleware spans

	// SpanHTTPRequest is the span for HTTP requests.
	SpanHTTPRequest = SpanPrefix + "http.request"

	// Session middleware spans

	// SpanJWTValidation is the span for JWT validation.
	SpanJWTValidation = SpanPrefix + "session.jwt_validation"

	// SpanAPIKeyValidation is the span for API key validation.
	SpanAPIKeyValidation = SpanPrefix + "session.apikey_validation"

	// Rate limiting spans

	// SpanRateLimitCheck is the span for rate limit checks.
	SpanRateLimitCheck = SpanPrefix + "ratelimit.check"
)

// Validation result constants for metric labels.
const (
	ResultValid   = "valid"
	ResultInvalid = "invalid"
	ResultExpired = "expired"
	ResultMissing = "missing"
	ResultRevoked = "revoked"
	ResultDenied  = "denied"
	ResultError   = "error"
)

// Status constants for metric labels.
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusDenied  = "denied"
)
