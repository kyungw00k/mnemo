package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/config"
	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
)

func registerExtraTools(
	s *server.MCPServer,
	cfg *config.Config,
	memSvc *service.MemoryService,
	noteSvc *service.NoteService,
	extractSvc *service.ExtractService,
	hostID string,
) {
	// memory_extract — opt-in: ENABLE_AUTO_EXTRACT=true
	if cfg.AutoExtractEnabled() && extractSvc != nil {
		memoryExtractTool := mcp.NewTool("memory_extract",
			mcp.WithDescription("Extract and save important memories from conversation text using AI"),
			mcp.WithString("text", mcp.Required(), mcp.Description("Conversation or text to extract memories from")),
			mcp.WithString("category_prefix", mcp.Description("Optional prefix to prepend to extracted categories")),
		)
		s.AddTool(memoryExtractTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			text := req.GetString("text", "")
			prefix := req.GetString("category_prefix", "")

			if text == "" {
				return mcp.NewToolResultError("text is required"), nil
			}

			extracted, err := extractSvc.Extract(ctx, text)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("extract failed: %v", err)), nil
			}

			if len(extracted) == 0 {
				return mcp.NewToolResultText("No memories extracted"), nil
			}

			var saved int
			for _, em := range extracted {
				category := em.Category
				if prefix != "" {
					category = prefix + "/" + category
				}
				if _, err := memSvc.Save(ctx, hostID, category, em.Key, em.Value, ""); err == nil {
					saved++
				}
			}

			return mcp.NewToolResultText(fmt.Sprintf("Extracted %d facts, saved %d memories", len(extracted), saved)), nil
		})
	}

	// memory_cleanup — opt-in: MEMORY_TTL_DAYS>0
	if cfg.TTLEnabled() {
		memoryCleanupTool := mcp.NewTool("memory_cleanup",
			mcp.WithDescription("Hard-delete all expired memories and notes"),
		)
		s.AddTool(memoryCleanupTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			deletedMem, err := memSvc.Cleanup(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("cleanup memories failed: %v", err)), nil
			}
			deletedNote, err := noteSvc.Cleanup(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("cleanup notes failed: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Deleted %d memories and %d notes", deletedMem, deletedNote)), nil
		})
	}

	// memory_export — always on
	memoryExportTool := mcp.NewTool("memory_export",
		mcp.WithDescription("Export all memories and notes for this host as JSON or Markdown"),
		mcp.WithString("format", mcp.Description("Output format: json (default) or markdown")),
		mcp.WithString("category", mcp.Description("Filter memories by category (optional)")),
	)
	s.AddTool(memoryExportTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		format := req.GetString("format", "json")
		categoryFilter := req.GetString("category", "")

		memories, err := memSvc.ExportAll(ctx, hostID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("export memories failed: %v", err)), nil
		}
		notes, err := noteSvc.ExportAll(ctx, hostID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("export notes failed: %v", err)), nil
		}

		// Apply category filter.
		if categoryFilter != "" {
			filtered := make([]*repository.Memory, 0, len(memories))
			for _, m := range memories {
				if m.Category == categoryFilter {
					filtered = append(filtered, m)
				}
			}
			memories = filtered
		}

		if format == "markdown" {
			return mcp.NewToolResultText(formatExportMarkdown(memories, notes)), nil
		}

		// JSON format.
		type exportPayload struct {
			Memories []*repository.Memory `json:"memories"`
			Notes    []*repository.Note   `json:"notes"`
		}
		payload := exportPayload{Memories: memories, Notes: notes}
		if payload.Memories == nil {
			payload.Memories = []*repository.Memory{}
		}
		if payload.Notes == nil {
			payload.Notes = []*repository.Note{}
		}
		out, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal export: %v", err)), nil
		}
		return mcp.NewToolResultText(string(out)), nil
	})

	// memory_import — always on
	memoryImportTool := mcp.NewTool("memory_import",
		mcp.WithDescription("Bulk import memories from JSON (same format as memory_export)"),
		mcp.WithString("data", mcp.Required(), mcp.Description("JSON string with memories array")),
	)
	s.AddTool(memoryImportTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data := req.GetString("data", "")
		if data == "" {
			return mcp.NewToolResultError("data is required"), nil
		}

		var memories []*repository.Memory

		// Try bare array first.
		if err := json.Unmarshal([]byte(data), &memories); err != nil {
			// Try envelope format: {"memories": [...]}
			var envelope struct {
				Memories []*repository.Memory `json:"memories"`
			}
			if err2 := json.Unmarshal([]byte(data), &envelope); err2 != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid JSON: %v", err)), nil
			}
			memories = envelope.Memories
		}

		if len(memories) == 0 {
			return mcp.NewToolResultText("No memories to import"), nil
		}

		// Assign hostID to all imported memories.
		for _, m := range memories {
			m.Host = hostID
		}

		if err := memSvc.BulkImport(ctx, memories); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("import failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Imported %d memories", len(memories))), nil
	})
}

// formatExportMarkdown formats memories and notes as human-readable Markdown.
func formatExportMarkdown(memories []*repository.Memory, notes []*repository.Note) string {
	var sb strings.Builder

	sb.WriteString("## Memories\n\n")
	if len(memories) == 0 {
		sb.WriteString("_(none)_\n\n")
	} else {
		// Group by category.
		catOrder := []string{}
		catMap := map[string][]*repository.Memory{}
		for _, m := range memories {
			if _, ok := catMap[m.Category]; !ok {
				catOrder = append(catOrder, m.Category)
			}
			catMap[m.Category] = append(catMap[m.Category], m)
		}
		for _, cat := range catOrder {
			sb.WriteString(fmt.Sprintf("### %s\n\n", cat))
			for _, m := range catMap[cat] {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", m.Key, m.Value))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Notes\n\n")
	if len(notes) == 0 {
		sb.WriteString("_(none)_\n\n")
	} else {
		for _, n := range notes {
			sb.WriteString(fmt.Sprintf("### %s\n\n", n.Title))
			sb.WriteString(n.Content)
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
