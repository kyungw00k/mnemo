package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// NewPostgres creates a new DBConn backed by PostgreSQL using pgxpool.
// It registers pgvector types on each new connection.
func NewPostgres(ctx context.Context, url string) (*DBConn, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	// Register pgvector types on every new connection in the pool.
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify connection.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// Create a database/sql adapter for unified query interface.
	stdDB := stdlib.OpenDBFromPool(pool)

	return &DBConn{
		postgres: pool,
		stdlibDB: stdDB,
		dbType:   "postgres",
	}, nil
}
