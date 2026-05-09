package multiapp

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// appContextKey stores the app context in request context.
	appContextKey contextKey = "systemforge.multiapp.app_context"
)

// AppContext contains information about the current app for a request.
// This is injected by the app context middleware in multi-app mode.
type AppContext struct {
	// AppID is the unique identifier for this app.
	AppID string

	// AppSlug is the URL-safe identifier.
	AppSlug string

	// AppName is the human-readable display name.
	AppName string

	// DatabaseSchema is the PostgreSQL schema for this app.
	DatabaseSchema string

	// Features lists enabled features for this app.
	Features []string

	// Settings contains app-specific configuration.
	Settings map[string]any
}

// WithAppContext returns a new context with the app context attached.
func WithAppContext(ctx context.Context, appCtx *AppContext) context.Context {
	return context.WithValue(ctx, appContextKey, appCtx)
}

// AppContextFromContext extracts the app context from the request context.
// Returns nil if no app context is present (e.g., in single-app mode without middleware).
func AppContextFromContext(ctx context.Context) *AppContext {
	v := ctx.Value(appContextKey)
	if v == nil {
		return nil
	}
	appCtx, ok := v.(*AppContext)
	if !ok {
		return nil
	}
	return appCtx
}

// HasAppContext checks if an app context is present in the context.
func HasAppContext(ctx context.Context) bool {
	return AppContextFromContext(ctx) != nil
}

// AppSlugFromContext returns the app slug from context, or empty string if not present.
func AppSlugFromContext(ctx context.Context) string {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil {
		return ""
	}
	return appCtx.AppSlug
}

// AppIDFromContext returns the app ID from context, or empty string if not present.
func AppIDFromContext(ctx context.Context) string {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil {
		return ""
	}
	return appCtx.AppID
}

// DatabaseSchemaFromContext returns the database schema from context, or empty string if not present.
func DatabaseSchemaFromContext(ctx context.Context) string {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil {
		return ""
	}
	return appCtx.DatabaseSchema
}

// HasFeature checks if a feature is enabled for the current app.
func HasFeature(ctx context.Context, feature string) bool {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil {
		return false
	}
	for _, f := range appCtx.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetSetting retrieves a setting value for the current app.
// Returns nil if the setting doesn't exist or no app context is present.
func GetSetting(ctx context.Context, key string) any {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil || appCtx.Settings == nil {
		return nil
	}
	return appCtx.Settings[key]
}
