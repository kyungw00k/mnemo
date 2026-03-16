package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/service"
)

func registerNoteTools(s *server.MCPServer, noteSvc *service.NoteService, hostID string) {
	// note_save
	noteSaveTool := mcp.NewTool("note_save",
		mcp.WithDescription("Save a structured note with optional tags and project"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Note title")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Note content")),
		mcp.WithString("project", mcp.Description("Optional project name to group notes")),
		mcp.WithArray("tags", mcp.Description("Optional list of tags")),
	)
	s.AddTool(noteSaveTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title := req.GetString("title", "")
		content := req.GetString("content", "")
		project := req.GetString("project", "")
		tags := req.GetStringSlice("tags", nil)

		if title == "" || content == "" {
			return mcp.NewToolResultError("title and content are required"), nil
		}

		note, err := noteSvc.Save(ctx, hostID, project, title, content, tags)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to save note: %v", err)), nil
		}

		var projectInfo string
		if project != "" {
			projectInfo = fmt.Sprintf(" [%s]", project)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Saved note%s: %q (id=%d)", projectInfo, title, note.ID)), nil
	})

	// note_search
	noteSearchTool := mcp.NewTool("note_search",
		mcp.WithDescription("Search notes using semantic similarity or full-text search"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("project", mcp.Description("Filter by project (optional)")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(10)),
	)
	s.AddTool(noteSearchTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		project := req.GetString("project", "")
		limit := int(req.GetFloat("limit", 10))

		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}
		if limit <= 0 {
			limit = 10
		}

		results, err := noteSvc.Search(ctx, hostID, project, query, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("No notes found"), nil
		}

		var sb strings.Builder
		for _, r := range results {
			tags := r.Tags
			if tags == "[]" || tags == "" {
				tags = "(no tags)"
			}
			projectStr := r.Project
			if projectStr == "" {
				projectStr = "(no project)"
			}
			sb.WriteString(fmt.Sprintf("[%s] %s (score=%.3f, id=%d)\n  Tags: %s\n  %s\n",
				projectStr, r.Title, r.Similarity, r.ID, tags, truncate(r.Content, 200)))
		}
		return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
	})

	// note_list
	noteListTool := mcp.NewTool("note_list",
		mcp.WithDescription("List notes, optionally filtered by project"),
		mcp.WithString("project", mcp.Description("Filter by project (optional)")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(20)),
	)
	s.AddTool(noteListTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		project := req.GetString("project", "")
		limit := int(req.GetFloat("limit", 20))
		if limit <= 0 {
			limit = 20
		}

		notes, err := noteSvc.List(ctx, hostID, project, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}

		if len(notes) == 0 {
			return mcp.NewToolResultText("No notes found"), nil
		}

		var sb strings.Builder
		for _, n := range notes {
			projectStr := n.Project
			if projectStr == "" {
				projectStr = "(no project)"
			}
			sb.WriteString(fmt.Sprintf("[%s] %s (id=%d)\n", projectStr, n.Title, n.ID))
		}
		return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
	})

	// note_delete
	noteDeleteTool := mcp.NewTool("note_delete",
		mcp.WithDescription("Soft-delete a note by its ID"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Note ID to delete")),
	)
	s.AddTool(noteDeleteTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := int64(req.GetFloat("id", 0))
		if id <= 0 {
			return mcp.NewToolResultError("valid id is required"), nil
		}

		if err := noteSvc.DeleteByID(ctx, id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Deleted note id=%d", id)), nil
	})
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
