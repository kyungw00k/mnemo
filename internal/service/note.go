package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kyungw00k/mnemo/internal/repository"
)

// NoteService handles business logic for note operations.
type NoteService struct {
	repo      *repository.NoteRepository
	embedding *EmbeddingService
	isSQLite  bool
	ttlDays   int
}

// NewNoteService creates a new NoteService.
func NewNoteService(repo *repository.NoteRepository, embedding *EmbeddingService, isSQLite bool, ttlDays int) *NoteService {
	return &NoteService{
		repo:      repo,
		embedding: embedding,
		isSQLite:  isSQLite,
		ttlDays:   ttlDays,
	}
}

// Save stores a new note with the given tags ([]string marshaled to JSON).
func (s *NoteService) Save(ctx context.Context, host, project, title, content string, tags []string) (*repository.Note, error) {
	// Marshal tags to JSON string.
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("marshal tags: %w", err)
	}

	var embedding []float32

	// Only generate embeddings in PostgreSQL mode.
	if !s.isSQLite && s.embedding != nil {
		embText := title + " " + content
		emb, err := s.embedding.Embed(ctx, embText)
		if err != nil {
			log.Printf("note embedding generation failed (continuing without): %v", err)
		} else {
			embedding = emb
		}
	}

	// Calculate expires_at if TTL is configured.
	var expiresAt *time.Time
	if s.ttlDays > 0 {
		t := time.Now().AddDate(0, 0, s.ttlDays)
		expiresAt = &t
	}

	note, err := s.repo.Insert(ctx, host, project, title, content, string(tagsJSON), embedding, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("save note: %w", err)
	}
	return note, nil
}

// Search searches notes using vector similarity (PostgreSQL) or FTS5 (SQLite).
func (s *NoteService) Search(ctx context.Context, host, project, query string, limit int) ([]repository.NoteSearchResult, error) {
	if s.isSQLite {
		return s.repo.TextSearch(ctx, host, project, query, limit)
	}

	// Try vector search first.
	if s.embedding != nil {
		vec, err := s.embedding.Embed(ctx, query)
		if err == nil && vec != nil {
			results, err := s.repo.VectorSearch(ctx, host, project, vec, limit)
			if err == nil {
				return results, nil
			}
			log.Printf("note vector search failed, falling back to text: %v", err)
		}
	}

	return s.repo.TextSearch(ctx, host, project, query, limit)
}

// List returns notes for a given host and optional project.
func (s *NoteService) List(ctx context.Context, host, project string, limit int) ([]*repository.Note, error) {
	return s.repo.ListByProject(ctx, host, project, limit)
}

// DeleteByID soft-deletes a note by its ID.
func (s *NoteService) DeleteByID(ctx context.Context, id int64) error {
	return s.repo.SoftDeleteByID(ctx, id)
}

// Cleanup hard-deletes all expired notes.
func (s *NoteService) Cleanup(ctx context.Context) (int64, error) {
	return s.repo.HardDeleteExpired(ctx)
}

// ExportAll returns all non-deleted notes for the given host.
func (s *NoteService) ExportAll(ctx context.Context, host string) ([]*repository.Note, error) {
	return s.repo.ExportAll(ctx, host)
}

// BulkImport inserts multiple notes.
func (s *NoteService) BulkImport(ctx context.Context, notes []*repository.Note) error {
	return s.repo.BulkInsert(ctx, notes)
}
