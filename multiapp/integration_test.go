package multiapp

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/grokify/coreforge/rls"
	"github.com/grokify/coreforge/session/jwt"
	"github.com/grokify/coreforge/session/middleware"
	"github.com/stretchr/testify/assert"
)

func TestFullContextFromContext(t *testing.T) {
	ctx := context.Background()

	// Empty context should have nil values
	fc := FullContextFromContext(ctx)
	assert.NotNil(t, fc)
	assert.Nil(t, fc.App)
	assert.Nil(t, fc.Claims)
	assert.Equal(t, uuid.Nil, fc.TenantID)
	assert.Equal(t, uuid.Nil, fc.UserID)
	assert.False(t, fc.IsAuthenticated())
	assert.False(t, fc.HasApp())
	assert.False(t, fc.IsFederated())
}

func TestFullContextWithAllContexts(t *testing.T) {
	ctx := context.Background()

	// Add app context
	appCtx := &AppContext{
		AppID:          "app1",
		AppSlug:        "app1",
		AppName:        "App1",
		DatabaseSchema: "app_app1",
	}
	ctx = WithAppContext(ctx, appCtx)

	// Add JWT claims
	principalID := uuid.New()
	orgID := uuid.New()
	claims := &jwt.Claims{
		PrincipalID:    principalID,
		OrganizationID: &orgID,
		Role:           "admin",
	}
	ctx = middleware.ContextWithClaims(ctx, claims)

	// Add RLS context
	tenantID := uuid.New()
	userID := uuid.New()
	ctx = rls.ContextWithTenantAndUser(ctx, tenantID, userID)

	// Verify full context
	fc := FullContextFromContext(ctx)
	assert.NotNil(t, fc)
	assert.True(t, fc.HasApp())
	assert.True(t, fc.IsAuthenticated())
	assert.Equal(t, "app1", fc.App.AppID)
	assert.Equal(t, principalID, fc.PrincipalID())
	assert.Equal(t, &orgID, fc.OrganizationID())
	assert.Equal(t, tenantID, fc.TenantID)
	assert.Equal(t, userID, fc.UserID)
}

func TestWithRLSFromClaims(t *testing.T) {
	ctx := context.Background()

	// Add JWT claims with org context
	principalID := uuid.New()
	orgID := uuid.New()
	claims := &jwt.Claims{
		PrincipalID:    principalID,
		OrganizationID: &orgID,
	}
	ctx = middleware.ContextWithClaims(ctx, claims)

	// Apply RLS from claims
	ctx = WithRLSFromClaims(ctx)

	// Verify RLS context was set
	assert.Equal(t, principalID, rls.UserIDFromContext(ctx))
	assert.Equal(t, orgID, rls.TenantIDFromContext(ctx))
}

func TestRequireAuth(t *testing.T) {
	ctx := context.Background()

	// Without auth
	_, err := RequireAuth(ctx)
	assert.ErrorIs(t, err, ErrNotAuthenticated)

	// With auth
	principalID := uuid.New()
	claims := &jwt.Claims{PrincipalID: principalID}
	ctx = middleware.ContextWithClaims(ctx, claims)

	id, err := RequireAuth(ctx)
	assert.NoError(t, err)
	assert.Equal(t, principalID, id)
}

func TestRequireAppAndAuth(t *testing.T) {
	ctx := context.Background()

	// Without app or auth
	_, err := RequireAppAndAuth(ctx)
	assert.ErrorIs(t, err, ErrNoAppContext)

	// With app but no auth
	appCtx := &AppContext{AppID: "test"}
	ctx = WithAppContext(ctx, appCtx)
	_, err = RequireAppAndAuth(ctx)
	assert.ErrorIs(t, err, ErrNotAuthenticated)

	// With app and auth
	claims := &jwt.Claims{PrincipalID: uuid.New()}
	ctx = middleware.ContextWithClaims(ctx, claims)

	fc, err := RequireAppAndAuth(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, fc)
	assert.True(t, fc.HasApp())
	assert.True(t, fc.IsAuthenticated())
}

func TestValidateAppFeature(t *testing.T) {
	ctx := context.Background()

	// Without app context
	err := ValidateAppFeature(ctx, "auth")
	assert.ErrorIs(t, err, ErrFeatureNotEnabled)

	// With app but missing feature
	appCtx := &AppContext{
		AppID:    "test",
		Features: []string{"tenancy"},
	}
	ctx = WithAppContext(ctx, appCtx)
	err = ValidateAppFeature(ctx, "auth")
	assert.ErrorIs(t, err, ErrFeatureNotEnabled)

	// With app and feature
	appCtx.Features = append(appCtx.Features, "auth")
	err = ValidateAppFeature(ctx, "auth")
	assert.NoError(t, err)
}
