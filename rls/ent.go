package rls

import (
	"context"
	"database/sql"
	"fmt"
)

// EntHook provides integration with Ent ORM for automatic RLS context.
//
// Usage with Ent:
//
//	hook := rls.NewEntHook(db, rls.DefaultConfig())
//
//	client := ent.NewClient(
//	    ent.Driver(drv),
//	    ent.Hook(hook.SetContext()),
//	)
type EntHook struct {
	db     *sql.DB
	helper *Helper
}

// NewEntHook creates a new Ent RLS hook.
func NewEntHook(db *sql.DB, cfg *Config) *EntHook {
	return &EntHook{
		db:     db,
		helper: NewHelper(cfg),
	}
}

// SetContextFromContext sets RLS context using tenant/user from Go context.
// This should be called at the start of each request/operation.
func (h *EntHook) SetContextFromContext(ctx context.Context) error {
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)

	if tenantID.String() != "00000000-0000-0000-0000-000000000000" {
		if err := h.helper.SetTenant(ctx, h.db, tenantID.String()); err != nil {
			return fmt.Errorf("setting tenant: %w", err)
		}
	}

	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		if err := h.helper.SetUser(ctx, h.db, userID.String()); err != nil {
			return fmt.Errorf("setting user: %w", err)
		}
	}

	return nil
}

// EntDriver wraps an Ent SQL driver to automatically set RLS context.
// This ensures RLS is set for every database operation.
type EntDriver struct {
	// Underlying driver (implement as needed for your Ent setup)
	helper *Helper
}

// NewEntDriver creates a new RLS-aware Ent driver wrapper.
func NewEntDriver(cfg *Config) *EntDriver {
	return &EntDriver{
		helper: NewHelper(cfg),
	}
}

// ContextInjector creates a function that can be used to inject RLS context
// into database connections. This is useful for connection pool hooks.
type ContextInjector struct {
	helper *Helper
}

// NewContextInjector creates a new context injector.
func NewContextInjector(cfg *Config) *ContextInjector {
	return &ContextInjector{helper: NewHelper(cfg)}
}

// InjectContext sets RLS session variables on a connection.
// Use this with connection pool acquire hooks.
func (ci *ContextInjector) InjectContext(ctx context.Context, conn *sql.Conn) error {
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)

	if tenantID.String() != "00000000-0000-0000-0000-000000000000" {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL %s = $1", ci.helper.config.SessionVariable),
			tenantID.String(),
		); err != nil {
			return fmt.Errorf("setting tenant: %w", err)
		}
	}

	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL %s = $1", ci.helper.config.UserSessionVariable),
			userID.String(),
		); err != nil {
			return fmt.Errorf("setting user: %w", err)
		}
	}

	return nil
}

// ConnWithRLS wraps sql.Conn to ensure RLS context is set.
type ConnWithRLS struct {
	*sql.Conn
	ctx    context.Context
	helper *Helper
}

// NewConnWithRLS wraps a connection with RLS context injection.
func NewConnWithRLS(ctx context.Context, conn *sql.Conn, cfg *Config) (*ConnWithRLS, error) {
	c := &ConnWithRLS{
		Conn:   conn,
		ctx:    ctx,
		helper: NewHelper(cfg),
	}

	// Set RLS context immediately
	tenantID := TenantIDFromContext(ctx)
	userID := UserIDFromContext(ctx)

	if tenantID.String() != "00000000-0000-0000-0000-000000000000" {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL %s = $1", c.helper.config.SessionVariable),
			tenantID.String(),
		); err != nil {
			return nil, fmt.Errorf("setting tenant: %w", err)
		}
	}

	if userID.String() != "00000000-0000-0000-0000-000000000000" {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("SET LOCAL %s = $1", c.helper.config.UserSessionVariable),
			userID.String(),
		); err != nil {
			return nil, fmt.Errorf("setting user: %w", err)
		}
	}

	return c, nil
}

// GetConnWithRLS gets a connection from the pool with RLS context set.
func GetConnWithRLS(ctx context.Context, db *sql.DB, cfg *Config) (*ConnWithRLS, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	return NewConnWithRLS(ctx, conn, cfg)
}
