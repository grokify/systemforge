package oauth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ory/fosite"
)

// API provides HTTP handlers for OAuth 2.0 endpoints using Huma/Chi.
// It uses a hybrid approach: discovery endpoints use Huma for full typed handling,
// while Fosite-integrated endpoints use Chi handlers directly.
type API struct {
	huma     huma.API
	router   chi.Router
	provider *Provider
	logger   *slog.Logger
}

// Option configures an API.
type Option func(*API)

// WithLogger sets the logger for the API.
// If not set, slog.Default() is used.
func WithLogger(logger *slog.Logger) Option {
	return func(a *API) {
		a.logger = logger
	}
}

// NewAPI creates a new OAuth API with Chi router and Huma.
func NewAPI(provider *Provider, opts ...Option) (*API, error) {
	api := &API{
		provider: provider,
		logger:   slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(api)
	}

	router := chi.NewRouter()

	// Add standard middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(api.loggerMiddleware)

	// Create Huma API with OpenAPI configuration
	config := huma.DefaultConfig("OAuth 2.0 / OpenID Connect API", "1.0")
	config.Info.Description = "OAuth 2.0 Authorization Server with OpenID Connect support"

	// Add contact info
	config.Info.Contact = &huma.Contact{
		Name: "SystemForge",
		URL:  "https://github.com/grokify/systemforge",
	}

	// Add license
	config.Info.License = &huma.License{
		Name: "MIT",
		URL:  "https://opensource.org/licenses/MIT",
	}

	// Configure servers based on issuer
	config.Servers = []*huma.Server{
		{URL: provider.config.Issuer, Description: "OAuth API"},
	}

	humaAPI := humachi.New(router, config)

	api.huma = humaAPI
	api.router = router

	// Register all endpoints
	api.registerEndpoints()

	return api, nil
}

// Logger returns the API's logger.
func (a *API) Logger() *slog.Logger {
	return a.logger
}

// loggerMiddleware injects the logger into the request context.
func (a *API) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := withLogger(r.Context(), a.logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggerKey is the context key for the logger.
type loggerKey struct{}

// withLogger adds a logger to the context.
func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext returns the logger from context, or slog.Default() if not set.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// Router returns the Chi router for mounting or serving.
func (a *API) Router() chi.Router {
	return a.router
}

// Huma returns the underlying Huma API for advanced configuration.
func (a *API) Huma() huma.API {
	return a.huma
}

// Provider returns the OAuth provider.
func (a *API) Provider() *Provider {
	return a.provider
}

// ServeHTTP implements http.Handler.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// registerEndpoints registers all OAuth endpoints.
func (a *API) registerEndpoints() {
	// Discovery endpoints via Huma (typed handlers)
	a.registerDiscoveryEndpoints()

	// Fosite endpoints via Chi (raw handlers for Fosite integration)
	a.registerFositeEndpoints()
}

// Middleware provides OAuth token validation middleware.
func (a *API) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := LoggerFromContext(ctx)

		// Extract token from Authorization header
		token := fosite.AccessTokenFromRequest(r)
		if token == "" {
			logger.Debug("missing access token", "path", r.URL.Path)
			http.Error(w, "missing access token", http.StatusUnauthorized)
			return
		}

		// Create session for introspection
		session := a.provider.Session("")

		// Validate the token
		_, ar, err := a.provider.OAuth2Provider().IntrospectToken(ctx, token, fosite.AccessToken, session)
		if err != nil {
			logger.Debug("invalid access token", "path", r.URL.Path, "error", err)
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}

		logger.Debug("token validated", "client_id", ar.GetClient().GetID(), "path", r.URL.Path)

		// Add token info to context
		ctx = WithAccessRequest(ctx, ar)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
