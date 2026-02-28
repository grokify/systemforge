package bff

import (
	"context"
	"net/http"
	"time"
)

// contextKey is a type for BFF context keys.
type contextKey string

const (
	// ContextKeySession is the context key for the session.
	ContextKeySession contextKey = "bff_session"
)

// MiddlewareConfig contains configuration for the BFF middleware.
type MiddlewareConfig struct {
	// Store is the session store.
	Store Store

	// CookieManager handles session cookies.
	CookieManager *CookieManager

	// RefreshThreshold is how long before access token expiry to trigger refresh.
	// Default: 5 minutes.
	RefreshThreshold time.Duration

	// OnSessionLoad is called after a session is loaded.
	// Can be used for logging or session modification.
	OnSessionLoad func(ctx context.Context, session *Session) error

	// OnSessionExpired is called when a session is expired.
	OnSessionExpired func(w http.ResponseWriter, r *http.Request)

	// OnSessionInvalid is called when a session is invalid.
	OnSessionInvalid func(w http.ResponseWriter, r *http.Request)

	// OnNoSession is called when no session is found and RequireSession is true.
	OnNoSession func(w http.ResponseWriter, r *http.Request)

	// RequireSession when true rejects requests without valid sessions.
	RequireSession bool

	// TouchOnAccess when true updates LastAccessedAt on each request.
	TouchOnAccess bool
}

// SessionMiddleware creates middleware that loads sessions from cookies.
func SessionMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	if config.RefreshThreshold == 0 {
		config.RefreshThreshold = 5 * time.Minute
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract session ID from cookie
			sessionID := config.CookieManager.GetSessionID(r)
			if sessionID == "" {
				if config.RequireSession {
					handleNoSession(w, r, config)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Load session from store
			session, err := config.Store.Get(r.Context(), sessionID)
			if err != nil {
				if err == ErrSessionExpired {
					handleSessionExpired(w, r, config)
					return
				}
				if err == ErrSessionNotFound {
					// Clear the invalid cookie
					config.CookieManager.ClearSessionCookie(w)
					if config.RequireSession {
						handleNoSession(w, r, config)
						return
					}
					next.ServeHTTP(w, r)
					return
				}
				// Other errors
				handleSessionInvalid(w, r, config)
				return
			}

			// Touch session if configured
			if config.TouchOnAccess {
				_ = config.Store.Touch(r.Context(), sessionID)
			}

			// Call session load hook if configured
			if config.OnSessionLoad != nil {
				if err := config.OnSessionLoad(r.Context(), session); err != nil {
					handleSessionInvalid(w, r, config)
					return
				}
			}

			// Add session to context
			ctx := context.WithValue(r.Context(), ContextKeySession, session)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleNoSession handles missing session errors.
func handleNoSession(w http.ResponseWriter, r *http.Request, config MiddlewareConfig) {
	if config.OnNoSession != nil {
		config.OnNoSession(w, r)
		return
	}
	http.Error(w, "Unauthorized: Session required", http.StatusUnauthorized)
}

// handleSessionExpired handles expired session errors.
func handleSessionExpired(w http.ResponseWriter, r *http.Request, config MiddlewareConfig) {
	if config.OnSessionExpired != nil {
		config.OnSessionExpired(w, r)
		return
	}
	http.Error(w, "Unauthorized: Session expired", http.StatusUnauthorized)
}

// handleSessionInvalid handles invalid session errors.
func handleSessionInvalid(w http.ResponseWriter, r *http.Request, config MiddlewareConfig) {
	if config.OnSessionInvalid != nil {
		config.OnSessionInvalid(w, r)
		return
	}
	http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
}

// GetSession retrieves the session from the request context.
func GetSession(ctx context.Context) *Session {
	session, _ := ctx.Value(ContextKeySession).(*Session)
	return session
}

// RequireSessionMiddleware creates middleware that requires a valid session.
// This is a convenience function that wraps SessionMiddleware.
func RequireSessionMiddleware(store Store, cookieManager *CookieManager) func(http.Handler) http.Handler {
	return SessionMiddleware(MiddlewareConfig{
		Store:          store,
		CookieManager:  cookieManager,
		RequireSession: true,
		TouchOnAccess:  true,
	})
}

// OptionalSessionMiddleware creates middleware that loads sessions if present.
// Requests without sessions are allowed through.
func OptionalSessionMiddleware(store Store, cookieManager *CookieManager) func(http.Handler) http.Handler {
	return SessionMiddleware(MiddlewareConfig{
		Store:          store,
		CookieManager:  cookieManager,
		RequireSession: false,
		TouchOnAccess:  true,
	})
}
