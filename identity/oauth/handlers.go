package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/ory/fosite"
)

// Handler provides HTTP handlers for OAuth 2.0 endpoints.
type Handler struct {
	provider *Provider
}

// NewHandler creates a new OAuth HTTP handler.
func NewHandler(provider *Provider) *Handler {
	return &Handler{provider: provider}
}

// AuthorizeEndpoint handles GET/POST /oauth/authorize.
// This is where the authorization flow begins.
func (h *Handler) AuthorizeEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	oauth2 := h.provider.OAuth2Provider()

	// Parse the authorization request
	ar, err := oauth2.NewAuthorizeRequest(ctx, r)
	if err != nil {
		oauth2.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	// For this example, we assume the user is already authenticated
	// In a real implementation, you would:
	// 1. Check if user is logged in
	// 2. If not, redirect to login page
	// 3. If logged in, check for existing consent
	// 4. If no consent, show consent screen
	// 5. After consent, continue here

	// Get user ID from session/context (you need to implement this)
	userID := getUserIDFromSession(r)
	if userID == "" {
		// Redirect to login
		http.Redirect(w, r, "/login?redirect="+r.URL.String(), http.StatusFound)
		return
	}

	// Create session for the user
	session := h.provider.Session(userID)

	// Grant the requested scopes (in production, filter based on user consent)
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}

	// Create the authorization response
	response, err := oauth2.NewAuthorizeResponse(ctx, ar, session)
	if err != nil {
		oauth2.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	// Write the response (redirect with code)
	oauth2.WriteAuthorizeResponse(ctx, w, ar, response)
}

// TokenEndpoint handles POST /oauth/token.
// This handles all token grant types.
func (h *Handler) TokenEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	oauth2 := h.provider.OAuth2Provider()

	// Create session (will be populated by the grant handler)
	session := h.provider.Session("")

	// Process the token request
	ar, err := oauth2.NewAccessRequest(ctx, r, session)
	if err != nil {
		oauth2.WriteAccessError(ctx, w, ar, err)
		return
	}

	// Grant the requested scopes
	for _, scope := range ar.GetRequestedScopes() {
		if ar.GetGrantTypes().ExactOne("client_credentials") {
			// For client_credentials, grant based on client's allowed scopes
			ar.GrantScope(scope)
		} else {
			// For other grants, scopes are already validated
			ar.GrantScope(scope)
		}
	}

	// Create the token response
	response, err := oauth2.NewAccessResponse(ctx, ar)
	if err != nil {
		oauth2.WriteAccessError(ctx, w, ar, err)
		return
	}

	// Write the response
	oauth2.WriteAccessResponse(ctx, w, ar, response)
}

// IntrospectionEndpoint handles POST /oauth/introspect.
// This allows resource servers to validate tokens.
func (h *Handler) IntrospectionEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	oauth2 := h.provider.OAuth2Provider()

	session := h.provider.Session("")

	ir, err := oauth2.NewIntrospectionRequest(ctx, r, session)
	if err != nil {
		// Introspection MUST return 200 even for invalid tokens
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"active": false,
		})
		return
	}

	oauth2.WriteIntrospectionResponse(ctx, w, ir)
}

// RevocationEndpoint handles POST /oauth/revoke.
// This allows clients to revoke tokens.
func (h *Handler) RevocationEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	oauth2 := h.provider.OAuth2Provider()

	err := oauth2.NewRevocationRequest(ctx, r)
	if err != nil {
		oauth2.WriteRevocationResponse(ctx, w, err)
		return
	}

	oauth2.WriteRevocationResponse(ctx, w, nil)
}

// JWKSEndpoint handles GET /.well-known/jwks.json.
// This returns the public keys for token verification.
func (h *Handler) JWKSEndpoint(w http.ResponseWriter, r *http.Request) {
	// In production, you would return the actual JWK set
	// For now, return an empty set
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"keys":[]}`))
}

// WellKnownEndpoint handles GET /.well-known/openid-configuration.
// This returns the OIDC discovery document.
func (h *Handler) WellKnownEndpoint(w http.ResponseWriter, r *http.Request) {
	issuer := h.provider.config.Issuer

	config := map[string]interface{}{
		"issuer":                 issuer,
		"authorization_endpoint": issuer + "/oauth/authorize",
		"token_endpoint":         issuer + "/oauth/token",
		"introspection_endpoint": issuer + "/oauth/introspect",
		"revocation_endpoint":    issuer + "/oauth/revoke",
		"jwks_uri":               issuer + "/.well-known/jwks.json",
		"response_types_supported": []string{
			"code",
			"token",
		},
		"grant_types_supported": []string{
			"authorization_code",
			"refresh_token",
			"client_credentials",
		},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
			"none",
		},
		"code_challenge_methods_supported": []string{
			"S256",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

// getUserIDFromSession extracts the user ID from the request session.
// This is a placeholder - implement based on your session management.
func getUserIDFromSession(r *http.Request) string {
	// In a real implementation, you would:
	// 1. Check for session cookie
	// 2. Validate the session
	// 3. Return the user ID from the session

	// For now, check for a header (used in BFF pattern)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}

	return ""
}

// Middleware provides OAuth token validation middleware.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract token from Authorization header
		token := fosite.AccessTokenFromRequest(r)
		if token == "" {
			http.Error(w, "missing access token", http.StatusUnauthorized)
			return
		}

		// Create session for introspection
		session := h.provider.Session("")

		// Validate the token
		_, ar, err := h.provider.OAuth2Provider().IntrospectToken(ctx, token, fosite.AccessToken, session)
		if err != nil {
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}

		// Add token info to context
		ctx = WithAccessRequest(ctx, ar)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
