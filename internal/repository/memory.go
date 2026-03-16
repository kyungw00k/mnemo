package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyungw00k/mnemo/internal/db"
)

// Memory represents a stored key-value memory entry.
type Memory struct {
	ID        int64
	Host      string
	Category  string
	Key       string
	Value     string
	Metadata  string
	Embedding []float32 // nil for SQLite
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time // nil = no expiry
}

// MemorySearchResult holds a search result with a similarity score.
type MemorySearchResult struct {
	ID         int64
	Category   string
	Key        string
	Value      string
	Similarity float64
	CreatedAt  time.Time
}

// MemoryRepository provides data access for memories.
type MemoryRepository struct {
	db *db.DBConn
}

// NewMemoryRepository creates a new MemoryRepository.
func NewMemoryRepository(conn *db.DBConn) *MemoryRepository {
	return &MemoryRepository{db: conn}
}

// Upsert inserts or updates a memory entry (conflicts on host+category+key).
func (r *MemoryRepository) Upsert(ctx context.Context, host, category, key, value, metadata string, embedding []float32, expiresAt *time.Time) (*Memory, error) {
	now := time.Now()

	if r.db.IsSQLite() {
		result, err := r.db.Exec(ctx,
			`INSERT INTO memories (host, category, memory_key, memory_value, metadata, expires_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(host, category, memory_key) DO UPDATE SET
			   memory_value = excluded.memory_value,
			   metadata = excluded.metadata,
			   expires_at = excluded.expires_at,
			   updated_at = excluded.updated_at`,
			host, category, key, value, metadata, expiresAt, now,
		)
		if err != nil {
			return nil, fmt.Errorf("upsert memory: %w", err)
		}
		id, _ := result.LastInsertId()
		return &Memory{
			ID:        id,
			Host:      host,
			Category:  category,
			Key:       key,
			Value:     value,
			Metadata:  metadata,
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: expiresAt,
		}, nil
	}

	// PostgreSQL with optional vector embedding.
	var id int64
	if embedding != nil {
		err := r.db.QueryRow(ctx,
			`INSERT INTO memories (host, category, memory_key, memory_value, metadata, embedding, expires_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6::vector, $7, $8)
			 ON CONFLICT(host, category, memory_key) DO UPDATE SET
			   memory_value = EXCLUDED.memory_value,
			   metadata = EXCLUDED.metadata,
			   embedding = EXCLUDED.embedding,
			   expires_at = EXCLUDED.expires_at,
			   updated_at = EXCLUDED.updated_at
			 RETURNING id`,
			host, category, key, value, metadata, vectorString(embedding), expiresAt, now,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert memory with embedding: %w", err)
		}
	} else {
		err := r.db.QueryRow(ctx,
			`INSERT INTO memories (host, category, memory_key, memory_value, metadata, expires_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT(host, category, memory_key) DO UPDATE SET
			   memory_value = EXCLUDED.memory_value,
			   metadata = EXCLUDED.metadata,
			   expires_at = EXCLUDED.expires_at,
			   updated_at = EXCLUDED.updated_at
			 RETURNING id`,
			host, category, key, value, metadata, expiresAt, now,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert memory: %w", err)
		}
	}

	return &Memory{
		ID:        id,
		Host:      host,
		Category:  category,
		Key:       key,
		Value:     value,
		Metadata:  metadata,
		Embedding: embedding,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

// VectorSearch performs a cosine similarity search using pgvector (PostgreSQL only).
func (r *MemoryRepository) VectorSearch(ctx context.Context, host, category string, vector []float32, limit int) ([]MemorySearchResult, error) {
	query := `SELECT id, category, memory_key, memory_value,
	           1-(embedding<=>$1::vector) as similarity, created_at
	           FROM memories
	           WHERE del_yn='N' AND host=$2
	           AND ($3='' OR category=$3)
	           AND embedding IS NOT NULL
	           AND (expires_at IS NULL OR expires_at > NOW())
	           ORDER BY embedding<=>$1::vector
	           LIMIT $4`

	rows, err := r.db.Query(ctx, query, vectorString(vector), host, category, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search memories: %w", err)
	}
	defer rows.Close()
	return scanMemoryResults(rows)
}

// TextSearch performs a full-text search using FTS5 (SQLite) or LIKE (PostgreSQL fallback).
func (r *MemoryRepository) TextSearch(ctx context.Context, host, category, query string, limit int) ([]MemorySearchResult, error) {
	if r.db.IsSQLite() {
		sqlQuery := `SELECT m.id, m.category, m.memory_key, m.memory_value, 1.0 as similarity, m.created_at
		             FROM memories m
		             JOIN memories_fts f ON m.id = f.rowid
		             WHERE f.memories_fts MATCH ?
		             AND m.del_yn='N' AND m.host=?
		             AND (m.expires_at IS NULL OR m.expires_at > datetime('now'))`
		args := []any{query, host}
		if category != "" {
			sqlQuery += " AND m.category=?"
			args = append(args, category)
		}
		sqlQuery += " ORDER BY rank LIMIT ?"
		args = append(args, limit)

		rows, err := r.db.Query(ctx, sqlQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("fts5 search memories: %w", err)
		}
		defer rows.Close()
		return scanMemoryResults(rows)
	}

	// PostgreSQL LIKE fallback.
	like := "%" + query + "%"
	sqlQuery := `SELECT id, category, memory_key, memory_value, 1.0 as similarity, created_at
	             FROM memories
	             WHERE del_yn='N' AND host=$1
	             AND ($2='' OR category=$2)
	             AND (memory_key ILIKE $3 OR memory_value ILIKE $3)
	             AND (expires_at IS NULL OR expires_at > NOW())
	             ORDER BY updated_at DESC
	             LIMIT $4`
	rows, err := r.db.Query(ctx, sqlQuery, host, category, like, limit)
	if err != nil {
		return nil, fmt.Errorf("like search memories: %w", err)
	}
	defer rows.Close()
	return scanMemoryResults(rows)
}

// ListByCategory lists memories for a given host and category.
func (r *MemoryRepository) ListByCategory(ctx context.Context, host, category string, limit int) ([]*Memory, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if r.db.IsSQLite() {
		if category != "" {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
				 FROM memories WHERE del_yn='N' AND host=? AND category=?
				 AND (expires_at IS NULL OR expires_at > datetime('now'))
				 ORDER BY updated_at DESC LIMIT ?`,
				host, category, limit)
		} else {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
				 FROM memories WHERE del_yn='N' AND host=?
				 AND (expires_at IS NULL OR expires_at > datetime('now'))
				 ORDER BY updated_at DESC LIMIT ?`,
				host, limit)
		}
	} else {
		if category != "" {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
				 FROM memories WHERE del_yn='N' AND host=$1 AND category=$2
				 AND (expires_at IS NULL OR expires_at > NOW())
				 ORDER BY updated_at DESC LIMIT $3`,
				host, category, limit)
		} else {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
				 FROM memories WHERE del_yn='N' AND host=$1
				 AND (expires_at IS NULL OR expires_at > NOW())
				 ORDER BY updated_at DESC LIMIT $2`,
				host, limit)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var results []*Memory
	for rows.Next() {
		m := &Memory{}
		if err := rows.Scan(&m.ID, &m.Host, &m.Category, &m.Key, &m.Value, &m.Metadata, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// ListCategories returns distinct categories for a given host.
func (r *MemoryRepository) ListCategories(ctx context.Context, host string) ([]string, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if r.db.IsSQLite() {
		rows, err = r.db.Query(ctx,
			`SELECT DISTINCT category FROM memories WHERE del_yn='N' AND host=? ORDER BY category`,
			host)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT DISTINCT category FROM memories WHERE del_yn='N' AND host=$1 ORDER BY category`,
			host)
	}
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, cat)
	}
	return cats, rows.Err()
}

// SoftDeleteByID marks a memory as deleted by its ID.
func (r *MemoryRepository) SoftDeleteByID(ctx context.Context, id int64) error {
	var err error
	if r.db.IsSQLite() {
		_, err = r.db.Exec(ctx, `UPDATE memories SET del_yn='Y', updated_at=datetime('now') WHERE id=?`, id)
	} else {
		_, err = r.db.Exec(ctx, `UPDATE memories SET del_yn='Y', updated_at=NOW() WHERE id=$1`, id)
	}
	if err != nil {
		return fmt.Errorf("soft delete memory by id: %w", err)
	}
	return nil
}

// SoftDeleteByKey marks a memory as deleted by host+category+key.
func (r *MemoryRepository) SoftDeleteByKey(ctx context.Context, host, category, key string) error {
	var err error
	if r.db.IsSQLite() {
		_, err = r.db.Exec(ctx,
			`UPDATE memories SET del_yn='Y', updated_at=datetime('now') WHERE host=? AND category=? AND memory_key=?`,
			host, category, key)
	} else {
		_, err = r.db.Exec(ctx,
			`UPDATE memories SET del_yn='Y', updated_at=NOW() WHERE host=$1 AND category=$2 AND memory_key=$3`,
			host, category, key)
	}
	if err != nil {
		return fmt.Errorf("soft delete memory by key: %w", err)
	}
	return nil
}

// HardDeleteExpired permanently deletes expired memories (del_yn='N' only).
func (r *MemoryRepository) HardDeleteExpired(ctx context.Context) (int64, error) {
	var result sql.Result
	var err error
	if r.db.IsSQLite() {
		result, err = r.db.Exec(ctx,
			`DELETE FROM memories WHERE del_yn='N' AND expires_at IS NOT NULL AND expires_at <= datetime('now')`)
	} else {
		result, err = r.db.Exec(ctx,
			`DELETE FROM memories WHERE del_yn='N' AND expires_at IS NOT NULL AND expires_at <= NOW()`)
	}
	if err != nil {
		return 0, fmt.Errorf("hard delete expired memories: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// ExportAll returns all non-deleted memories for a given host without limit.
func (r *MemoryRepository) ExportAll(ctx context.Context, host string) ([]*Memory, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if r.db.IsSQLite() {
		rows, err = r.db.Query(ctx,
			`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
			 FROM memories WHERE del_yn='N' AND host=?
			 ORDER BY category, memory_key`,
			host)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, host, category, memory_key, memory_value, COALESCE(metadata,''), created_at, updated_at
			 FROM memories WHERE del_yn='N' AND host=$1
			 ORDER BY category, memory_key`,
			host)
	}
	if err != nil {
		return nil, fmt.Errorf("export memories: %w", err)
	}
	defer rows.Close()

	var results []*Memory
	for rows.Next() {
		m := &Memory{}
		if err := rows.Scan(&m.ID, &m.Host, &m.Category, &m.Key, &m.Value, &m.Metadata, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan memory export: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// BulkUpsert upserts multiple memories by iterating and calling Upsert for each.
func (r *MemoryRepository) BulkUpsert(ctx context.Context, memories []*Memory) error {
	for _, m := range memories {
		if _, err := r.Upsert(ctx, m.Host, m.Category, m.Key, m.Value, m.Metadata, m.Embedding, m.ExpiresAt); err != nil {
			return fmt.Errorf("bulk upsert memory [%s/%s]: %w", m.Category, m.Key, err)
		}
	}
	return nil
}

// scanMemoryResults scans *sql.Rows into a []MemorySearchResult slice.
func scanMemoryResults(rows *sql.Rows) ([]MemorySearchResult, error) {
	var results []MemorySearchResult
	for rows.Next() {
		var r MemorySearchResult
		if err := rows.Scan(&r.ID, &r.Category, &r.Key, &r.Value, &r.Similarity, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan memory result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// vectorString converts a float32 slice to the PostgreSQL vector literal format.
func vectorString(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%g", f)
	}
	s += "]"
	return s
}
