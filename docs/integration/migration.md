# Migration Guide

This guide covers database migrations when integrating SystemForge.

## Migration Strategy

### Phase 1: Side-by-Side (Safe)

```
Week 1-2: Create SystemForge tables
Week 3:   Sync existing data
Week 4:   Enable dual-write
```

### Phase 2: Cutover (Careful)

```
Week 5:   Switch reads to SystemForge
Week 6:   Validate data integrity
Week 7:   Disable old writes
```

### Phase 3: Cleanup (Final)

```
Week 8:   Archive old tables
Week 9:   Drop old tables
Week 10:  Remove dual-write code
```

## Creating SystemForge Tables

### Using Ent Migrations

```go
package main

import (
    "context"
    "log"

    "github.com/grokify/systemforge/identity/ent"
    _ "github.com/lib/pq"
)

func main() {
    client, err := ent.Open("postgres",
        "postgres://user:pass@localhost/myapp?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Create tables
    if err := client.Schema.Create(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("SystemForge tables created")
}
```

### Using SQL Migrations

If you use a migration tool (golang-migrate, goose, etc.):

```sql
-- migrations/001_systemforge_users.up.sql
CREATE TABLE cf_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    password_hash TEXT,
    is_platform_admin BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cf_users_email ON cf_users(email);
CREATE INDEX idx_cf_users_active ON cf_users(active);
```

## Data Sync Scripts

### Sync Users

```go
func syncUsers(ctx context.Context, src, dst *sql.DB) error {
    tx, err := dst.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    rows, err := src.QueryContext(ctx, `
        SELECT id, email, name, COALESCE(avatar_url, ''),
               COALESCE(password_hash, ''), created_at
        FROM users
        WHERE deleted_at IS NULL
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO cf_users (id, email, name, avatar_url, password_hash, created_at, updated_at)
        VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $6)
        ON CONFLICT (id) DO UPDATE SET
            email = EXCLUDED.email,
            name = EXCLUDED.name,
            avatar_url = EXCLUDED.avatar_url,
            password_hash = EXCLUDED.password_hash,
            updated_at = now()
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    count := 0
    for rows.Next() {
        var id uuid.UUID
        var email, name, avatar, pwHash string
        var createdAt time.Time

        if err := rows.Scan(&id, &email, &name, &avatar, &pwHash, &createdAt); err != nil {
            return err
        }

        if _, err := stmt.ExecContext(ctx, id, email, name, avatar, pwHash, createdAt); err != nil {
            return fmt.Errorf("sync user %s: %w", id, err)
        }
        count++
    }

    if err := tx.Commit(); err != nil {
        return err
    }

    log.Printf("Synced %d users", count)
    return nil
}
```

### Sync Organizations

```go
func syncOrganizations(ctx context.Context, src, dst *sql.DB) error {
    rows, err := src.QueryContext(ctx, `
        SELECT id, name, slug, logo_url, settings, plan, created_at
        FROM tenants
        WHERE deleted_at IS NULL
    `)
    // ... similar pattern
}
```

### Sync Memberships

```go
func syncMemberships(ctx context.Context, src, dst *sql.DB) error {
    rows, err := src.QueryContext(ctx, `
        SELECT user_id, tenant_id, role, created_at
        FROM tenant_members
    `)
    // ... similar pattern
}
```

## Validation Queries

### Compare Counts

```sql
-- Users
SELECT
    (SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) as old_count,
    (SELECT COUNT(*) FROM cf_users) as new_count;

-- Organizations
SELECT
    (SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL) as old_count,
    (SELECT COUNT(*) FROM cf_organizations) as new_count;

-- Memberships
SELECT
    (SELECT COUNT(*) FROM tenant_members) as old_count,
    (SELECT COUNT(*) FROM cf_memberships) as new_count;
```

### Find Missing Records

```sql
-- Users in old table but not in SystemForge
SELECT u.id, u.email
FROM users u
LEFT JOIN cf_users cf ON u.id = cf.id
WHERE u.deleted_at IS NULL AND cf.id IS NULL;
```

### Compare Data

```sql
-- Find data mismatches
SELECT u.id, u.email as old_email, cf.email as new_email
FROM users u
JOIN cf_users cf ON u.id = cf.id
WHERE u.email != cf.email;
```

## Rollback Plan

### Keep Sync Running

During dual-write phase, sync changes back to old tables:

```go
// On SystemForge user update
func onUserUpdate(ctx context.Context, user *ent.User) {
    // Also update old table
    _, err := oldDB.ExecContext(ctx, `
        UPDATE users SET
            email = $2,
            name = $3,
            updated_at = now()
        WHERE id = $1
    `, user.ID, user.Email, user.Name)
    if err != nil {
        log.Printf("Warning: failed to sync to old table: %v", err)
    }
}
```

### Rollback Script

If issues arise, rollback reads to old tables:

```go
var useOldTables = os.Getenv("USE_OLD_TABLES") == "true"

func getUser(ctx context.Context, id uuid.UUID) (*User, error) {
    if useOldTables {
        return getOldUser(ctx, id)
    }
    return getSystemForgeUser(ctx, id)
}
```

## Archive and Cleanup

### Archive Old Tables

```sql
-- Create archive schema
CREATE SCHEMA archive;

-- Move old tables
ALTER TABLE users SET SCHEMA archive;
ALTER TABLE tenants SET SCHEMA archive;
ALTER TABLE tenant_members SET SCHEMA archive;
```

### Drop After Validation

```sql
-- After 30 days with no issues
DROP SCHEMA archive CASCADE;
```

## Common Issues

### UUID Mismatches

If your old tables use integers:

```go
// Create mapping table
_, err := db.ExecContext(ctx, `
    CREATE TABLE user_id_mapping (
        old_id BIGINT PRIMARY KEY,
        new_id UUID NOT NULL
    )
`)

// Generate UUIDs for existing records
_, err = db.ExecContext(ctx, `
    INSERT INTO user_id_mapping (old_id, new_id)
    SELECT id, gen_random_uuid()
    FROM users
`)
```

### Password Hash Format

If using different hash format:

```go
func migratePasswordHash(oldHash string) string {
    // If old hash is bcrypt, it works with SystemForge
    if strings.HasPrefix(oldHash, "$2a$") || strings.HasPrefix(oldHash, "$2b$") {
        return oldHash
    }

    // If old hash is unsupported, users must reset password
    return ""
}
```
