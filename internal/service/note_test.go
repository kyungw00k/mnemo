package service_test

import (
	"context"
	"testing"

	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
	"github.com/kyungw00k/mnemo/internal/testutil"
)

func newNoteService(t *testing.T) *service.NoteService {
	t.Helper()
	conn := testutil.NewSQLiteConn(t)
	repo := repository.NewNoteRepository(conn)
	return service.NewNoteService(repo, nil, 0)
}

func TestNoteService_Save(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	note, err := svc.Save(ctx, testHost, "mnemo", "Build Guide", "run make build", []string{"build", "go"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if note.Title != "Build Guide" {
		t.Errorf("Title = %q, want %q", note.Title, "Build Guide")
	}
	if note.Project != "mnemo" {
		t.Errorf("Project = %q, want %q", note.Project, "mnemo")
	}
}

func TestNoteService_Save_EmptyTags(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	note, err := svc.Save(ctx, testHost, "", "Untitled", "some content", nil)
	if err != nil {
		t.Fatalf("Save with nil tags: %v", err)
	}
	if note.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestNoteService_Search_FTS5Fallback(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "proj", "Deployment", "kubernetes helm chart deployment", []string{"k8s"})

	results, err := svc.Search(ctx, testHost, "", "kubernetes", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("Search returned no results, expected at least 1")
	}
}

func TestNoteService_Search_ProjectFilter(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "proj-a", "Note", "migration strategy overview", nil)
	_, _ = svc.Save(ctx, testHost, "proj-b", "Note", "migration strategy overview", nil)

	results, err := svc.Search(ctx, testHost, "proj-a", "migration", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d results with project filter, want 1", len(results))
	}
}

func TestNoteService_List(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	_, _ = svc.Save(ctx, testHost, "proj", "Note 1", "content", nil)
	_, _ = svc.Save(ctx, testHost, "proj", "Note 2", "content", nil)
	_, _ = svc.Save(ctx, testHost, "other", "Note 3", "content", nil)

	notes, err := svc.List(ctx, testHost, "proj", 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 2 {
		t.Errorf("got %d notes, want 2", len(notes))
	}
}

func TestNoteService_DeleteByID(t *testing.T) {
	svc := newNoteService(t)
	ctx := context.Background()

	note, _ := svc.Save(ctx, testHost, "proj", "To Delete", "content", nil)
	if err := svc.DeleteByID(ctx, note.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	notes, _ := svc.List(ctx, testHost, "proj", 10)
	if len(notes) != 0 {
		t.Error("deleted note should not appear in list")
	}
}
