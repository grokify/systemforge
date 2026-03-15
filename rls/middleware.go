package rls

import (
	"database/sql"
	"net/http"

	"github.com/grokify/coreforge/session/middleware"
)

// Middleware provides HTTP middleware that automatically sets PostgreSQL
// RLS context from JWT claims.
type Middleware struct {
	db     *sql.DB
	helper *Helper
}

// NewMiddleware creates a new RLS middleware.
func NewMiddleware(db *sql.DB, cfg *Config) *Middleware {
	return &Middleware{
		db:     db,
		helper: NewHelper(cfg),
	}
}

// SetRLSContext returns middleware that sets PostgreSQL session variables
// from the authenticated user's JWT claims.
//
// This middleware should be used AFTER authentication middleware.
// It extracts the user ID and organization ID from the JWT claims
// and sets them as PostgreSQL session variables for RLS policies.
//
// Usage with Chi:
//
//	r.Use(middleware.ChiAuth(jwtService))
//	r.Use(rlsMiddleware.SetRLSContext())
func (m *Middleware) SetRLSContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get claims from context (set by auth middleware)
			claims := middleware.ClaimsFromContext(ctx)
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Set tenant context if organization is present
			if claims.OrganizationID != nil {
				if err := m.helper.SetTenant(ctx, m.db, claims.OrganizationID.String()); err != nil {
					http.Error(w, "failed to set tenant context", http.StatusInternalServerError)
					return
				}

				// Also add to Go context for non-DB use
				ctx = ContextWithTenant(ctx, *claims.OrganizationID)
			}

			// Set user context
			if err := m.helper.SetUser(ctx, m.db, claims.PrincipalID.String()); err != nil {
				http.Error(w, "failed to set user context", http.StatusInternalServerError)
				return
			}
			ctx = ContextWithUser(ctx, claims.PrincipalID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant returns middleware that ensures a tenant context is present.
// Returns 403 Forbidden if no organization is in the JWT claims.
func (m *Middleware) RequireTenant() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := middleware.ClaimsFromContext(r.Context())
			if claims == nil || claims.OrganizationID == nil {
				http.Error(w, "tenant context required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DBWithRLS wraps a database connection to automatically set RLS context
// for each connection obtained from the pool.
type DBWithRLS struct {
	*sql.DB
	helper *Helper
}

// NewDBWithRLS creates a new RLS-aware database wrapper.
func NewDBWithRLS(db *sql.DB, cfg *Config) *DBWithRLS {
	return &DBWithRLS{
		DB:     db,
		helper: NewHelper(cfg),
	}
}
