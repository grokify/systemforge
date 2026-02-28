package rls

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// TxOptions configures transaction behavior.
type TxOptions struct {
	// TenantID is the tenant to scope the transaction to.
	TenantID uuid.UUID

	// UserID is the user executing the transaction.
	UserID uuid.UUID

	// IsolationLevel sets the transaction isolation level.
	IsolationLevel sql.IsolationLevel

	// ReadOnly marks the transaction as read-only.
	ReadOnly bool
}

// TxFunc is a function that executes within a transaction.
type TxFunc func(tx *sql.Tx) error

// WithTenant executes a function within a transaction with RLS context set.
// The tenant and user session variables are set before the function executes,
// and the transaction is committed if the function returns nil.
//
// Example:
//
//	err := rls.WithTenant(ctx, db, helper, tenantID, userID, func(tx *sql.Tx) error {
//	    _, err := tx.ExecContext(ctx, "INSERT INTO items (name) VALUES ($1)", name)
//	    return err
//	})
func WithTenant(ctx context.Context, db *sql.DB, helper *Helper, tenantID, userID uuid.UUID, fn TxFunc) error {
	opts := &TxOptions{
		TenantID: tenantID,
		UserID:   userID,
	}
	return WithTenantOpts(ctx, db, helper, opts, fn)
}

// WithTenantOpts executes a function within a transaction with RLS context and custom options.
func WithTenantOpts(ctx context.Context, db *sql.DB, helper *Helper, opts *TxOptions, fn TxFunc) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: opts.IsolationLevel,
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	// Set RLS context within the transaction
	if opts.TenantID != uuid.Nil {
		if err := helper.SetTenant(ctx, tx, opts.TenantID.String()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("setting tenant context: %w", err)
		}
	}

	if opts.UserID != uuid.Nil {
		if err := helper.SetUser(ctx, tx, opts.UserID.String()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("setting user context: %w", err)
		}
	}

	// Execute the function
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// WithTenantFromContext executes a function within a transaction using
// tenant and user IDs from the context.
func WithTenantFromContext(ctx context.Context, db *sql.DB, helper *Helper, fn TxFunc) error {
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)

	return WithTenant(ctx, db, helper, tenantID, userID, fn)
}

// TxManager manages transactions with automatic RLS context.
type TxManager struct {
	db     *sql.DB
	helper *Helper
}

// NewTxManager creates a new transaction manager.
func NewTxManager(db *sql.DB, cfg *Config) *TxManager {
	return &TxManager{
		db:     db,
		helper: NewHelper(cfg),
	}
}

// WithTenant executes a function within a tenant-scoped transaction.
func (m *TxManager) WithTenant(ctx context.Context, tenantID, userID uuid.UUID, fn TxFunc) error {
	return WithTenant(ctx, m.db, m.helper, tenantID, userID, fn)
}

// WithTenantFromContext executes a function using context tenant/user.
func (m *TxManager) WithTenantFromContext(ctx context.Context, fn TxFunc) error {
	return WithTenantFromContext(ctx, m.db, m.helper, fn)
}

// Begin starts a new transaction with RLS context set.
// The caller is responsible for committing or rolling back.
func (m *TxManager) Begin(ctx context.Context, tenantID, userID uuid.UUID) (*sql.Tx, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}

	if tenantID != uuid.Nil {
		if err := m.helper.SetTenant(ctx, tx, tenantID.String()); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("setting tenant context: %w", err)
		}
	}

	if userID != uuid.Nil {
		if err := m.helper.SetUser(ctx, tx, userID.String()); err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("setting user context: %w", err)
		}
	}

	return tx, nil
}

// BeginFromContext starts a transaction using context tenant/user.
func (m *TxManager) BeginFromContext(ctx context.Context) (*sql.Tx, error) {
	return m.Begin(ctx, TenantIDFromContext(ctx), UserIDFromContext(ctx))
}
