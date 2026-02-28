package casbin_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/authz"
	"github.com/grokify/coreforge/authz/casbin"
	"github.com/grokify/coreforge/authz/providertest"
)

// TestConformance runs the full conformance test suite for the casbin provider.
func TestConformance(t *testing.T) {
	principalID := uuid.New()
	orgID := uuid.New()

	// In-memory role storage for testing
	roles := map[string]map[string]string{
		principalID.String(): {
			orgID.String(): "admin",
		},
	}
	platformAdmins := map[string]bool{}

	provider, err := casbin.New(
		casbin.WithModel(casbin.RBACModel),
		casbin.WithRoleGetter(func(ctx context.Context, pid, oid uuid.UUID) (string, error) {
			if orgRoles, ok := roles[pid.String()]; ok {
				return orgRoles[oid.String()], nil
			}
			return "", nil
		}),
		casbin.WithPlatformAdminChecker(func(ctx context.Context, pid uuid.UUID) (bool, error) {
			return platformAdmins[pid.String()], nil
		}),
		casbin.WithOwnerFullAccess(true),
		casbin.WithPlatformAdminBypass(true),
	)
	if err != nil {
		t.Fatalf("Failed to create casbin provider: %v", err)
	}

	// Load default permissions into Casbin
	if err := provider.LoadPoliciesFromMap(authz.DefaultRolePermissions); err != nil {
		t.Fatalf("Failed to load policies: %v", err)
	}

	providertest.RunAll(t, providertest.Config{
		Provider:        provider,
		SkipIntegration: false,
		TestPrincipalID: principalID,
		TestOrgID:       orgID,
	})
}

// TestInterfaceConformance verifies casbin.Provider implements all expected interfaces.
func TestInterfaceConformance(t *testing.T) {
	provider, err := casbin.New()
	if err != nil {
		t.Fatalf("Failed to create casbin provider: %v", err)
	}

	// Verify interface implementation via type assertions
	var _ authz.Authorizer = provider
	var _ authz.OrgAuthorizer = provider
	var _ authz.PlatformAuthorizer = provider
}

// TestWithRBACModel tests the RBAC model configuration.
func TestWithRBACModel(t *testing.T) {
	provider, err := casbin.New(casbin.WithModel(casbin.RBACModel))
	if err != nil {
		t.Fatalf("WithModel(RBACModel) error: %v", err)
	}

	// Add a policy and verify it works
	if err := provider.AddPolicy("admin", "resource", "read"); err != nil {
		t.Fatalf("AddPolicy error: %v", err)
	}

	enforcer := provider.Enforcer()
	allowed, err := enforcer.Enforce("admin", "resource", "read")
	if err != nil {
		t.Fatalf("Enforce error: %v", err)
	}
	if !allowed {
		t.Error("Policy should allow admin to read resource")
	}
}

// TestWithABACModel tests the ABAC model configuration.
func TestWithABACModel(t *testing.T) {
	provider, err := casbin.New(casbin.WithModel(casbin.ABACModel))
	if err != nil {
		t.Fatalf("WithModel(ABACModel) error: %v", err)
	}

	// Verify enforcer was created
	if provider.Enforcer() == nil {
		t.Error("Enforcer should not be nil")
	}
}

// TestLoadPoliciesFromMap tests loading policies from RolePermissions map.
func TestLoadPoliciesFromMap(t *testing.T) {
	provider, err := casbin.New(casbin.WithModel(casbin.RBACModel))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	permissions := authz.RolePermissions{
		"editor": {"document.create", "document.read", "document.update"},
		"viewer": {"document.read"},
	}

	if err := provider.LoadPoliciesFromMap(permissions); err != nil {
		t.Fatalf("LoadPoliciesFromMap error: %v", err)
	}

	enforcer := provider.Enforcer()

	// Editor should have create permission
	allowed, _ := enforcer.Enforce("editor", "document", "create")
	if !allowed {
		t.Error("Editor should have document.create permission")
	}

	// Viewer should have read permission
	allowed, _ = enforcer.Enforce("viewer", "document", "read")
	if !allowed {
		t.Error("Viewer should have document.read permission")
	}

	// Viewer should NOT have create permission
	allowed, _ = enforcer.Enforce("viewer", "document", "create")
	if allowed {
		t.Error("Viewer should NOT have document.create permission")
	}
}

// TestAddRoleForUser tests role assignment to users.
func TestAddRoleForUser(t *testing.T) {
	provider, err := casbin.New(casbin.WithModel(casbin.RBACModel))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Add policy for admin role
	if err := provider.AddPolicy("admin", "resource", "delete"); err != nil {
		t.Fatalf("AddPolicy error: %v", err)
	}

	// Assign admin role to user
	userID := uuid.New().String()
	if err := provider.AddRoleForUser(userID, "admin"); err != nil {
		t.Fatalf("AddRoleForUser error: %v", err)
	}

	// User should now have delete permission through admin role
	enforcer := provider.Enforcer()
	allowed, _ := enforcer.Enforce(userID, "resource", "delete")
	if !allowed {
		t.Error("User with admin role should have delete permission")
	}
}

// TestOwnerBypass tests owner full access bypass.
func TestOwnerBypass(t *testing.T) {
	ownerID := uuid.New()
	resourceID := uuid.New()

	provider, err := casbin.New(
		casbin.WithOwnerFullAccess(true),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	principal := authz.NewUserPrincipal(ownerID)
	resource := authz.NewOwnedResource(authz.ResourceType("document"), resourceID, ownerID)

	// Owner should have delete access even without policies
	allowed, err := provider.Can(context.Background(), principal, authz.ActionDelete, resource)
	if err != nil {
		t.Fatalf("Can(delete) error: %v", err)
	}
	if !allowed {
		t.Error("Owner should have full access to their resource")
	}
}

// TestPlatformAdminBypass tests platform admin bypass.
func TestPlatformAdminBypass(t *testing.T) {
	adminID := uuid.New()
	orgID := uuid.New()

	provider, err := casbin.New(
		casbin.WithPlatformAdminBypass(true),
		casbin.WithPlatformAdminChecker(func(ctx context.Context, pid uuid.UUID) (bool, error) {
			return pid == adminID, nil
		}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	principal := authz.NewUserPrincipal(adminID)
	resource := authz.NewOrgResource(authz.ResourceType("resource"), orgID)

	// Platform admin should have access without any policies
	allowed, err := provider.Can(context.Background(), principal, authz.ActionDelete, resource)
	if err != nil {
		t.Fatalf("Can(delete) error: %v", err)
	}
	if !allowed {
		t.Error("Platform admin should bypass permission checks")
	}
}

// TestNonMemberDenied tests that non-members are denied access.
func TestNonMemberDenied(t *testing.T) {
	nonMemberID := uuid.New()
	orgID := uuid.New()

	provider, err := casbin.New(
		casbin.WithRoleGetter(func(ctx context.Context, pid, oid uuid.UUID) (string, error) {
			return "", nil // No role = not a member
		}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	principal := authz.NewUserPrincipal(nonMemberID)
	resource := authz.NewOrgResource(authz.ResourceType("resource"), orgID)

	allowed, err := provider.Can(context.Background(), principal, authz.ActionRead, resource)
	if err != nil {
		t.Fatalf("Can(read) error: %v", err)
	}
	if allowed {
		t.Error("Non-member should be denied access")
	}
}

// TestRoleBasedAccess tests role-based permission checking.
func TestRoleBasedAccess(t *testing.T) {
	principalID := uuid.New()
	orgID := uuid.New()

	provider, err := casbin.New(
		casbin.WithModel(casbin.RBACModel),
		casbin.WithRoleGetter(func(ctx context.Context, pid, oid uuid.UUID) (string, error) {
			if pid == principalID && oid == orgID {
				return "editor", nil
			}
			return "", nil
		}),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Add policies for editor role
	_ = provider.AddPolicy("editor", "resource", "read")
	_ = provider.AddPolicy("editor", "resource", "update")
	// Editor does NOT have delete

	principal := authz.NewUserPrincipal(principalID)
	resource := authz.NewOrgResource(authz.ResourceType("resource"), orgID)

	// Editor should have read access
	allowed, err := provider.Can(context.Background(), principal, authz.ActionRead, resource)
	if err != nil {
		t.Fatalf("Can(read) error: %v", err)
	}
	if !allowed {
		t.Error("Editor should have read permission")
	}

	// Editor should NOT have delete access
	allowed, err = provider.Can(context.Background(), principal, authz.ActionDelete, resource)
	if err != nil {
		t.Fatalf("Can(delete) error: %v", err)
	}
	if allowed {
		t.Error("Editor should NOT have delete permission")
	}
}
