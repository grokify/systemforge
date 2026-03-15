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

func newLicenseTestClient(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Logf("failed to close client: %v", err)
		}
	})
	return client
}

// createTestLicenseFixtures creates org, principal, and listing for license tests.
func createTestLicenseFixtures(t *testing.T, ctx context.Context, client *ent.Client, listingSvc *EntListingService) (orgID, principalID, listingID uuid.UUID) {
	org, err := client.Organization.Create().
		SetName("Test Org").
		SetSlug("test-org-license-" + uuid.New().String()[:8]).
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	principal, err := client.Principal.Create().
		SetType("human").
		SetIdentifier("license-test@example.com").
		SetDisplayName("License Test User").
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create principal: %v", err)
	}

	// Create and publish a listing
	listing := &Listing{
		ID:           uuid.New(),
		CreatorOrgID: org.ID,
		OwnerID:      principal.ID,
		ProductType:  "course",
		ProductID:    uuid.New(),
		Title:        "Test Course for License",
		PricingModel: PricingPerSeat,
		PriceCents:   9999,
		Currency:     "USD",
	}
	if err := listingSvc.Create(ctx, listing); err != nil {
		t.Fatalf("failed to create listing: %v", err)
	}
	if err := listingSvc.Publish(ctx, listing.ID); err != nil {
		t.Fatalf("failed to publish listing: %v", err)
	}

	return org.ID, principal.ID, listing.ID
}

func TestEntLicenseService_GrantAndGet(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create a license
	seats := 10
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseSeatBased,
		Seats:          &seats,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}

	err := svc.Grant(ctx, license)
	if err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Verify license was created
	if license.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set after Grant")
	}

	// Get the license
	got, err := svc.Get(ctx, license.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ListingID != license.ListingID {
		t.Errorf("ListingID = %v, want %v", got.ListingID, license.ListingID)
	}
	if got.OrganizationID != license.OrganizationID {
		t.Errorf("OrganizationID = %v, want %v", got.OrganizationID, license.OrganizationID)
	}
	if got.LicenseType != LicenseSeatBased {
		t.Errorf("LicenseType = %v, want %v", got.LicenseType, LicenseSeatBased)
	}
	if got.Seats == nil || *got.Seats != 10 {
		t.Errorf("Seats = %v, want 10", got.Seats)
	}
}

func TestEntLicenseService_GrantDuplicate(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create first license
	license1 := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license1); err != nil {
		t.Fatalf("Grant() first license error = %v", err)
	}

	// Try to create duplicate license
	license2 := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	err := svc.Grant(ctx, license2)
	if err != ErrAlreadyLicensed {
		t.Errorf("Grant() duplicate error = %v, want %v", err, ErrAlreadyLicensed)
	}
}

func TestEntLicenseService_GetByListingAndOrg(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create a license
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Get by listing and org
	got, err := svc.GetByListingAndOrg(ctx, listingID, orgID)
	if err != nil {
		t.Fatalf("GetByListingAndOrg() error = %v", err)
	}
	if got.ID != license.ID {
		t.Errorf("ID = %v, want %v", got.ID, license.ID)
	}

	// Get non-existent
	_, err = svc.GetByListingAndOrg(ctx, uuid.New(), uuid.New())
	if err != ErrLicenseNotFound {
		t.Errorf("GetByListingAndOrg() error = %v, want %v", err, ErrLicenseNotFound)
	}
}

//nolint:dupl // Test functions have similar structure but test different methods
func TestEntLicenseService_Check(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Check before license exists
	valid, err := svc.Check(ctx, listingID, orgID)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if valid {
		t.Error("Check() = true, want false for non-existent license")
	}

	// Create a valid license
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now().Add(-time.Hour),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Check valid license
	valid, err = svc.Check(ctx, listingID, orgID)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !valid {
		t.Error("Check() = false, want true for valid license")
	}
}

func TestEntLicenseService_Revoke(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create a license
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Revoke the license
	if err := svc.Revoke(ctx, license.ID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	// Verify it's gone
	_, err := svc.Get(ctx, license.ID)
	if err != ErrLicenseNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrLicenseNotFound)
	}

	// Revoke non-existent
	err = svc.Revoke(ctx, uuid.New())
	if err != ErrLicenseNotFound {
		t.Errorf("Revoke() error = %v, want %v", err, ErrLicenseNotFound)
	}
}

func TestEntLicenseService_List(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create multiple licenses for different listings
	for i := 0; i < 3; i++ {
		// Create another listing
		listing := &Listing{
			ID:           uuid.New(),
			CreatorOrgID: orgID,
			OwnerID:      principalID,
			ProductType:  "course",
			ProductID:    uuid.New(),
			Title:        "Additional Course",
			PricingModel: PricingOneTime,
			Currency:     "USD",
		}
		if err := listingSvc.Create(ctx, listing); err != nil {
			t.Fatalf("Create listing error = %v", err)
		}
		if err := listingSvc.Publish(ctx, listing.ID); err != nil {
			t.Fatalf("Publish listing error = %v", err)
		}

		license := &License{
			ID:             uuid.New(),
			ListingID:      listing.ID,
			OrganizationID: orgID,
			LicenseType:    LicenseUnlimited,
			ValidFrom:      time.Now(),
			PurchasedBy:    principalID,
		}
		if err := svc.Grant(ctx, license); err != nil {
			t.Fatalf("Grant() error = %v", err)
		}
	}

	// List by org
	licenses, err := svc.List(ctx, orgID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(licenses) != 3 {
		t.Errorf("List() count = %v, want 3", len(licenses))
	}

	// List by listing (should be 0 for the original listing since we didn't create one for it in this test section)
	licenses, err = svc.ListByListing(ctx, listingID)
	if err != nil {
		t.Fatalf("ListByListing() error = %v", err)
	}
	// Should be 0 since we created licenses for new listings, not the fixture listing
	if len(licenses) != 0 {
		t.Errorf("ListByListing() count = %v, want 0", len(licenses))
	}
}

func TestEntLicenseService_SeatAssignment(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create a seat-based license with 2 seats
	seats := 2
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseSeatBased,
		Seats:          &seats,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Create additional principals for seat assignment
	principal2, err := client.Principal.Create().
		SetType("human").
		SetIdentifier("user2@example.com").
		SetDisplayName("User 2").
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create principal2: %v", err)
	}

	principal3, err := client.Principal.Create().
		SetType("human").
		SetIdentifier("user3@example.com").
		SetDisplayName("User 3").
		Save(ctx)
	if err != nil {
		t.Fatalf("failed to create principal3: %v", err)
	}

	// Assign first seat
	assignment1 := &SeatAssignment{
		ID:          uuid.New(),
		LicenseID:   license.ID,
		PrincipalID: principalID,
		AssignedBy:  principalID,
	}
	if err := svc.AssignSeat(ctx, assignment1); err != nil {
		t.Fatalf("AssignSeat() first seat error = %v", err)
	}

	// Verify used seats increased
	got, _ := svc.Get(ctx, license.ID)
	if got.UsedSeats != 1 {
		t.Errorf("UsedSeats = %v, want 1", got.UsedSeats)
	}

	// Try to assign duplicate seat (while seats are still available)
	assignmentDup := &SeatAssignment{
		ID:          uuid.New(),
		LicenseID:   license.ID,
		PrincipalID: principalID,
		AssignedBy:  principalID,
	}
	err = svc.AssignSeat(ctx, assignmentDup)
	if err != ErrSeatAlreadyAssigned {
		t.Errorf("AssignSeat() duplicate error = %v, want %v", err, ErrSeatAlreadyAssigned)
	}

	// Assign second seat
	assignment2 := &SeatAssignment{
		ID:          uuid.New(),
		LicenseID:   license.ID,
		PrincipalID: principal2.ID,
		AssignedBy:  principalID,
	}
	if err := svc.AssignSeat(ctx, assignment2); err != nil {
		t.Fatalf("AssignSeat() second seat error = %v", err)
	}

	// Try to assign third seat (should fail - no seats available)
	assignment3 := &SeatAssignment{
		ID:          uuid.New(),
		LicenseID:   license.ID,
		PrincipalID: principal3.ID,
		AssignedBy:  principalID,
	}
	err = svc.AssignSeat(ctx, assignment3)
	if err != ErrNoSeatsAvailable {
		t.Errorf("AssignSeat() third seat error = %v, want %v", err, ErrNoSeatsAvailable)
	}

	// List seat assignments
	assignments, err := svc.ListSeatAssignments(ctx, license.ID)
	if err != nil {
		t.Fatalf("ListSeatAssignments() error = %v", err)
	}
	if len(assignments) != 2 {
		t.Errorf("ListSeatAssignments() count = %v, want 2", len(assignments))
	}

	// Unassign a seat
	if err := svc.UnassignSeat(ctx, license.ID, principalID); err != nil {
		t.Fatalf("UnassignSeat() error = %v", err)
	}

	// Verify used seats decreased
	got, _ = svc.Get(ctx, license.ID)
	if got.UsedSeats != 1 {
		t.Errorf("UsedSeats = %v, want 1 after unassign", got.UsedSeats)
	}

	// Now third seat should succeed
	if err := svc.AssignSeat(ctx, assignment3); err != nil {
		t.Errorf("AssignSeat() third seat after unassign error = %v", err)
	}

	// Try to unassign non-existent seat
	err = svc.UnassignSeat(ctx, license.ID, uuid.New())
	if err != ErrSeatNotAssigned {
		t.Errorf("UnassignSeat() non-existent error = %v, want %v", err, ErrSeatNotAssigned)
	}
}

func TestEntLicenseService_Update(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Create a license
	seats := 5
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseSeatBased,
		Seats:          &seats,
		ValidFrom:      time.Now(),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Update the license
	newSeats := 10
	expiry := time.Now().Add(365 * 24 * time.Hour)
	license.Seats = &newSeats
	license.ValidUntil = &expiry
	license.LicenseType = LicenseTeam

	if err := svc.Update(ctx, license); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := svc.Get(ctx, license.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Seats == nil || *got.Seats != 10 {
		t.Errorf("Seats = %v, want 10", got.Seats)
	}
	if got.LicenseType != LicenseTeam {
		t.Errorf("LicenseType = %v, want %v", got.LicenseType, LicenseTeam)
	}
	if got.ValidUntil == nil {
		t.Error("ValidUntil should not be nil")
	}
}

//nolint:dupl // Test functions have similar structure but test different methods
func TestEntLicenseService_CheckPrincipal(t *testing.T) {
	client := newLicenseTestClient(t)
	listingSvc := NewEntListingService(client, nil)
	svc := NewEntLicenseService(client, nil)
	ctx := context.Background()

	orgID, principalID, listingID := createTestLicenseFixtures(t, ctx, client, listingSvc)

	// Check before any license
	hasAccess, err := svc.CheckPrincipal(ctx, listingID, principalID)
	if err != nil {
		t.Fatalf("CheckPrincipal() error = %v", err)
	}
	if hasAccess {
		t.Error("CheckPrincipal() = true, want false for no license")
	}

	// Create unlimited license
	license := &License{
		ID:             uuid.New(),
		ListingID:      listingID,
		OrganizationID: orgID,
		LicenseType:    LicenseUnlimited,
		ValidFrom:      time.Now().Add(-time.Hour),
		PurchasedBy:    principalID,
	}
	if err := svc.Grant(ctx, license); err != nil {
		t.Fatalf("Grant() error = %v", err)
	}

	// Check with unlimited license (should have access)
	hasAccess, err = svc.CheckPrincipal(ctx, listingID, principalID)
	if err != nil {
		t.Fatalf("CheckPrincipal() error = %v", err)
	}
	if !hasAccess {
		t.Error("CheckPrincipal() = false, want true for unlimited license")
	}
}
