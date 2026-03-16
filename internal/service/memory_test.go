package service_test

import (
	"context"
	"testing"

	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
	"github.com/kyungw00k/mnemo/internal/testutil"
)

const testHost = "test-host"

func newMemoryService(t *testing.T) *service.MemoryService {
	t.Helper()
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewMemoryRepository(conn)
	return service.NewMemoryService(repo, nil, 0)
}

func TestMemoryService_Save(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	mem, err := svc.Save(ctx, testHost, "decision", "db-choice", "use sqlite", "")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if mem.Key != "db-choice" {
		t.Errorf("Key = %q, want %q", mem.Key, "db-choice")
	}
}

func TestMemoryService_Save_Update(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "config", "timeout", "30s", "")
	mem, err := svc.Save(ctx, testHost, "config", "timeout", "60s", "")
	if err != nil {
		t.Fatalf("Save (update): %v", err)
	}
	if mem.Value != "60s" {
		t.Errorf("Value = %q, want %q", mem.Value, "60s")
	}
}

func TestMemoryService_Search_FTS5Fallback(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "bug", "oom", "out of memory crash", "")

	results, err := svc.Search(ctx, testHost, "", "memory crash", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("Search returned no results, expected at least 1")
	}
}

func TestMemoryService_Search_CategoryFilter(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "bug", "issue-1", "goroutine leak detected", "")
	_, _ = svc.Save(ctx, testHost, "decision", "choice-1", "goroutine pool selected", "")

	results, err := svc.Search(ctx, testHost, "bug", "goroutine", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results with category filter, want 1", len(results))
	}
}

func TestMemoryService_List(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "convention", "naming", "use snake_case", "")
	_, _ = svc.Save(ctx, testHost, "convention", "imports", "use goimports", "")

	memories, err := svc.List(ctx, testHost, "convention", 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("got %d memories, want 2", len(memories))
	}
}

func TestMemoryService_Categories(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "alpha", "k1", "v1", "")
	_, _ = svc.Save(ctx, testHost, "beta", "k2", "v2", "")

	cats, err := svc.Categories(ctx, testHost)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}
	if len(cats) != 2 {
		t.Errorf("got %d categories, want 2", len(cats))
	}
}

func TestMemoryService_DeleteByID(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	mem, _ := svc.Save(ctx, testHost, "tmp", "key", "value", "")
	if err := svc.DeleteByID(ctx, mem.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	list, _ := svc.List(ctx, testHost, "tmp", 10)
	if len(list) != 0 {
		t.Error("deleted memory should not appear in list")
	}
}

func TestMemoryService_DeleteByKey(t *testing.T) {
	svc := newMemoryService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "preference", "editor", "neovim", "")
	if err := svc.DeleteByKey(ctx, testHost, "preference", "editor"); err != nil {
		t.Fatalf("DeleteByKey: %v", err)
	}

	list, _ := svc.List(ctx, testHost, "preference", 10)
	if len(list) != 0 {
		t.Error("deleted memory should not appear in list")
	}
}
