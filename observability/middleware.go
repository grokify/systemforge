package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/plexusone/omniobserve/observops"
)

// Middleware creates HTTP middleware that traces requests and records metrics.
// It starts a span for each request and records method, path, status_code, and duration.
func (o *Observability) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Start span for request
			ctx, span := o.provider.Tracer().Start(r.Context(), SpanHTTPRequest,
				observops.WithSpanKind(observops.SpanKindServer),
				observops.WithSpanAttributes(
					observops.Attribute("http.method", r.Method),
					observops.Attribute("http.url", r.URL.String()),
					observops.Attribute("http.target", r.URL.Path),
					observops.Attribute("http.host", r.Host),
					observops.Attribute("http.user_agent", r.UserAgent()),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Inject trace ID into response headers
			sc := span.SpanContext()
			if sc.TraceID != "" {
				wrapped.Header().Set("X-Trace-ID", sc.TraceID)
			}

			// Process request with span context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Record span attributes
			duration := time.Since(start)
			span.SetAttributes(
				observops.Attribute("http.status_code", wrapped.statusCode),
				observops.Attribute("http.response_content_length", wrapped.bytesWritten),
			)

			// Set span status based on HTTP status code
			if wrapped.statusCode >= 400 {
				span.SetStatus(observops.StatusCodeError, http.StatusText(wrapped.statusCode))
			} else {
				span.SetStatus(observops.StatusCodeOK, "")
			}

			// Record request metric
			o.recordHTTPRequest(r.Context(), r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// recordHTTPRequest records HTTP request metrics.
func (o *Observability) recordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	// Use the pre-created auth latency metric for now
	// In a more complete implementation, we'd have a dedicated HTTP request metric
	o.authLatency.Record(ctx, float64(duration.Milliseconds()), observops.WithAttributes(
		observops.Attribute("http.method", method),
		observops.Attribute("http.route", path),
		observops.Attribute("http.status_code", statusCode),
	))
}

// responseWriter wraps http.ResponseWriter to capture the status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written.
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter (for http.ResponseController).
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
