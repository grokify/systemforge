package productgraph

import (
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// RequestTrackerMiddleware creates middleware that tracks HTTP requests to ProductGraph.
// It records api.response events with method, path, status code, and duration.
func RequestTrackerMiddleware(client *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if client == nil || !client.IsEnabled() {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			// Track the request asynchronously
			_ = client.TrackAPICall(
				r.Context(),
				r.Method,
				r.URL.Path,
				ww.status,
				duration,
			)
		})
	}
}

// ChainMiddleware chains correlation and request tracking middleware.
// This is a convenience function that applies both middlewares in the correct order:
// 1. CorrelationMiddleware - extracts session/request IDs
// 2. RequestTrackerMiddleware - tracks API calls with correlation
func ChainMiddleware(client *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Apply in reverse order so correlation runs first
		return CorrelationMiddleware(
			RequestTrackerMiddleware(client)(next),
		)
	}
}
