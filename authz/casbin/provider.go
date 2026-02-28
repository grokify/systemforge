// Package casbin provides a Casbin-based authorization provider.
//
// This provider uses Casbin for flexible RBAC/ABAC policy evaluation.
// It supports both code-defined policies and database-stored policies.
//
// Example usage:
//
//	provider, err := casbin.New(
//	    casbin.WithModel(casbin.RBACModel),
//	    casbin.WithRoleGetter(getRoleFromDB),
//	)
package casbin

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/grokify/coreforge/authz"
)

// Pre-defined Casbin models for common use cases.
const (
	// RBACModel is a basic RBAC model with role hierarchy.
	RBACModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

	// RBACWithResourceRolesModel supports resource-specific roles.
	RBACWithResourceRolesModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _
g2 = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && g2(r.obj, p.obj) && r.act == p.act
`

	// ABACModel supports attribute-based access control.
	ABACModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`
)

// Provider implements the Authorizer interface using Casbin.
type Provider struct {
	enforcer *casbin.Enforcer

	// Callbacks for looking up data
	getRoleFn       func(ctx context.Context, principalID, orgID uuid.UUID) (string, error)
	isPlatformAdmin func(ctx context.Context, principalID uuid.UUID) (bool, error)

	// Options
	allowOwnerFullAccess bool
	platformAdminBypass  bool
}

// Option configures an Provider.
type Option func(*Provider) error

// WithModel sets the Casbin model from a string.
func WithModel(modelText string) Option {
	return func(a *Provider) error {
		m, err := model.NewModelFromString(modelText)
		if err != nil {
			return fmt.Errorf("parsing model: %w", err)
		}
		e, err := casbin.NewEnforcer(m)
		if err != nil {
			return fmt.Errorf("creating enforcer: %w", err)
		}
		a.enforcer = e
		return nil
	}
}

// WithEnforcer sets a pre-configured Casbin enforcer.
func WithEnforcer(e *casbin.Enforcer) Option {
	return func(a *Provider) error {
		a.enforcer = e
		return nil
	}
}

// WithRoleGetter sets the function to retrieve a principal's role in an organization.
func WithRoleGetter(fn func(ctx context.Context, principalID, orgID uuid.UUID) (string, error)) Option {
	return func(a *Provider) error {
		a.getRoleFn = fn
		return nil
	}
}

// WithPlatformAdminChecker sets the function to check platform admin status.
func WithPlatformAdminChecker(fn func(ctx context.Context, principalID uuid.UUID) (bool, error)) Option {
	return func(a *Provider) error {
		a.isPlatformAdmin = fn
		return nil
	}
}

// WithOwnerFullAccess enables full access for resource owners.
func WithOwnerFullAccess(enabled bool) Option {
	return func(a *Provider) error {
		a.allowOwnerFullAccess = enabled
		return nil
	}
}

// WithPlatformAdminBypass enables platform admin bypass for all checks.
func WithPlatformAdminBypass(enabled bool) Option {
	return func(a *Provider) error {
		a.platformAdminBypass = enabled
		return nil
	}
}

// New creates a new Casbin Provider with the given options.
func New(opts ...Option) (*Provider, error) {
	a := &Provider{
		allowOwnerFullAccess: true,
		platformAdminBypass:  true,
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, err
		}
	}

	// If no enforcer was provided, create a default RBAC one
	if a.enforcer == nil {
		m, err := model.NewModelFromString(RBACModel)
		if err != nil {
			return nil, fmt.Errorf("parsing default model: %w", err)
		}
		e, err := casbin.NewEnforcer(m)
		if err != nil {
			return nil, fmt.Errorf("creating default enforcer: %w", err)
		}
		a.enforcer = e
	}

	return a, nil
}

// Enforcer returns the underlying Casbin enforcer for advanced configuration.
func (a *Provider) Enforcer() *casbin.Enforcer {
	return a.enforcer
}

// AddPolicy adds a policy rule.
func (a *Provider) AddPolicy(role, resource, action string) error {
	_, err := a.enforcer.AddPolicy(role, resource, action)
	return err
}

// AddRoleForUser assigns a role to a user.
func (a *Provider) AddRoleForUser(user, role string) error {
	_, err := a.enforcer.AddGroupingPolicy(user, role)
	return err
}

// AddPolicies adds multiple policy rules at once.
func (a *Provider) AddPolicies(policies [][]string) error {
	_, err := a.enforcer.AddPolicies(policies)
	return err
}

// LoadPoliciesFromMap loads policies from a role-permission map.
func (a *Provider) LoadPoliciesFromMap(permissions authz.RolePermissions) error {
	for role, perms := range permissions {
		for _, perm := range perms {
			// Split permission into resource.action format
			parts := strings.SplitN(perm, ".", 2)
			if len(parts) == 2 {
				if err := a.AddPolicy(role, parts[0], parts[1]); err != nil {
					return err
				}
			} else {
				// If no dot, treat whole thing as resource with wildcard action
				if err := a.AddPolicy(role, perm, "*"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Can checks if a principal can perform an action on a resource.
func (a *Provider) Can(ctx context.Context, principal authz.Principal, action authz.Action, resource authz.Resource) (bool, error) {
	// Check platform admin bypass
	if a.platformAdminBypass && a.isPlatformAdmin != nil {
		isAdmin, err := a.isPlatformAdmin(ctx, principal.ID)
		if err != nil {
			return false, fmt.Errorf("checking platform admin: %w", err)
		}
		if isAdmin {
			return true, nil
		}
	}

	// Check owner access
	if a.allowOwnerFullAccess && resource.IsOwner(principal.ID) {
		return true, nil
	}

	// Determine subject (role or principal ID)
	subject := principal.ID.String()
	if resource.OrgID != nil && a.getRoleFn != nil {
		role, err := a.getRoleFn(ctx, principal.ID, *resource.OrgID)
		if err != nil {
			return false, fmt.Errorf("getting role: %w", err)
		}
		if role == "" {
			return false, nil // Not a member
		}
		subject = role
	}

	// Build resource string
	resourceStr := strings.ToLower(string(resource.Type))

	// Enforce using Casbin
	allowed, err := a.enforcer.Enforce(subject, resourceStr, strings.ToLower(string(action)))
	if err != nil {
		return false, fmt.Errorf("enforcing policy: %w", err)
	}

	return allowed, nil
}

// CanAll checks if a principal can perform all specified actions on a resource.
func (a *Provider) CanAll(ctx context.Context, principal authz.Principal, actions []authz.Action, resource authz.Resource) (bool, error) {
	for _, action := range actions {
		allowed, err := a.Can(ctx, principal, action, resource)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

// CanAny checks if a principal can perform any of the specified actions on a resource.
func (a *Provider) CanAny(ctx context.Context, principal authz.Principal, actions []authz.Action, resource authz.Resource) (bool, error) {
	for _, action := range actions {
		allowed, err := a.Can(ctx, principal, action, resource)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

// Filter returns only the resources the principal can access with the given action.
func (a *Provider) Filter(ctx context.Context, principal authz.Principal, action authz.Action, resources []authz.Resource) ([]authz.Resource, error) {
	var allowed []authz.Resource
	for _, resource := range resources {
		can, err := a.Can(ctx, principal, action, resource)
		if err != nil {
			return nil, err
		}
		if can {
			allowed = append(allowed, resource)
		}
	}
	return allowed, nil
}

// CanForOrg checks permission scoped to a specific organization.
func (a *Provider) CanForOrg(ctx context.Context, principal authz.Principal, orgID uuid.UUID, action authz.Action, resource authz.Resource) (bool, error) {
	resourceWithOrg := resource.WithOrg(orgID)
	return a.Can(ctx, principal, action, resourceWithOrg)
}

// GetRole returns the principal's role in an organization.
func (a *Provider) GetRole(ctx context.Context, principal authz.Principal, orgID uuid.UUID) (string, error) {
	if a.getRoleFn == nil {
		return "", nil
	}
	return a.getRoleFn(ctx, principal.ID, orgID)
}

// IsMember checks if a principal is a member of an organization.
func (a *Provider) IsMember(ctx context.Context, principal authz.Principal, orgID uuid.UUID) (bool, error) {
	role, err := a.GetRole(ctx, principal, orgID)
	if err != nil {
		return false, err
	}
	return role != "", nil
}

// IsPlatformAdmin checks if a principal has platform-wide admin access.
func (a *Provider) IsPlatformAdmin(ctx context.Context, principal authz.Principal) (bool, error) {
	if a.isPlatformAdmin == nil {
		return false, nil
	}
	return a.isPlatformAdmin(ctx, principal.ID)
}

// Verify Provider implements the required interfaces.
var (
	_ authz.Authorizer         = (*Provider)(nil)
	_ authz.OrgAuthorizer      = (*Provider)(nil)
	_ authz.PlatformAuthorizer = (*Provider)(nil)
)
