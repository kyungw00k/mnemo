package repository_test

import (
	"context"
	"testing"

	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/testutil"
)

func TestNoteRepository_Insert(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	note, err := repo.Insert(ctx, testHost, "myproject", "Setup Guide", "Run make build", `["setup","go"]`, nil, nil)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if note.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if note.Title != "Setup Guide" {
		t.Errorf("Title = %q, want %q", note.Title, "Setup Guide")
	}
	if note.Project != "myproject" {
		t.Errorf("Project = %q, want %q", note.Project, "myproject")
	}
}

func TestNoteRepository_TextSearch(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	_, err := repo.Insert(ctx, testHost, "proj", "Architecture", "we use hexagonal architecture pattern", "", nil, nil)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	results, err := repo.TextSearch(ctx, testHost, "", "hexagonal", 10)
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) == 0 {
		t.Error("TextSearch returned no results, expected at least 1")
	}
	if results[0].Title != "Architecture" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Architecture")
	}
}

func TestNoteRepository_TextSearch_ProjectFilter(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	_, _ = repo.Insert(ctx, testHost, "project-a", "Deploy", "kubernetes deployment steps", "", nil, nil)
	_, _ = repo.Insert(ctx, testHost, "project-b", "Deploy", "kubernetes deployment steps", "", nil, nil)

	results, err := repo.TextSearch(ctx, testHost, "project-a", "kubernetes", 10)
	if err != nil {
		t.Fatalf("TextSearch: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with project filter, got %d", len(results))
	}
	if results[0].Project != "project-a" {
		t.Errorf("Project = %q, want %q", results[0].Project, "project-a")
	}
}

func TestNoteRepository_ListByProject(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	_, _ = repo.Insert(ctx, testHost, "proj", "Note 1", "content 1", "", nil, nil)
	_, _ = repo.Insert(ctx, testHost, "proj", "Note 2", "content 2", "", nil, nil)
	_, _ = repo.Insert(ctx, testHost, "other", "Note 3", "content 3", "", nil, nil)

	notes, err := repo.ListByProject(ctx, testHost, "proj", 10)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}
}

func TestNoteRepository_SoftDelete(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	note, _ := repo.Insert(ctx, testHost, "proj", "To Delete", "content", "", nil, nil)

	if err := repo.SoftDeleteByID(ctx, note.ID); err != nil {
		t.Fatalf("SoftDeleteByID: %v", err)
	}

	notes, _ := repo.ListByProject(ctx, testHost, "proj", 10)
	for _, n := range notes {
		if n.ID == note.ID {
			t.Error("soft-deleted note should not appear in list")
		}
	}
}

func TestNoteRepository_ExportAll(t *testing.T) {
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	ctx := context.Background()

	_, _ = repo.Insert(ctx, testHost, "proj", "Note A", "content a", "", nil, nil)
	_, _ = repo.Insert(ctx, testHost, "proj", "Note B", "content b", "", nil, nil)

	notes, err := repo.ExportAll(ctx, testHost)
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}
}
