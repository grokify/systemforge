package stores

import (
	"context"
	"testing"

	"github.com/grokify/coreforge/featureflags"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Test Get non-existent
	flag, err := store.Get(ctx, "nonexistent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if flag != nil {
		t.Error("expected nil for nonexistent flag")
	}

	// Test Set
	testFlag := &featureflags.Flag{
		Key:     "test-feature",
		Name:    "Test Feature",
		Enabled: true,
	}
	err = store.Set(ctx, testFlag)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get existing
	flag, err = store.Get(ctx, "test-feature")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if flag == nil {
		t.Fatal("expected flag to exist")
	}
	if flag.Key != "test-feature" {
		t.Errorf("expected key test-feature, got %s", flag.Key)
	}
	if flag.Name != "Test Feature" {
		t.Errorf("expected name Test Feature, got %s", flag.Name)
	}
	if !flag.Enabled {
		t.Error("expected enabled to be true")
	}

	// Test immutability (modifying returned flag shouldn't affect store)
	flag.Enabled = false
	flag2, _ := store.Get(ctx, "test-feature")
	if !flag2.Enabled {
		t.Error("store should return copies, not references")
	}
}

func TestMemoryStoreGetAll(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create multiple flags
	for _, key := range []string{"flag-1", "flag-2", "flag-3"} {
		err := store.Set(ctx, &featureflags.Flag{Key: key, Enabled: true})
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	flags, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(flags))
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create flag
	err := store.Set(ctx, &featureflags.Flag{Key: "deletable", Enabled: true})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete flag
	err = store.Delete(ctx, "deletable")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	flag, err := store.Get(ctx, "deletable")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if flag != nil {
		t.Error("expected flag to be deleted")
	}
}

func TestMemoryStorePreload(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	flags := []*featureflags.Flag{
		{Key: "flag-a", Enabled: true},
		{Key: "flag-b", Enabled: false},
	}

	store.Preload(flags)

	flag, err := store.Get(ctx, "flag-a")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if flag == nil || !flag.Enabled {
		t.Error("expected flag-a to be enabled")
	}

	flag, err = store.Get(ctx, "flag-b")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if flag == nil || flag.Enabled {
		t.Error("expected flag-b to be disabled")
	}
}

func TestMemoryStoreInvalidFlag(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Test nil flag
	err := store.Set(ctx, nil)
	if err == nil {
		t.Error("expected error for nil flag")
	}

	// Test empty key
	err = store.Set(ctx, &featureflags.Flag{Key: ""})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestFeatureFlagEngine(t *testing.T) {
	store := NewMemoryStore()
	engine := featureflags.NewEngine(store)
	ctx := context.Background()

	// Test IsEnabled for nonexistent flag
	enabled, err := engine.IsEnabled(ctx, "nonexistent", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected disabled for nonexistent flag")
	}

	// Test SetFlag and IsEnabled
	err = engine.SetFlag(ctx, &featureflags.Flag{
		Key:     "feature-a",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	enabled, err = engine.IsEnabled(ctx, "feature-a", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected enabled for feature-a")
	}
}

func TestFeatureFlagTargeting(t *testing.T) {
	store := NewMemoryStore()
	engine := featureflags.NewEngine(store)
	ctx := context.Background()

	// Create flag with targets
	err := engine.SetFlag(ctx, &featureflags.Flag{
		Key:     "beta-feature",
		Enabled: true,
		Targets: []string{"user-1", "org-a"},
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	// Targeted user should get flag
	enabled, err := engine.IsEnabled(ctx, "beta-feature", &featureflags.EvaluationContext{
		UserID: "user-1",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected enabled for targeted user")
	}

	// Non-targeted user should not get flag
	enabled, err = engine.IsEnabled(ctx, "beta-feature", &featureflags.EvaluationContext{
		UserID: "user-2",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected disabled for non-targeted user")
	}
}

func TestFeatureFlagPercentage(t *testing.T) {
	store := NewMemoryStore()
	engine := featureflags.NewEngine(store)
	ctx := context.Background()

	// Create flag with 50% rollout
	err := engine.SetFlag(ctx, &featureflags.Flag{
		Key:        "gradual-rollout",
		Enabled:    true,
		Percentage: 50,
	})
	if err != nil {
		t.Fatalf("SetFlag failed: %v", err)
	}

	// Test consistency - same user should always get same result
	evalCtx := &featureflags.EvaluationContext{UserID: "test-user-1"}
	first, _ := engine.IsEnabled(ctx, "gradual-rollout", evalCtx)
	for i := 0; i < 10; i++ {
		result, _ := engine.IsEnabled(ctx, "gradual-rollout", evalCtx)
		if result != first {
			t.Error("percentage rollout should be consistent for same user")
		}
	}
}

func TestFeatureFlagEnableDisable(t *testing.T) {
	store := NewMemoryStore()
	engine := featureflags.NewEngine(store)
	ctx := context.Background()

	// Enable flag
	err := engine.EnableFlag(ctx, "toggle-feature")
	if err != nil {
		t.Fatalf("EnableFlag failed: %v", err)
	}

	enabled, _ := engine.IsEnabledSimple(ctx, "toggle-feature")
	if !enabled {
		t.Error("expected enabled after EnableFlag")
	}

	// Disable flag
	err = engine.DisableFlag(ctx, "toggle-feature")
	if err != nil {
		t.Fatalf("DisableFlag failed: %v", err)
	}

	enabled, _ = engine.IsEnabledSimple(ctx, "toggle-feature")
	if enabled {
		t.Error("expected disabled after DisableFlag")
	}
}
