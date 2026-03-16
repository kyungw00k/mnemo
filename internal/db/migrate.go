package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migrate runs pending SQL migrations from the provided filesystem against the database.
// The migrations FS should contain files in a flat structure named like "001_initial.sql".
// dimensions is used to replace the $DIMENSIONS placeholder in PostgreSQL migrations.
func Migrate(ctx context.Context, conn *DBConn, migrationsFS fs.FS, dimensions int) error {
	// Create migration tracking table.
	createTable := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT DEFAULT (datetime('now'))
	)`
	if conn.IsPostgres() {
		createTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)`
	}
	if _, err := conn.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Read migration files from FS root.
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	// Sort by filename to ensure execution order.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := entry.Name()

		// Check whether this migration has already been applied.
		var count int
		var row interface{ Scan(...any) error }
		if conn.IsSQLite() {
			row = conn.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version)
		} else {
			row = conn.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version)
		}
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("check migration %q: %w", version, err)
		}
		if count > 0 {
			continue
		}

		// Read SQL content.
		content, err := fs.ReadFile(migrationsFS, version)
		if err != nil {
			return fmt.Errorf("read migration file %q: %w", version, err)
		}

		// Replace $DIMENSIONS placeholder for vector column definitions (PostgreSQL and SQLite).
		sqlContent := string(content)
		sqlContent = strings.ReplaceAll(sqlContent, "$DIMENSIONS", fmt.Sprintf("%d", dimensions))

		// Execute the migration SQL.
		if _, err := conn.Exec(ctx, sqlContent); err != nil {
			return fmt.Errorf("execute migration %q: %w", version, err)
		}

		// Record successful migration.
		if conn.IsSQLite() {
			if _, err := conn.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES (?)", version); err != nil {
				return fmt.Errorf("record migration %q: %w", version, err)
			}
		} else {
			if _, err := conn.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES ($1)", version); err != nil {
				return fmt.Errorf("record migration %q: %w", version, err)
			}
		}
	}

	return nil
}
