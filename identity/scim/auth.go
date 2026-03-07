package scim

import (
	"context"
	"strings"
)

// SCIM authorization scopes follow the pattern: scim:{resource}:{action}
const (
	// User scopes
	ScopeUsersRead  = "scim:users:read"
	ScopeUsersWrite = "scim:users:write"

	// Group scopes
	ScopeGroupsRead  = "scim:groups:read"
	ScopeGroupsWrite = "scim:groups:write"

	// Full access scope
	ScopeFull = "scim:full"
)

// ScopedAuthorizationHook implements AuthorizationHook using OAuth scopes.
// It checks that the request context contains appropriate scopes for the operation.
type ScopedAuthorizationHook struct {
	// RequireScopes determines whether to enforce scope checking.
	// If false, all operations are allowed (useful for development/testing).
	RequireScopes bool
}

// NewScopedAuthorizationHook creates a new scoped authorization hook.
func NewScopedAuthorizationHook(requireScopes bool) *ScopedAuthorizationHook {
	return &ScopedAuthorizationHook{
		RequireScopes: requireScopes,
	}
}

// CanRead checks if the authenticated user can read the resource.
func (h *ScopedAuthorizationHook) CanRead(ctx context.Context, resourceType, resourceID string) error {
	if !h.RequireScopes {
		return nil
	}

	scopes := AuthScopesFromContext(ctx)
	required := h.readScope(resourceType)

	if !h.hasScope(scopes, required) {
		return ErrForbidden("missing required scope: " + required)
	}

	return nil
}

// CanCreate checks if the authenticated user can create resources of this type.
func (h *ScopedAuthorizationHook) CanCreate(ctx context.Context, resourceType string) error {
	if !h.RequireScopes {
		return nil
	}

	scopes := AuthScopesFromContext(ctx)
	required := h.writeScope(resourceType)

	if !h.hasScope(scopes, required) {
		return ErrForbidden("missing required scope: " + required)
	}

	return nil
}

// CanUpdate checks if the authenticated user can update the resource.
func (h *ScopedAuthorizationHook) CanUpdate(ctx context.Context, resourceType, resourceID string) error {
	if !h.RequireScopes {
		return nil
	}

	scopes := AuthScopesFromContext(ctx)
	required := h.writeScope(resourceType)

	if !h.hasScope(scopes, required) {
		return ErrForbidden("missing required scope: " + required)
	}

	return nil
}

// CanDelete checks if the authenticated user can delete the resource.
func (h *ScopedAuthorizationHook) CanDelete(ctx context.Context, resourceType, resourceID string) error {
	if !h.RequireScopes {
		return nil
	}

	scopes := AuthScopesFromContext(ctx)
	required := h.writeScope(resourceType)

	if !h.hasScope(scopes, required) {
		return ErrForbidden("missing required scope: " + required)
	}

	return nil
}

// readScope returns the read scope for a resource type.
func (h *ScopedAuthorizationHook) readScope(resourceType string) string {
	switch resourceType {
	case ResourceTypeUser:
		return ScopeUsersRead
	case ResourceTypeGroup:
		return ScopeGroupsRead
	default:
		return "scim:" + strings.ToLower(resourceType) + ":read"
	}
}

// writeScope returns the write scope for a resource type.
func (h *ScopedAuthorizationHook) writeScope(resourceType string) string {
	switch resourceType {
	case ResourceTypeUser:
		return ScopeUsersWrite
	case ResourceTypeGroup:
		return ScopeGroupsWrite
	default:
		return "scim:" + strings.ToLower(resourceType) + ":write"
	}
}

// hasScope checks if the required scope is present in the list.
// Also accepts the full access scope as a substitute.
func (h *ScopedAuthorizationHook) hasScope(scopes []string, required string) bool {
	for _, s := range scopes {
		if s == required || s == ScopeFull {
			return true
		}
	}
	return false
}

// RoleBasedAuthorizationHook implements AuthorizationHook using role-based access.
// It checks that the authenticated user has an appropriate role for the operation.
type RoleBasedAuthorizationHook struct {
	// AdminRoles are roles that have full SCIM access.
	AdminRoles []string

	// ReadOnlyRoles are roles that have read-only SCIM access.
	ReadOnlyRoles []string

	// RoleExtractor extracts roles from context.
	// If nil, defaults to looking for "roles" in context values.
	RoleExtractor func(ctx context.Context) []string
}

// NewRoleBasedAuthorizationHook creates a new role-based authorization hook.
func NewRoleBasedAuthorizationHook(adminRoles, readOnlyRoles []string) *RoleBasedAuthorizationHook {
	return &RoleBasedAuthorizationHook{
		AdminRoles:    adminRoles,
		ReadOnlyRoles: readOnlyRoles,
	}
}

// CanRead checks if the authenticated user can read the resource.
func (h *RoleBasedAuthorizationHook) CanRead(ctx context.Context, resourceType, resourceID string) error {
	roles := h.getRoles(ctx)
	if h.hasAnyRole(roles, h.AdminRoles) || h.hasAnyRole(roles, h.ReadOnlyRoles) {
		return nil
	}
	return ErrForbidden("insufficient permissions to read " + resourceType)
}

// CanCreate checks if the authenticated user can create resources of this type.
func (h *RoleBasedAuthorizationHook) CanCreate(ctx context.Context, resourceType string) error {
	roles := h.getRoles(ctx)
	if h.hasAnyRole(roles, h.AdminRoles) {
		return nil
	}
	return ErrForbidden("insufficient permissions to create " + resourceType)
}

// CanUpdate checks if the authenticated user can update the resource.
func (h *RoleBasedAuthorizationHook) CanUpdate(ctx context.Context, resourceType, resourceID string) error {
	roles := h.getRoles(ctx)
	if h.hasAnyRole(roles, h.AdminRoles) {
		return nil
	}
	return ErrForbidden("insufficient permissions to update " + resourceType)
}

// CanDelete checks if the authenticated user can delete the resource.
func (h *RoleBasedAuthorizationHook) CanDelete(ctx context.Context, resourceType, resourceID string) error {
	roles := h.getRoles(ctx)
	if h.hasAnyRole(roles, h.AdminRoles) {
		return nil
	}
	return ErrForbidden("insufficient permissions to delete " + resourceType)
}

// getRoles extracts roles from the context.
func (h *RoleBasedAuthorizationHook) getRoles(ctx context.Context) []string {
	if h.RoleExtractor != nil {
		return h.RoleExtractor(ctx)
	}

	// Default: try to get roles from context value
	if roles, ok := ctx.Value(authRolesKey).([]string); ok {
		return roles
	}
	return nil
}

// hasAnyRole checks if any of the required roles is present.
func (h *RoleBasedAuthorizationHook) hasAnyRole(userRoles, required []string) bool {
	for _, ur := range userRoles {
		for _, rr := range required {
			if ur == rr {
				return true
			}
		}
	}
	return false
}

// Context key for roles - uses the same type as context.go
const authRolesKey contextKey = "scim_auth_roles"

// WithRoles adds roles to the context for role-based authorization.
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, authRolesKey, roles)
}

// RolesFromContext extracts roles from the context.
func RolesFromContext(ctx context.Context) []string {
	if roles, ok := ctx.Value(authRolesKey).([]string); ok {
		return roles
	}
	return nil
}

// CompositeAuthorizationHook combines multiple authorization hooks.
// All hooks must pass for the operation to be allowed.
type CompositeAuthorizationHook struct {
	hooks []AuthorizationHook
}

// NewCompositeAuthorizationHook creates a hook that combines multiple hooks.
func NewCompositeAuthorizationHook(hooks ...AuthorizationHook) *CompositeAuthorizationHook {
	return &CompositeAuthorizationHook{hooks: hooks}
}

// CanRead checks if all hooks allow reading the resource.
func (h *CompositeAuthorizationHook) CanRead(ctx context.Context, resourceType, resourceID string) error {
	for _, hook := range h.hooks {
		if err := hook.CanRead(ctx, resourceType, resourceID); err != nil {
			return err
		}
	}
	return nil
}

// CanCreate checks if all hooks allow creating the resource type.
func (h *CompositeAuthorizationHook) CanCreate(ctx context.Context, resourceType string) error {
	for _, hook := range h.hooks {
		if err := hook.CanCreate(ctx, resourceType); err != nil {
			return err
		}
	}
	return nil
}

// CanUpdate checks if all hooks allow updating the resource.
func (h *CompositeAuthorizationHook) CanUpdate(ctx context.Context, resourceType, resourceID string) error {
	for _, hook := range h.hooks {
		if err := hook.CanUpdate(ctx, resourceType, resourceID); err != nil {
			return err
		}
	}
	return nil
}

// CanDelete checks if all hooks allow deleting the resource.
func (h *CompositeAuthorizationHook) CanDelete(ctx context.Context, resourceType, resourceID string) error {
	for _, hook := range h.hooks {
		if err := hook.CanDelete(ctx, resourceType, resourceID); err != nil {
			return err
		}
	}
	return nil
}

// ValidateScopes validates that the provided scopes are known SCIM scopes.
func ValidateScopes(scopes []string) []string {
	validScopes := []string{
		ScopeUsersRead, ScopeUsersWrite,
		ScopeGroupsRead, ScopeGroupsWrite,
		ScopeFull,
	}

	var invalid []string
	for _, s := range scopes {
		found := false
		for _, vs := range validScopes {
			if s == vs {
				found = true
				break
			}
		}
		if !found && strings.HasPrefix(s, "scim:") {
			invalid = append(invalid, s)
		}
	}
	return invalid
}

// ParseScopes parses a space-separated scope string into a slice.
func ParseScopes(scopeString string) []string {
	if scopeString == "" {
		return nil
	}
	return strings.Fields(scopeString)
}

// ScopesString converts a scope slice to a space-separated string.
func ScopesString(scopes []string) string {
	return strings.Join(scopes, " ")
}
