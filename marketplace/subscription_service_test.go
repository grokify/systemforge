package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grokify/coreforge/identity/ent"
	"github.com/grokify/coreforge/identity/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
)

func newSubscriptionTestClient(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	})
	return client
}

// createTestOrg creates a test organization for subscription tests.
func createTestOrg(t *testing.T, ctx context.Context, client *ent.Client, suffix string) uuid.UUID {
	org, err := client.Organization.Create().
		SetName("Test Org " + suffix).
		SetSlug("test-org-sub-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}
	return org.ID
}

func TestEntSubscriptionService_CreateAndGet(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "create-1")

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierPro,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}

	err := svc.Create(ctx, sub)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify subscription was created
	if sub.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set after Create")
	}

	// Get the subscription
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.OrganizationID != orgID {
		t.Errorf("OrganizationID = %v, want %v", got.OrganizationID, orgID)
	}
	if got.PlanTier != PlanTierPro {
		t.Errorf("PlanTier = %v, want %v", got.PlanTier, PlanTierPro)
	}
	if got.Status != SubscriptionStatusActive {
		t.Errorf("Status = %v, want %v", got.Status, SubscriptionStatusActive)
	}

	// Get non-existent
	_, err = svc.Get(ctx, uuid.New())
	if err != ErrSubscriptionNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_GetByOrg(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "getbyorg-1")

	// Get non-existent
	_, err := svc.GetByOrg(ctx, orgID)
	if err != ErrSubscriptionNotFound {
		t.Errorf("GetByOrg() error = %v, want %v", err, ErrSubscriptionNotFound)
	}

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierStarter,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get by org
	got, err := svc.GetByOrg(ctx, orgID)
	if err != nil {
		t.Fatalf("GetByOrg() error = %v", err)
	}
	if got.ID != sub.ID {
		t.Errorf("ID = %v, want %v", got.ID, sub.ID)
	}
}

func TestEntSubscriptionService_Update(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "update-1")

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierStarter,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the subscription
	sub.PlanTier = PlanTierPro
	sub.CancelAtPeriodEnd = true
	sub.StripeSubscriptionID = "sub_test123"
	sub.StripeCustomerID = "cus_test123"

	if err := svc.Update(ctx, sub); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.PlanTier != PlanTierPro {
		t.Errorf("PlanTier = %v, want %v", got.PlanTier, PlanTierPro)
	}
	if !got.CancelAtPeriodEnd {
		t.Error("CancelAtPeriodEnd = false, want true")
	}
	if got.StripeSubscriptionID != "sub_test123" {
		t.Errorf("StripeSubscriptionID = %v, want sub_test123", got.StripeSubscriptionID)
	}

	// Update non-existent
	sub.ID = uuid.New()
	err = svc.Update(ctx, sub)
	if err != ErrSubscriptionNotFound {
		t.Errorf("Update() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_Cancel(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "cancel-1")

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierPro,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cancel at period end
	if err := svc.Cancel(ctx, sub.ID); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	// Verify cancel flag set
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !got.CancelAtPeriodEnd {
		t.Error("CancelAtPeriodEnd = false, want true")
	}
	// Status should still be active
	if got.Status != SubscriptionStatusActive {
		t.Errorf("Status = %v, want %v (should remain active until period end)", got.Status, SubscriptionStatusActive)
	}

	// Cancel non-existent
	err = svc.Cancel(ctx, uuid.New())
	if err != ErrSubscriptionNotFound {
		t.Errorf("Cancel() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_CancelImmediately(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "cancel-imm-1")

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierPro,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Cancel immediately
	if err := svc.CancelImmediately(ctx, sub.ID); err != nil {
		t.Fatalf("CancelImmediately() error = %v", err)
	}

	// Verify status changed
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status != SubscriptionStatusCanceled {
		t.Errorf("Status = %v, want %v", got.Status, SubscriptionStatusCanceled)
	}

	// Cancel non-existent
	err = svc.CancelImmediately(ctx, uuid.New())
	if err != ErrSubscriptionNotFound {
		t.Errorf("CancelImmediately() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_Reactivate(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "reactivate-1")

	// Create and cancel a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierPro,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := svc.Cancel(ctx, sub.ID); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	// Reactivate
	if err := svc.Reactivate(ctx, sub.ID); err != nil {
		t.Fatalf("Reactivate() error = %v", err)
	}

	// Verify status and cancel flag
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status != SubscriptionStatusActive {
		t.Errorf("Status = %v, want %v", got.Status, SubscriptionStatusActive)
	}
	if got.CancelAtPeriodEnd {
		t.Error("CancelAtPeriodEnd = true, want false")
	}

	// Reactivate non-existent
	err = svc.Reactivate(ctx, uuid.New())
	if err != ErrSubscriptionNotFound {
		t.Errorf("Reactivate() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_ChangePlan(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "changeplan-1")

	// Create a subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierStarter,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Change plan
	if err := svc.ChangePlan(ctx, sub.ID, PlanTierEnterprise); err != nil {
		t.Fatalf("ChangePlan() error = %v", err)
	}

	// Verify plan changed
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.PlanTier != PlanTierEnterprise {
		t.Errorf("PlanTier = %v, want %v", got.PlanTier, PlanTierEnterprise)
	}

	// Change plan non-existent
	err = svc.ChangePlan(ctx, uuid.New(), PlanTierPro)
	if err != ErrSubscriptionNotFound {
		t.Errorf("ChangePlan() error = %v, want %v", err, ErrSubscriptionNotFound)
	}
}

func TestEntSubscriptionService_IsActive(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "isactive-1")

	// Check org with no subscription
	active, err := svc.IsActive(ctx, orgID)
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if active {
		t.Error("IsActive() = true, want false for no subscription")
	}

	// Create active subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierPro,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Check active subscription
	active, err = svc.IsActive(ctx, orgID)
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if !active {
		t.Error("IsActive() = false, want true for active subscription")
	}

	// Cancel and check again
	if err := svc.CancelImmediately(ctx, sub.ID); err != nil {
		t.Fatalf("CancelImmediately() error = %v", err)
	}
	active, err = svc.IsActive(ctx, orgID)
	if err != nil {
		t.Fatalf("IsActive() error = %v", err)
	}
	if active {
		t.Error("IsActive() = true, want false for canceled subscription")
	}
}

func TestEntSubscriptionService_GetPlanTier(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "plantier-1")

	// Get plan tier with no subscription (should return free)
	tier, err := svc.GetPlanTier(ctx, orgID)
	if err != nil {
		t.Fatalf("GetPlanTier() error = %v", err)
	}
	if tier != PlanTierFree {
		t.Errorf("GetPlanTier() = %v, want %v for no subscription", tier, PlanTierFree)
	}

	// Create subscription
	now := time.Now()
	sub := &Subscription{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		PlanTier:           PlanTierEnterprise,
		Status:             SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
	}
	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get plan tier
	tier, err = svc.GetPlanTier(ctx, orgID)
	if err != nil {
		t.Fatalf("GetPlanTier() error = %v", err)
	}
	if tier != PlanTierEnterprise {
		t.Errorf("GetPlanTier() = %v, want %v", tier, PlanTierEnterprise)
	}
}

func TestEntSubscriptionService_WithStripeIDs(t *testing.T) {
	client := newSubscriptionTestClient(t)
	svc := NewEntSubscriptionService(client)
	ctx := context.Background()

	orgID := createTestOrg(t, ctx, client, "stripe-1")

	// Create subscription with Stripe IDs
	now := time.Now()
	sub := &Subscription{
		ID:                   uuid.New(),
		OrganizationID:       orgID,
		PlanTier:             PlanTierPro,
		Status:               SubscriptionStatusTrialing,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.Add(14 * 24 * time.Hour),
		StripeSubscriptionID: "sub_abc123",
		StripeCustomerID:     "cus_xyz789",
	}

	if err := svc.Create(ctx, sub); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify Stripe IDs persisted
	got, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.StripeSubscriptionID != "sub_abc123" {
		t.Errorf("StripeSubscriptionID = %v, want sub_abc123", got.StripeSubscriptionID)
	}
	if got.StripeCustomerID != "cus_xyz789" {
		t.Errorf("StripeCustomerID = %v, want cus_xyz789", got.StripeCustomerID)
	}

	// Verify trialing is considered active
	if !got.IsActive() {
		t.Error("IsActive() = false, want true for trialing subscription")
	}
	if !got.IsInTrial() {
		t.Error("IsInTrial() = false, want true for trialing subscription")
	}
}
