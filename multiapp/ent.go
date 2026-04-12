package multiapp

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// EntConfig provides configuration for Ent client creation.
type EntConfig struct {
	// Schema is the PostgreSQL schema name for this app.
	Schema string

	// Debug enables Ent debug logging.
	Debug bool
}

// EntClientFactory creates Ent clients scoped to app schemas.
// This bridges the multiapp SchemaDB with Ent's database layer.
type EntClientFactory struct {
	db *AppAwareDB
}

// NewEntClientFactory creates a factory for app-scoped Ent clients.
func NewEntClientFactory(db *AppAwareDB) *EntClientFactory {
	return &EntClientFactory{db: db}
}

// StdDBForSchema returns a *sql.DB configured for a specific schema.
// The returned connection has search_path set to the app's schema.
//
// Usage with Ent:
//
//	factory := multiapp.NewEntClientFactory(appAwareDB)
//	db, err := factory.StdDBForSchema(ctx, "app_app1")
//	if err != nil {
//	    return err
//	}
//	drv := entsql.OpenDB(dialect.Postgres, db)
//	client := ent.NewClient(ent.Driver(drv))
func (f *EntClientFactory) StdDBForSchema(ctx context.Context, schema string) (*sql.DB, error) {
	// Get connection config from pool
	config := f.db.pool.Config().ConnConfig.Copy()

	// Add search_path to connection parameters
	// This ensures all connections from this pool use the correct schema
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["search_path"] = schema + ", public"

	// Create a new connection string with the modified config
	connStr := stdlib.RegisterConnConfig(config)

	// Open standard database connection
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("multiapp: failed to open database for schema %s: %w", schema, err)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("multiapp: failed to ping database for schema %s: %w", schema, err)
	}

	return db, nil
}

// EntDriverForSchema returns an Ent SQL driver scoped to a schema.
// This is a convenience wrapper around StdDBForSchema.
//
// Usage:
//
//	drv, err := factory.EntDriverForSchema(ctx, "app_app1")
//	client := ent.NewClient(ent.Driver(drv))
func (f *EntClientFactory) EntDriverForSchema(ctx context.Context, schema string) (*entsql.Driver, error) {
	db, err := f.StdDBForSchema(ctx, schema)
	if err != nil {
		return nil, err
	}
	return entsql.OpenDB(dialect.Postgres, db), nil
}

// SchemaFromAppSlug returns the database schema name for an app slug.
// By convention, app schemas are named "app_{slug}".
func SchemaFromAppSlug(slug string) string {
	return "app_" + slug
}

// EntConnHook provides a hook that sets schema context for each connection.
// Use this with Ent's connection hooks for dynamic schema selection.
type EntConnHook struct {
	db *AppAwareDB
}

// NewEntConnHook creates a new Ent connection hook.
func NewEntConnHook(db *AppAwareDB) *EntConnHook {
	return &EntConnHook{db: db}
}

// SetSchemaFromContext sets the search_path based on AppContext in the context.
// This should be used with Ent's BeginTx or connection acquire hooks.
func (h *EntConnHook) SetSchemaFromContext(ctx context.Context, conn *sql.Conn) error {
	appCtx := AppContextFromContext(ctx)
	if appCtx == nil {
		return ErrNoAppContext
	}

	schema := appCtx.DatabaseSchema
	if schema == "" {
		return fmt.Errorf("multiapp: app context has no database schema")
	}

	_, err := conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schema))
	if err != nil {
		return fmt.Errorf("multiapp: failed to set search_path: %w", err)
	}

	return nil
}

// PgxConnWithSchema acquires a pgx connection with schema set.
// This is useful for raw pgx operations outside of Ent.
func (f *EntClientFactory) PgxConnWithSchema(ctx context.Context, schema string) (*pgx.Conn, error) {
	config := f.db.pool.Config().ConnConfig.Copy()
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["search_path"] = schema + ", public"

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("multiapp: failed to connect with schema %s: %w", schema, err)
	}

	return conn, nil
}
