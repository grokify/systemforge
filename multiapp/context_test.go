package multiapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppContext(t *testing.T) {
	ctx := context.Background()

	// Initially no app context
	assert.Nil(t, AppContextFromContext(ctx))
	assert.False(t, HasAppContext(ctx))
	assert.Empty(t, AppSlugFromContext(ctx))
	assert.Empty(t, AppIDFromContext(ctx))
	assert.Empty(t, DatabaseSchemaFromContext(ctx))

	// Add app context
	appCtx := &AppContext{
		AppID:          "app1",
		AppSlug:        "app1",
		AppName:        "App1",
		DatabaseSchema: "app_app1",
		Features:       []string{"auth", "tenancy"},
		Settings:       map[string]any{"max_users": 1000},
	}
	ctx = WithAppContext(ctx, appCtx)

	// Now app context is present
	assert.True(t, HasAppContext(ctx))
	assert.Equal(t, "app1", AppSlugFromContext(ctx))
	assert.Equal(t, "app1", AppIDFromContext(ctx))
	assert.Equal(t, "app_app1", DatabaseSchemaFromContext(ctx))

	// Test features
	assert.True(t, HasFeature(ctx, "auth"))
	assert.True(t, HasFeature(ctx, "tenancy"))
	assert.False(t, HasFeature(ctx, "analytics"))

	// Test settings
	assert.Equal(t, 1000, GetSetting(ctx, "max_users"))
	assert.Nil(t, GetSetting(ctx, "nonexistent"))

	// Verify full context retrieval
	retrieved := AppContextFromContext(ctx)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "app1", retrieved.AppID)
	assert.Equal(t, "App1", retrieved.AppName)
}

func TestAppContextWithNilSettings(t *testing.T) {
	ctx := context.Background()
	appCtx := &AppContext{
		AppID:   "test",
		AppSlug: "test",
	}
	ctx = WithAppContext(ctx, appCtx)

	// Should not panic with nil settings
	assert.Nil(t, GetSetting(ctx, "anything"))
}

func TestHasFeatureWithoutContext(t *testing.T) {
	ctx := context.Background()
	// Should not panic without context
	assert.False(t, HasFeature(ctx, "anything"))
}
