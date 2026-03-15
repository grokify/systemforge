// Package middleware provides HTTP middleware for CoreForge session management.
package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/session/jwt"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	claimsKey contextKey = "coreforge.claims"
)

// ContextWithClaims returns a new context with the JWT claims attached.
func ContextWithClaims(ctx context.Context, claims *jwt.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts JWT claims from the context.
// Returns nil if no claims are present.
func ClaimsFromContext(ctx context.Context) *jwt.Claims {
	claims, _ := ctx.Value(claimsKey).(*jwt.Claims)
	return claims
}

// PrincipalIDFromContext extracts the principal ID from the context.
// Returns uuid.Nil if no claims or principal ID is present.
func PrincipalIDFromContext(ctx context.Context) uuid.UUID {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return uuid.Nil
	}
	return claims.PrincipalID
}

// PrincipalTypeFromContext extracts the principal type from the context.
// Returns empty string if no claims are present.
func PrincipalTypeFromContext(ctx context.Context) string {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return ""
	}
	return claims.PrincipalType
}

// UserIDFromContext extracts the principal ID from the context.
// Deprecated: Use PrincipalIDFromContext instead.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	return PrincipalIDFromContext(ctx)
}

// OrganizationIDFromContext extracts the organization ID from the context.
// Returns nil if no organization context is present.
func OrganizationIDFromContext(ctx context.Context) *uuid.UUID {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}
	return claims.OrganizationID
}

// RoleFromContext extracts the user's role from the context.
// Returns empty string if no role is present.
func RoleFromContext(ctx context.Context) string {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return ""
	}
	return claims.Role
}

// PermissionsFromContext extracts the user's permissions from the context.
// Returns nil if no permissions are present.
func PermissionsFromContext(ctx context.Context) []string {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return nil
	}
	return claims.Permissions
}

// IsPlatformAdminFromContext checks if the user is a platform admin.
// Returns false if no claims are present.
func IsPlatformAdminFromContext(ctx context.Context) bool {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return false
	}
	return claims.IsPlatformAdmin
}

// HasPermission checks if the user has a specific permission.
func HasPermission(ctx context.Context, permission string) bool {
	permissions := PermissionsFromContext(ctx)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the user has any of the specified permissions.
func HasAnyPermission(ctx context.Context, permissions ...string) bool {
	userPerms := PermissionsFromContext(ctx)
	permSet := make(map[string]bool, len(userPerms))
	for _, p := range userPerms {
		permSet[p] = true
	}
	for _, p := range permissions {
		if permSet[p] {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the user has all of the specified permissions.
func HasAllPermissions(ctx context.Context, permissions ...string) bool {
	userPerms := PermissionsFromContext(ctx)
	permSet := make(map[string]bool, len(userPerms))
	for _, p := range userPerms {
		permSet[p] = true
	}
	for _, p := range permissions {
		if !permSet[p] {
			return false
		}
	}
	return true
}

// HasRole checks if the user has a specific role.
func HasRole(ctx context.Context, role string) bool {
	return RoleFromContext(ctx) == role
}

// HasAnyRole checks if the user has any of the specified roles.
func HasAnyRole(ctx context.Context, roles ...string) bool {
	userRole := RoleFromContext(ctx)
	for _, r := range roles {
		if r == userRole {
			return true
		}
	}
	return false
}
