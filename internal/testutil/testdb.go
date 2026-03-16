// Package testutil provides shared test helpers for database setup.
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kyungw00k/mnemo/internal/db"
	"github.com/kyungw00k/mnemo/internal/migrations"
)

// NewSQLiteConn creates a temporary SQLite database with all migrations applied.
// The database file is cleaned up automatically when the test ends.
func NewSQLiteConn(t *testing.T) *db.DBConn {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.NewSQLite(path)
	if err != nil {
		t.Fatalf("testutil: create sqlite db: %v", err)
	}
	t.Cleanup(func() { conn.StdlibDB().Close() })

	sqliteFS, err := migrations.SQLiteFS()
	if err != nil {
		t.Fatalf("testutil: get sqlite migrations fs: %v", err)
	}

	if err := db.Migrate(context.Background(), conn, sqliteFS, 4); err != nil {
		t.Fatalf("testutil: run migrations: %v", err)
	}

	return conn
}
