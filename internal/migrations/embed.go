// Package migrations provides embedded SQL migration files for both
// PostgreSQL and SQLite database backends.
package migrations

import (
	"embed"
	"io/fs"
)

//go:embed postgres/*.sql
var postgresFS embed.FS

//go:embed sqlite/*.sql
var sqliteFS embed.FS

// PostgresFS returns the embedded PostgreSQL migration files as a flat FS
// (files accessible by their base name).
func PostgresFS() (fs.FS, error) {
	return fs.Sub(postgresFS, "postgres")
}

// SQLiteFS returns the embedded SQLite migration files as a flat FS.
func SQLiteFS() (fs.FS, error) {
	return fs.Sub(sqliteFS, "sqlite")
}
