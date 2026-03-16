package repository_test

import (
	"context"
	"testing"

	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/testutil"
)

const testHost = "test-host"

func TestMemoryRepository_Upsert(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	mem, err := repo.Upsert(ctx, testHost, "decision", "db-choice", "use sqlite", "", nil, nil)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if mem.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if mem.Key != "db-choice" {
		t.Errorf("Key = %q, want %q", mem.Key, "db-choice")
	}

	// Update the same key — should not error.
	mem2, err := repo.Upsert(ctx, testHost, "decision", "db-choice", "updated value", "", nil, nil)
	if err != nil {
		t.Fatalf("Upsert (update): %v", err)
	}
	if mem2.Value != "updated value" {
		t.Errorf("updated Value = %q, want %q", mem2.Value, "updated value")
	}
}

func TestMemoryRepository_TextSearch(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, err := repo.Upsert(ctx, testHost, "note", "animal", "the quick brown fox", "", nil, nil)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	results, err := repo.TextSearch(ctx, testHost, "", "fox", 10)
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) == 0 {
		t.Error("TextSearch returned no results, expected at least 1")
	}
	if results[0].Value != "the quick brown fox" {
		t.Errorf("Value = %q, want %q", results[0].Value, "the quick brown fox")
	}
}

func TestMemoryRepository_TextSearch_CategoryFilter(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, _ = repo.Upsert(ctx, testHost, "bug", "issue-1", "elephant memory leak", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "decision", "choice-1", "elephant was chosen", "", nil, nil)

	results, err := repo.TextSearch(ctx, testHost, "bug", "elephant", 10)
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with category filter, got %d", len(results))
	}
	if results[0].Category != "bug" {
		t.Errorf("Category = %q, want %q", results[0].Category, "bug")
	}
}

func TestMemoryRepository_ListByCategory(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, _ = repo.Upsert(ctx, testHost, "config", "key1", "val1", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "config", "key2", "val2", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "other", "key3", "val3", "", nil, nil)

	results, err := repo.ListByCategory(ctx, testHost, "config", 10)
	if err != nil {
		t.Fatalf("ListByCategory: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestMemoryRepository_SoftDelete_ByID(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	mem, _ := repo.Upsert(ctx, testHost, "decision", "to-delete", "value", "", nil, nil)

	if err := repo.SoftDeleteByID(ctx, mem.ID); err != nil {
		t.Fatalf("SoftDeleteByID: %v", err)
	}

	results, _ := repo.ListByCategory(ctx, testHost, "decision", 10)
	for _, r := range results {
		if r.ID == mem.ID {
			t.Error("soft-deleted memory should not appear in list")
		}
	}
}

func TestMemoryRepository_SoftDelete_ByKey(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, _ = repo.Upsert(ctx, testHost, "convention", "style", "gofmt", "", nil, nil)

	if err := repo.SoftDeleteByKey(ctx, testHost, "convention", "style"); err != nil {
		t.Fatalf("SoftDeleteByKey: %v", err)
	}

	results, _ := repo.ListByCategory(ctx, testHost, "convention", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 results after soft delete, got %d", len(results))
	}
}

func TestMemoryRepository_ListCategories(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, _ = repo.Upsert(ctx, testHost, "alpha", "k1", "v1", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "beta", "k2", "v2", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "alpha", "k3", "v3", "", nil, nil)

	cats, err := repo.ListCategories(ctx, testHost)
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 2 {
		t.Errorf("got %d categories, want 2", len(cats))
	}
}

func TestMemoryRepository_ExportAll(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	ctx := context.Background()

	_, _ = repo.Upsert(ctx, testHost, "cat", "k1", "v1", "", nil, nil)
	_, _ = repo.Upsert(ctx, testHost, "cat", "k2", "v2", "", nil, nil)

	memories, err := repo.ExportAll(ctx, testHost)
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("got %d memories, want 2", len(memories))
	}
}
