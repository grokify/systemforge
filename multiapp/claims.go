package multiapp

import (
	"github.com/grokify/coreforge/session/jwt"
)

// AppClaims extends jwt.Claims with app-specific fields for multi-app mode.
// This is used when you need to include app context in JWT tokens.
type AppClaims struct {
	*jwt.Claims

	// AppID is the app this token was issued for.
	// In multi-app mode, tokens are scoped to a specific app.
	AppID string `json:"app_id,omitempty"`

	// AppSlug is the URL-safe app identifier.
	AppSlug string `json:"app_slug,omitempty"`
}

// WithApp adds app context to the claims.
func (c *AppClaims) WithApp(appID, appSlug string) *AppClaims {
	c.AppID = appID
	c.AppSlug = appSlug
	return c
}

// NewAppClaims creates AppClaims from existing jwt.Claims.
func NewAppClaims(claims *jwt.Claims) *AppClaims {
	return &AppClaims{Claims: claims}
}

// NewAppClaimsForApp creates AppClaims with app context.
func NewAppClaimsForApp(claims *jwt.Claims, appID, appSlug string) *AppClaims {
	return &AppClaims{
		Claims:  claims,
		AppID:   appID,
		AppSlug: appSlug,
	}
}

// AppClaimsFromContext extracts app-aware claims from context.
// If the request has both app context and JWT claims, it combines them.
func AppClaimsFromContext(ctx any) *AppClaims {
	// This is a convenience wrapper - in practice, you'd extract
	// jwt.Claims from middleware.ClaimsFromContext and app context
	// from AppContextFromContext separately.
	return nil
}
