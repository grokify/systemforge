package scim

import (
	"context"
	"testing"
)

func TestScopedAuthorizationHook_CanRead(t *testing.T) {
	tests := []struct {
		name         string
		require      bool
		scopes       []string
		resourceType string
		wantErr      bool
	}{
		{
			name:         "no enforcement",
			require:      false,
			scopes:       nil,
			resourceType: ResourceTypeUser,
			wantErr:      false,
		},
		{
			name:         "has read scope",
			require:      true,
			scopes:       []string{ScopeUsersRead},
			resourceType: ResourceTypeUser,
			wantErr:      false,
		},
		{
			name:         "has write scope (not read)",
			require:      true,
			scopes:       []string{ScopeUsersWrite},
			resourceType: ResourceTypeUser,
			wantErr:      true,
		},
		{
			name:         "has full scope",
			require:      true,
			scopes:       []string{ScopeFull},
			resourceType: ResourceTypeUser,
			wantErr:      false,
		},
		{
			name:         "no scopes",
			require:      true,
			scopes:       nil,
			resourceType: ResourceTypeUser,
			wantErr:      true,
		},
		{
			name:         "wrong resource type scope",
			require:      true,
			scopes:       []string{ScopeGroupsRead},
			resourceType: ResourceTypeUser,
			wantErr:      true,
		},
		{
			name:         "group read scope for groups",
			require:      true,
			scopes:       []string{ScopeGroupsRead},
			resourceType: ResourceTypeGroup,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hook := NewScopedAuthorizationHook(tc.require)
			ctx := WithAuthScopes(context.Background(), tc.scopes)

			err := hook.CanRead(ctx, tc.resourceType, "resource-123")
			if (err != nil) != tc.wantErr {
				t.Errorf("CanRead() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestScopedAuthorizationHook_CanCreate(t *testing.T) {
	tests := []struct {
		name         string
		scopes       []string
		resourceType string
		wantErr      bool
	}{
		{
			name:         "has write scope",
			scopes:       []string{ScopeUsersWrite},
			resourceType: ResourceTypeUser,
			wantErr:      false,
		},
		{
			name:         "has read scope only",
			scopes:       []string{ScopeUsersRead},
			resourceType: ResourceTypeUser,
			wantErr:      true,
		},
		{
			name:         "has full scope",
			scopes:       []string{ScopeFull},
			resourceType: ResourceTypeUser,
			wantErr:      false,
		},
		{
			name:         "group write scope for groups",
			scopes:       []string{ScopeGroupsWrite},
			resourceType: ResourceTypeGroup,
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hook := NewScopedAuthorizationHook(true)
			ctx := WithAuthScopes(context.Background(), tc.scopes)

			err := hook.CanCreate(ctx, tc.resourceType)
			if (err != nil) != tc.wantErr {
				t.Errorf("CanCreate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestScopedAuthorizationHook_CanUpdate(t *testing.T) {
	hook := NewScopedAuthorizationHook(true)

	// With write scope
	ctx := WithAuthScopes(context.Background(), []string{ScopeUsersWrite})
	if err := hook.CanUpdate(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanUpdate() with write scope should succeed, got error: %v", err)
	}

	// Without write scope
	ctx = WithAuthScopes(context.Background(), []string{ScopeUsersRead})
	if err := hook.CanUpdate(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanUpdate() without write scope should fail")
	}
}

func TestScopedAuthorizationHook_CanDelete(t *testing.T) {
	hook := NewScopedAuthorizationHook(true)

	// With write scope
	ctx := WithAuthScopes(context.Background(), []string{ScopeUsersWrite})
	if err := hook.CanDelete(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanDelete() with write scope should succeed, got error: %v", err)
	}

	// Without write scope
	ctx = WithAuthScopes(context.Background(), []string{ScopeUsersRead})
	if err := hook.CanDelete(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanDelete() without write scope should fail")
	}
}

func TestRoleBasedAuthorizationHook_CanRead(t *testing.T) {
	hook := NewRoleBasedAuthorizationHook(
		[]string{"admin", "superadmin"},
		[]string{"viewer", "auditor"},
	)

	tests := []struct {
		name    string
		roles   []string
		wantErr bool
	}{
		{
			name:    "admin role",
			roles:   []string{"admin"},
			wantErr: false,
		},
		{
			name:    "viewer role",
			roles:   []string{"viewer"},
			wantErr: false,
		},
		{
			name:    "no roles",
			roles:   nil,
			wantErr: true,
		},
		{
			name:    "unknown role",
			roles:   []string{"guest"},
			wantErr: true,
		},
		{
			name:    "multiple roles including admin",
			roles:   []string{"guest", "admin"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := WithRoles(context.Background(), tc.roles)
			err := hook.CanRead(ctx, ResourceTypeUser, "user-123")
			if (err != nil) != tc.wantErr {
				t.Errorf("CanRead() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRoleBasedAuthorizationHook_CanCreate(t *testing.T) {
	hook := NewRoleBasedAuthorizationHook(
		[]string{"admin"},
		[]string{"viewer"},
	)

	// Admin can create
	ctx := WithRoles(context.Background(), []string{"admin"})
	if err := hook.CanCreate(ctx, ResourceTypeUser); err != nil {
		t.Errorf("CanCreate() with admin role should succeed, got error: %v", err)
	}

	// Viewer cannot create
	ctx = WithRoles(context.Background(), []string{"viewer"})
	if err := hook.CanCreate(ctx, ResourceTypeUser); err == nil {
		t.Error("CanCreate() with viewer role should fail")
	}
}

func TestRoleBasedAuthorizationHook_CanUpdate(t *testing.T) {
	hook := NewRoleBasedAuthorizationHook(
		[]string{"admin"},
		[]string{"viewer"},
	)

	// Admin can update
	ctx := WithRoles(context.Background(), []string{"admin"})
	if err := hook.CanUpdate(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanUpdate() with admin role should succeed, got error: %v", err)
	}

	// Viewer cannot update
	ctx = WithRoles(context.Background(), []string{"viewer"})
	if err := hook.CanUpdate(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanUpdate() with viewer role should fail")
	}
}

func TestRoleBasedAuthorizationHook_CanDelete(t *testing.T) {
	hook := NewRoleBasedAuthorizationHook(
		[]string{"admin"},
		[]string{"viewer"},
	)

	// Admin can delete
	ctx := WithRoles(context.Background(), []string{"admin"})
	if err := hook.CanDelete(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanDelete() with admin role should succeed, got error: %v", err)
	}

	// Viewer cannot delete
	ctx = WithRoles(context.Background(), []string{"viewer"})
	if err := hook.CanDelete(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanDelete() with viewer role should fail")
	}
}

func TestRoleBasedAuthorizationHook_CustomRoleExtractor(t *testing.T) {
	hook := &RoleBasedAuthorizationHook{
		AdminRoles: []string{"admin"},
		RoleExtractor: func(ctx context.Context) []string {
			return []string{"admin"} // Always return admin
		},
	}

	// Should succeed because custom extractor returns admin
	ctx := context.Background()
	if err := hook.CanCreate(ctx, ResourceTypeUser); err != nil {
		t.Errorf("CanCreate() with custom extractor returning admin should succeed, got error: %v", err)
	}
}

func TestCompositeAuthorizationHook(t *testing.T) {
	scopeHook := NewScopedAuthorizationHook(true)
	roleHook := NewRoleBasedAuthorizationHook([]string{"admin"}, nil)
	composite := NewCompositeAuthorizationHook(scopeHook, roleHook)

	// Both pass
	ctx := WithAuthScopes(context.Background(), []string{ScopeUsersRead})
	ctx = WithRoles(ctx, []string{"admin"})
	if err := composite.CanRead(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanRead() with both hooks passing should succeed, got error: %v", err)
	}

	// Scope fails
	ctx = WithAuthScopes(context.Background(), []string{ScopeGroupsRead})
	ctx = WithRoles(ctx, []string{"admin"})
	if err := composite.CanRead(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanRead() should fail when scope hook fails")
	}

	// Role fails
	ctx = WithAuthScopes(context.Background(), []string{ScopeUsersRead})
	ctx = WithRoles(ctx, []string{"viewer"})
	if err := composite.CanRead(ctx, ResourceTypeUser, "user-123"); err == nil {
		t.Error("CanRead() should fail when role hook fails")
	}
}

func TestCompositeAuthorizationHook_AllOperations(t *testing.T) {
	scopeHook := NewScopedAuthorizationHook(true)
	roleHook := NewRoleBasedAuthorizationHook([]string{"admin"}, nil)
	composite := NewCompositeAuthorizationHook(scopeHook, roleHook)

	ctx := WithAuthScopes(context.Background(), []string{ScopeUsersWrite})
	ctx = WithRoles(ctx, []string{"admin"})

	if err := composite.CanCreate(ctx, ResourceTypeUser); err != nil {
		t.Errorf("CanCreate() should succeed, got error: %v", err)
	}

	if err := composite.CanUpdate(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanUpdate() should succeed, got error: %v", err)
	}

	if err := composite.CanDelete(ctx, ResourceTypeUser, "user-123"); err != nil {
		t.Errorf("CanDelete() should succeed, got error: %v", err)
	}
}

func TestWithRoles(t *testing.T) {
	ctx := context.Background()
	roles := []string{"admin", "viewer"}

	ctx = WithRoles(ctx, roles)
	got := RolesFromContext(ctx)

	if len(got) != len(roles) {
		t.Errorf("RolesFromContext() returned %d roles, want %d", len(got), len(roles))
	}

	for i, role := range roles {
		if got[i] != role {
			t.Errorf("RolesFromContext()[%d] = %q, want %q", i, got[i], role)
		}
	}
}

func TestRolesFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := RolesFromContext(ctx)
	if got != nil {
		t.Errorf("RolesFromContext() = %v, want nil", got)
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name        string
		scopes      []string
		wantInvalid []string
	}{
		{
			name:        "all valid",
			scopes:      []string{ScopeUsersRead, ScopeUsersWrite, ScopeGroupsRead},
			wantInvalid: nil,
		},
		{
			name:        "some invalid",
			scopes:      []string{ScopeUsersRead, "scim:invalid:scope"},
			wantInvalid: []string{"scim:invalid:scope"},
		},
		{
			name:        "non-scim scopes ignored",
			scopes:      []string{"openid", "profile", ScopeUsersRead},
			wantInvalid: nil,
		},
		{
			name:        "full scope valid",
			scopes:      []string{ScopeFull},
			wantInvalid: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateScopes(tc.scopes)
			if len(got) != len(tc.wantInvalid) {
				t.Errorf("ValidateScopes() = %v, want %v", got, tc.wantInvalid)
			}
		})
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"scim:users:read scim:users:write", []string{"scim:users:read", "scim:users:write"}},
		{"scim:full", []string{"scim:full"}},
		{"", nil},
		{"  scim:users:read   scim:groups:read  ", []string{"scim:users:read", "scim:groups:read"}},
	}

	for _, tc := range tests {
		got := ParseScopes(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("ParseScopes(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("ParseScopes(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestScopesString(t *testing.T) {
	tests := []struct {
		scopes []string
		want   string
	}{
		{[]string{"scim:users:read", "scim:users:write"}, "scim:users:read scim:users:write"},
		{[]string{"scim:full"}, "scim:full"},
		{nil, ""},
		{[]string{}, ""},
	}

	for _, tc := range tests {
		got := ScopesString(tc.scopes)
		if got != tc.want {
			t.Errorf("ScopesString(%v) = %q, want %q", tc.scopes, got, tc.want)
		}
	}
}
