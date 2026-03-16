package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kyungw00k/mnemo/internal/repository"
)

// MemoryService handles business logic for memory operations.
type MemoryService struct {
	repo      *repository.MemoryRepository
	embedding *EmbeddingService
	isSQLite  bool
	ttlDays   int
}

// NewMemoryService creates a new MemoryService.
func NewMemoryService(repo *repository.MemoryRepository, embedding *EmbeddingService, isSQLite bool, ttlDays int) *MemoryService {
	return &MemoryService{
		repo:      repo,
		embedding: embedding,
		isSQLite:  isSQLite,
		ttlDays:   ttlDays,
	}
}

// Save upserts a memory entry, optionally generating an embedding for PostgreSQL.
func (s *MemoryService) Save(ctx context.Context, host, category, key, value, metadata string) (*repository.Memory, error) {
	var embedding []float32

	// Only generate embeddings in PostgreSQL mode.
	if !s.isSQLite && s.embedding != nil {
		emb, err := s.embedding.Embed(ctx, value)
		if err != nil {
			log.Printf("embedding generation failed (continuing without): %v", err)
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

	mem, err := s.repo.Upsert(ctx, host, category, key, value, metadata, embedding, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("save memory: %w", err)
	}
	return mem, nil
}

// Search searches memories using vector similarity (PostgreSQL) or FTS5 (SQLite).
// Falls back to text search if embedding fails.
func (s *MemoryService) Search(ctx context.Context, host, category, query string, limit int) ([]repository.MemorySearchResult, error) {
	if s.isSQLite {
		return s.repo.TextSearch(ctx, host, category, query, limit)
	}

	// Try vector search first.
	if s.embedding != nil {
		vec, err := s.embedding.Embed(ctx, query)
		if err == nil && vec != nil {
			results, err := s.repo.VectorSearch(ctx, host, category, vec, limit)
			if err == nil {
				return results, nil
			}
			log.Printf("vector search failed, falling back to text: %v", err)
		}
	}

	// Fall back to text search.
	return s.repo.TextSearch(ctx, host, category, query, limit)
}

// List returns memories for a given host and category.
func (s *MemoryService) List(ctx context.Context, host, category string, limit int) ([]*repository.Memory, error) {
	return s.repo.ListByCategory(ctx, host, category, limit)
}

// Categories returns distinct categories for a given host.
func (s *MemoryService) Categories(ctx context.Context, host string) ([]string, error) {
	return s.repo.ListCategories(ctx, host)
}

// DeleteByID soft-deletes a memory by its ID.
func (s *MemoryService) DeleteByID(ctx context.Context, id int64) error {
	return s.repo.SoftDeleteByID(ctx, id)
}

// DeleteByKey soft-deletes a memory by host+category+key.
func (s *MemoryService) DeleteByKey(ctx context.Context, host, category, key string) error {
	return s.repo.SoftDeleteByKey(ctx, host, category, key)
}

// Cleanup hard-deletes all expired memories.
func (s *MemoryService) Cleanup(ctx context.Context) (int64, error) {
	return s.repo.HardDeleteExpired(ctx)
}

// ExportAll returns all non-deleted memories for the given host.
func (s *MemoryService) ExportAll(ctx context.Context, host string) ([]*repository.Memory, error) {
	return s.repo.ExportAll(ctx, host)
}

// BulkImport upserts multiple memories.
func (s *MemoryService) BulkImport(ctx context.Context, memories []*repository.Memory) error {
	return s.repo.BulkUpsert(ctx, memories)
}
