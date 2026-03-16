package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kyungw00k/mnemo/internal/db"
)

// Note represents a stored structured note.
type Note struct {
	ID        int64
	Host      string
	Project   string
	Title     string
	Content   string
	Tags      string // JSON string: []string marshaled to JSON
	Embedding []float32
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt *time.Time // nil = no expiry
}

// NoteSearchResult holds a note search result with a similarity score.
type NoteSearchResult struct {
	ID         int64
	Project    string
	Title      string
	Content    string
	Tags       string
	Similarity float64
	CreatedAt  time.Time
}

// NoteRepository provides data access for notes.
type NoteRepository struct {
	db *db.DBConn
}

// NewNoteRepository creates a new NoteRepository.
func NewNoteRepository(conn *db.DBConn) *NoteRepository {
	return &NoteRepository{db: conn}
}

// Insert saves a new note to the database.
func (r *NoteRepository) Insert(ctx context.Context, host, project, title, content, tags string, embedding []float32, expiresAt *time.Time) (*Note, error) {
	now := time.Now()

	if r.db.IsSQLite() {
		result, err := r.db.Exec(ctx,
			`INSERT INTO notes (host, project, title, content, tags, expires_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			host, project, title, content, tags, expiresAt, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("insert note: %w", err)
		}
		id, _ := result.LastInsertId()
		return &Note{
			ID:        id,
			Host:      host,
			Project:   project,
			Title:     title,
			Content:   content,
			Tags:      tags,
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: expiresAt,
		}, nil
	}

	// PostgreSQL with optional vector embedding.
	var id int64
	if embedding != nil {
		err := r.db.QueryRow(ctx,
			`INSERT INTO notes (host, project, title, content, tags, embedding, expires_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6::vector, $7, $8, $9)
			 RETURNING id`,
			host, project, title, content, tags, vectorString(embedding), expiresAt, now, now,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert note with embedding: %w", err)
		}
	} else {
		err := r.db.QueryRow(ctx,
			`INSERT INTO notes (host, project, title, content, tags, expires_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING id`,
			host, project, title, content, tags, expiresAt, now, now,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert note: %w", err)
		}
	}

	return &Note{
		ID:        id,
		Host:      host,
		Project:   project,
		Title:     title,
		Content:   content,
		Tags:      tags,
		Embedding: embedding,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

// VectorSearch performs a cosine similarity search on notes (PostgreSQL only).
func (r *NoteRepository) VectorSearch(ctx context.Context, host, project string, vector []float32, limit int) ([]NoteSearchResult, error) {
	query := `SELECT id, COALESCE(project,''), title, content, COALESCE(tags,''),
	           1-(embedding<=>$1::vector) as similarity, created_at
	           FROM notes
	           WHERE del_yn='N' AND host=$2
	           AND ($3='' OR project=$3)
	           AND embedding IS NOT NULL
	           AND (expires_at IS NULL OR expires_at > NOW())
	           ORDER BY embedding<=>$1::vector
	           LIMIT $4`

	rows, err := r.db.Query(ctx, query, vectorString(vector), host, project, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search notes: %w", err)
	}
	defer rows.Close()
	return scanNoteResults(rows)
}

// TextSearch performs a full-text search on notes.
func (r *NoteRepository) TextSearch(ctx context.Context, host, project, query string, limit int) ([]NoteSearchResult, error) {
	if r.db.IsSQLite() {
		sqlQuery := `SELECT n.id, COALESCE(n.project,''), n.title, n.content, COALESCE(n.tags,''), 1.0 as similarity, n.created_at
		             FROM notes n
		             JOIN notes_fts f ON n.id = f.rowid
		             WHERE f.notes_fts MATCH ?
		             AND n.del_yn='N' AND n.host=?
		             AND (n.expires_at IS NULL OR n.expires_at > datetime('now'))`
		args := []any{query, host}
		if project != "" {
			sqlQuery += " AND n.project=?"
			args = append(args, project)
		}
		sqlQuery += " ORDER BY rank LIMIT ?"
		args = append(args, limit)

		rows, err := r.db.Query(ctx, sqlQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("fts5 search notes: %w", err)
		}
		defer rows.Close()
		return scanNoteResults(rows)
	}

	// PostgreSQL LIKE fallback.
	like := "%" + query + "%"
	sqlQuery := `SELECT id, COALESCE(project,''), title, content, COALESCE(tags,''), 1.0 as similarity, created_at
	             FROM notes
	             WHERE del_yn='N' AND host=$1
	             AND ($2='' OR project=$2)
	             AND (title ILIKE $3 OR content ILIKE $3)
	             AND (expires_at IS NULL OR expires_at > NOW())
	             ORDER BY updated_at DESC
	             LIMIT $4`
	rows, err := r.db.Query(ctx, sqlQuery, host, project, like, limit)
	if err != nil {
		return nil, fmt.Errorf("like search notes: %w", err)
	}
	defer rows.Close()
	return scanNoteResults(rows)
}

// ListByProject lists notes for a given host and optional project.
func (r *NoteRepository) ListByProject(ctx context.Context, host, project string, limit int) ([]*Note, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if r.db.IsSQLite() {
		if project != "" {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
				 FROM notes WHERE del_yn='N' AND host=? AND project=?
				 AND (expires_at IS NULL OR expires_at > datetime('now'))
				 ORDER BY updated_at DESC LIMIT ?`,
				host, project, limit)
		} else {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
				 FROM notes WHERE del_yn='N' AND host=?
				 AND (expires_at IS NULL OR expires_at > datetime('now'))
				 ORDER BY updated_at DESC LIMIT ?`,
				host, limit)
		}
	} else {
		if project != "" {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
				 FROM notes WHERE del_yn='N' AND host=$1 AND project=$2
				 AND (expires_at IS NULL OR expires_at > NOW())
				 ORDER BY updated_at DESC LIMIT $3`,
				host, project, limit)
		} else {
			rows, err = r.db.Query(ctx,
				`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
				 FROM notes WHERE del_yn='N' AND host=$1
				 AND (expires_at IS NULL OR expires_at > NOW())
				 ORDER BY updated_at DESC LIMIT $2`,
				host, limit)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var results []*Note
	for rows.Next() {
		n := &Note{}
		if err := rows.Scan(&n.ID, &n.Host, &n.Project, &n.Title, &n.Content, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

// SoftDeleteByID marks a note as deleted by its ID.
func (r *NoteRepository) SoftDeleteByID(ctx context.Context, id int64) error {
	var err error
	if r.db.IsSQLite() {
		_, err = r.db.Exec(ctx, `UPDATE notes SET del_yn='Y', updated_at=datetime('now') WHERE id=?`, id)
	} else {
		_, err = r.db.Exec(ctx, `UPDATE notes SET del_yn='Y', updated_at=NOW() WHERE id=$1`, id)
	}
	if err != nil {
		return fmt.Errorf("soft delete note: %w", err)
	}
	return nil
}

// HardDeleteExpired permanently deletes expired notes (del_yn='N' only).
func (r *NoteRepository) HardDeleteExpired(ctx context.Context) (int64, error) {
	var result sql.Result
	var err error
	if r.db.IsSQLite() {
		result, err = r.db.Exec(ctx,
			`DELETE FROM notes WHERE del_yn='N' AND expires_at IS NOT NULL AND expires_at <= datetime('now')`)
	} else {
		result, err = r.db.Exec(ctx,
			`DELETE FROM notes WHERE del_yn='N' AND expires_at IS NOT NULL AND expires_at <= NOW()`)
	}
	if err != nil {
		return 0, fmt.Errorf("hard delete expired notes: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// ExportAll returns all non-deleted notes for a given host without limit.
func (r *NoteRepository) ExportAll(ctx context.Context, host string) ([]*Note, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if r.db.IsSQLite() {
		rows, err = r.db.Query(ctx,
			`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
			 FROM notes WHERE del_yn='N' AND host=?
			 ORDER BY updated_at DESC`,
			host)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, host, COALESCE(project,''), title, content, COALESCE(tags,''), created_at, updated_at
			 FROM notes WHERE del_yn='N' AND host=$1
			 ORDER BY updated_at DESC`,
			host)
	}
	if err != nil {
		return nil, fmt.Errorf("export notes: %w", err)
	}
	defer rows.Close()

	var results []*Note
	for rows.Next() {
		n := &Note{}
		if err := rows.Scan(&n.ID, &n.Host, &n.Project, &n.Title, &n.Content, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan note export: %w", err)
		}
		results = append(results, n)
	}
	return results, rows.Err()
}

// BulkInsert inserts multiple notes by iterating and calling Insert for each.
func (r *NoteRepository) BulkInsert(ctx context.Context, notes []*Note) error {
	for _, n := range notes {
		if _, err := r.Insert(ctx, n.Host, n.Project, n.Title, n.Content, n.Tags, n.Embedding, n.ExpiresAt); err != nil {
			return fmt.Errorf("bulk insert note [%s]: %w", n.Title, err)
		}
	}
	return nil
}

// scanNoteResults scans *sql.Rows into a []NoteSearchResult slice.
func scanNoteResults(rows *sql.Rows) ([]NoteSearchResult, error) {
	var results []NoteSearchResult
	for rows.Next() {
		var r NoteSearchResult
		if err := rows.Scan(&r.ID, &r.Project, &r.Title, &r.Content, &r.Tags, &r.Similarity, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan note result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
