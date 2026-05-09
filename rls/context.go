package rls

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is a private type for context keys.
type contextKey string

const (
	tenantIDKey contextKey = "systemforge.rls.tenant_id"
	userIDKey   contextKey = "systemforge.rls.user_id"
)

// ContextWithTenant returns a new context with the tenant ID attached.
func ContextWithTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// ContextWithUser returns a new context with the user ID attached.
func ContextWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// ContextWithTenantAndUser returns a new context with both tenant and user IDs.
func ContextWithTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) context.Context {
	ctx = ContextWithTenant(ctx, tenantID)
	return ContextWithUser(ctx, userID)
}

// TenantIDFromContext extracts the tenant ID from the context.
// Returns uuid.Nil if no tenant ID is present.
func TenantIDFromContext(ctx context.Context) uuid.UUID {
	v := ctx.Value(tenantIDKey)
	if v == nil {
		return uuid.Nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// UserIDFromContext extracts the user ID from the context.
// Returns uuid.Nil if no user ID is present.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	v := ctx.Value(userIDKey)
	if v == nil {
		return uuid.Nil
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// HasTenant checks if a tenant ID is present in the context.
func HasTenant(ctx context.Context) bool {
	return TenantIDFromContext(ctx) != uuid.Nil
}

// HasUser checks if a user ID is present in the context.
func HasUser(ctx context.Context) bool {
	return UserIDFromContext(ctx) != uuid.Nil
}

// TenantIDString returns the tenant ID as a string, or empty string if not present.
func TenantIDString(ctx context.Context) string {
	id := TenantIDFromContext(ctx)
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// UserIDString returns the user ID as a string, or empty string if not present.
func UserIDString(ctx context.Context) string {
	id := UserIDFromContext(ctx)
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}
