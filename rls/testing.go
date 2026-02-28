package rls

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// TestHelper provides testing utilities for RLS.
type TestHelper struct {
	db     *sql.DB
	helper *Helper
	t      *testing.T
}

// NewTestHelper creates a new RLS test helper.
func NewTestHelper(t *testing.T, db *sql.DB, cfg *Config) *TestHelper {
	return &TestHelper{
		db:     db,
		helper: NewHelper(cfg),
		t:      t,
	}
}

// AsUser executes a test function with RLS context set for a specific user and tenant.
//
// Example:
//
//	th := rls.NewTestHelper(t, db, nil)
//	th.AsUser(tenantID, userID, func(ctx context.Context, tx *sql.Tx) {
//	    // Queries here are scoped to tenantID
//	    rows, err := tx.QueryContext(ctx, "SELECT * FROM items")
//	    // ... assertions
//	})
func (th *TestHelper) AsUser(tenantID, userID uuid.UUID, fn func(ctx context.Context, tx *sql.Tx)) {
	th.t.Helper()

	ctx := context.Background()
	ctx = ContextWithTenant(ctx, tenantID)
	ctx = ContextWithUser(ctx, userID)

	tx, err := th.db.BeginTx(ctx, nil)
	if err != nil {
		th.t.Fatalf("failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Set RLS context
	if err := th.helper.SetTenant(ctx, tx, tenantID.String()); err != nil {
		th.t.Fatalf("failed to set tenant: %v", err)
	}
	if err := th.helper.SetUser(ctx, tx, userID.String()); err != nil {
		th.t.Fatalf("failed to set user: %v", err)
	}

	fn(ctx, tx)
}

// AsTenant executes a test function with only tenant context (no specific user).
func (th *TestHelper) AsTenant(tenantID uuid.UUID, fn func(ctx context.Context, tx *sql.Tx)) {
	th.t.Helper()

	ctx := context.Background()
	ctx = ContextWithTenant(ctx, tenantID)

	tx, err := th.db.BeginTx(ctx, nil)
	if err != nil {
		th.t.Fatalf("failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := th.helper.SetTenant(ctx, tx, tenantID.String()); err != nil {
		th.t.Fatalf("failed to set tenant: %v", err)
	}

	fn(ctx, tx)
}

// WithoutRLS executes a test function without RLS context.
// Useful for setup/teardown or cross-tenant assertions.
func (th *TestHelper) WithoutRLS(fn func(ctx context.Context, tx *sql.Tx)) {
	th.t.Helper()

	ctx := context.Background()
	tx, err := th.db.BeginTx(ctx, nil)
	if err != nil {
		th.t.Fatalf("failed to begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	fn(ctx, tx)
}

// AssertTenantIsolation verifies that RLS properly isolates data between tenants.
// It inserts data as tenantA, then verifies tenantB cannot see it.
func (th *TestHelper) AssertTenantIsolation(table, insertCol, valueCol string) {
	th.t.Helper()

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()
	testValue := uuid.NewString()

	// Insert as tenant A
	th.AsUser(tenantA, userA, func(ctx context.Context, tx *sql.Tx) {
		_, err := tx.ExecContext(ctx,
			fmt.Sprintf("INSERT INTO %s (organization_id, %s) VALUES ($1, $2)", table, insertCol),
			tenantA, testValue,
		)
		if err != nil {
			th.t.Fatalf("failed to insert as tenant A: %v", err)
		}

		// Tenant A should see the row
		var count int
		err = tx.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", table, valueCol),
			testValue,
		).Scan(&count)
		if err != nil {
			th.t.Fatalf("failed to query as tenant A: %v", err)
		}
		if count != 1 {
			th.t.Errorf("tenant A should see 1 row, got %d", count)
		}
	})

	// Tenant B should NOT see tenant A's data
	th.AsUser(tenantB, userB, func(ctx context.Context, tx *sql.Tx) {
		var count int
		err := tx.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", table, valueCol),
			testValue,
		).Scan(&count)
		if err != nil {
			th.t.Fatalf("failed to query as tenant B: %v", err)
		}
		if count != 0 {
			th.t.Errorf("tenant B should see 0 rows (isolation violation), got %d", count)
		}
	})
}

// AssertCanRead verifies that a tenant can read specific data.
func (th *TestHelper) AssertCanRead(tenantID, userID uuid.UUID, table, whereClause string, args ...any) {
	th.t.Helper()

	th.AsUser(tenantID, userID, func(ctx context.Context, tx *sql.Tx) {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, whereClause) //nolint:gosec // G201: test helper - table/whereClause are test-controlled, not user input
		err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			th.t.Fatalf("query failed: %v", err)
		}
		if count == 0 {
			th.t.Errorf("expected to read data from %s but got 0 rows", table)
		}
	})
}

// AssertCannotRead verifies that a tenant cannot read specific data.
func (th *TestHelper) AssertCannotRead(tenantID, userID uuid.UUID, table, whereClause string, args ...any) {
	th.t.Helper()

	th.AsUser(tenantID, userID, func(ctx context.Context, tx *sql.Tx) {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, whereClause) //nolint:gosec // G201: test helper - table/whereClause are test-controlled, not user input
		err := tx.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			th.t.Fatalf("query failed: %v", err)
		}
		if count > 0 {
			th.t.Errorf("expected 0 rows from %s but got %d (isolation violation)", table, count)
		}
	})
}

// SetupTestTenant creates a test tenant and returns cleanup function.
func (th *TestHelper) SetupTestTenant(name string) (tenantID uuid.UUID, cleanup func()) {
	th.t.Helper()

	tenantID = uuid.New()
	ctx := context.Background()

	_, err := th.db.ExecContext(ctx,
		"INSERT INTO cf_organizations (id, name, slug) VALUES ($1, $2, $3)",
		tenantID, name, name,
	)
	if err != nil {
		th.t.Fatalf("failed to create test tenant: %v", err)
	}

	cleanup = func() {
		_, _ = th.db.ExecContext(ctx, "DELETE FROM cf_organizations WHERE id = $1", tenantID)
	}

	return tenantID, cleanup
}

// SetupTestUser creates a test user in a tenant and returns cleanup function.
func (th *TestHelper) SetupTestUser(tenantID uuid.UUID, email, role string) (userID uuid.UUID, cleanup func()) {
	th.t.Helper()

	userID = uuid.New()
	ctx := context.Background()

	_, err := th.db.ExecContext(ctx,
		"INSERT INTO cf_users (id, email, name) VALUES ($1, $2, $3)",
		userID, email, email,
	)
	if err != nil {
		th.t.Fatalf("failed to create test user: %v", err)
	}

	membershipID := uuid.New()
	_, err = th.db.ExecContext(ctx,
		"INSERT INTO cf_memberships (id, user_id, organization_id, role) VALUES ($1, $2, $3, $4)",
		membershipID, userID, tenantID, role,
	)
	if err != nil {
		th.t.Fatalf("failed to create membership: %v", err)
	}

	cleanup = func() {
		_, _ = th.db.ExecContext(ctx, "DELETE FROM cf_memberships WHERE id = $1", membershipID)
		_, _ = th.db.ExecContext(ctx, "DELETE FROM cf_users WHERE id = $1", userID)
	}

	return userID, cleanup
}
