package db

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConn wraps either a pgxpool.Pool (PostgreSQL, accessed via stdlib adapter)
// or *sql.DB (SQLite) with a unified database/sql interface.
type DBConn struct {
	postgres *pgxpool.Pool
	stdlibDB *sql.DB // stdlib adapter for postgres OR the sqlite *sql.DB
	dbType   string  // "postgres" or "sqlite"
}

// IsSQLite returns true if the connection is backed by SQLite.
func (d *DBConn) IsSQLite() bool {
	return d.dbType == "sqlite"
}

// IsPostgres returns true if the connection is backed by PostgreSQL.
func (d *DBConn) IsPostgres() bool {
	return d.dbType == "postgres"
}

// Exec executes a statement that does not return rows.
func (d *DBConn) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.stdlibDB.ExecContext(ctx, query, args...)
}

// Query executes a query that returns multiple rows.
func (d *DBConn) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.stdlibDB.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row.
func (d *DBConn) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.stdlibDB.QueryRowContext(ctx, query, args...)
}

// PostgresPool returns the underlying pgxpool.Pool (nil if SQLite).
func (d *DBConn) PostgresPool() *pgxpool.Pool {
	return d.postgres
}

// StdlibDB returns the underlying *sql.DB (works for both drivers).
func (d *DBConn) StdlibDB() *sql.DB {
	return d.stdlibDB
}
