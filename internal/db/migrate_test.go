package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kyungw00k/mnemo/internal/db"
	"github.com/kyungw00k/mnemo/internal/migrations"
)

func TestMigrate_SQLite_TablesExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	defer conn.StdlibDB().Close()

	sqliteFS, err := migrations.SQLiteFS()
	if err != nil {
		t.Fatalf("SQLiteFS: %v", err)
	}

	if err := db.Migrate(context.Background(), conn, sqliteFS, 4); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify core tables exist.
	tables := []string{"memories", "notes", "schema_migrations", "memories_fts", "notes_fts"}
	for _, table := range tables {
		row := conn.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','shadow') AND name=?", table)
		var count int
		if err := row.Scan(&count); err != nil {
			t.Errorf("check table %q: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("table %q not found after migration", table)
		}
	}

	// Verify vec virtual tables exist.
	vecTables := []string{"vec_memories", "vec_notes"}
	for _, table := range vecTables {
		row := conn.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table)
		var count int
		if err := row.Scan(&count); err != nil {
			t.Errorf("check vec table %q: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("vec table %q not found after migration", table)
		}
	}
}

func TestMigrate_SQLite_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.NewSQLite(path)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	defer conn.StdlibDB().Close()

	sqliteFS, err := migrations.SQLiteFS()
	if err != nil {
		t.Fatalf("SQLiteFS: %v", err)
	}

	// Run twice — should not error.
	for i := range 2 {
		if err := db.Migrate(context.Background(), conn, sqliteFS, 4); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}
}
