package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/service"
)

func registerMemoryTools(s *server.MCPServer, memSvc *service.MemoryService, hostID string) {
	// memory_save
	memorySaveTool := mcp.NewTool("memory_save",
		mcp.WithDescription("Save or update a key-value memory entry in the given category"),
		mcp.WithString("category", mcp.Required(), mcp.Description("Category to organize memories")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Unique key within the category")),
		mcp.WithString("value", mcp.Required(), mcp.Description("The memory value to store")),
		mcp.WithString("metadata", mcp.Description("Optional JSON metadata")),
	)
	s.AddTool(memorySaveTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		category := req.GetString("category", "")
		key := req.GetString("key", "")
		value := req.GetString("value", "")
		metadata := req.GetString("metadata", "")

		if category == "" || key == "" || value == "" {
			return mcp.NewToolResultError("category, key, and value are required"), nil
		}

		mem, err := memSvc.Save(ctx, hostID, category, key, value, metadata)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to save memory: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Saved: [%s] %s (id=%d)", category, key, mem.ID)), nil
	})

	// memory_search
	memorySearchTool := mcp.NewTool("memory_search",
		mcp.WithDescription("Search memories using semantic similarity or full-text search"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("category", mcp.Description("Filter by category (optional)")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(10)),
	)
	s.AddTool(memorySearchTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		category := req.GetString("category", "")
		limit := int(req.GetFloat("limit", 10))

		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}
		if limit <= 0 {
			limit = 10
		}

		results, err := memSvc.Search(ctx, hostID, category, query, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		if len(results) == 0 {
			return mcp.NewToolResultText("No memories found"), nil
		}

		var sb strings.Builder
		for _, r := range results {
			sb.WriteString(fmt.Sprintf("[%s] %s = %s (score=%.3f, id=%d)\n",
				r.Category, r.Key, r.Value, r.Similarity, r.ID))
		}
		return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
	})

	// memory_list
	memoryListTool := mcp.NewTool("memory_list",
		mcp.WithDescription("List memories, optionally filtered by category"),
		mcp.WithString("category", mcp.Description("Filter by category (optional)")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(50)),
	)
	s.AddTool(memoryListTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		category := req.GetString("category", "")
		limit := int(req.GetFloat("limit", 50))
		if limit <= 0 {
			limit = 50
		}

		memories, err := memSvc.List(ctx, hostID, category, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}

		if len(memories) == 0 {
			return mcp.NewToolResultText("No memories found"), nil
		}

		var sb strings.Builder
		for _, m := range memories {
			sb.WriteString(fmt.Sprintf("[%s] %s = %s (id=%d)\n", m.Category, m.Key, m.Value, m.ID))
		}
		return mcp.NewToolResultText(strings.TrimRight(sb.String(), "\n")), nil
	})

	// memory_delete
	memoryDeleteTool := mcp.NewTool("memory_delete",
		mcp.WithDescription("Soft-delete a memory by id, or by category+key"),
		mcp.WithNumber("id", mcp.Description("Memory ID to delete")),
		mcp.WithString("category", mcp.Description("Category (used with key)")),
		mcp.WithString("key", mcp.Description("Key (used with category)")),
	)
	s.AddTool(memoryDeleteTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := int64(req.GetFloat("id", 0))
		category := req.GetString("category", "")
		key := req.GetString("key", "")

		if id > 0 {
			if err := memSvc.DeleteByID(ctx, id); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Deleted memory id=%d", id)), nil
		}

		if category != "" && key != "" {
			if err := memSvc.DeleteByKey(ctx, hostID, category, key); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Deleted memory [%s] %s", category, key)), nil
		}

		return mcp.NewToolResultError("provide either id, or both category and key"), nil
	})

	// memory_categories
	memoryCatsTool := mcp.NewTool("memory_categories",
		mcp.WithDescription("List all distinct memory categories for this host"),
	)
	s.AddTool(memoryCatsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cats, err := memSvc.Categories(ctx, hostID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list categories failed: %v", err)), nil
		}
		if len(cats) == 0 {
			return mcp.NewToolResultText("No categories found"), nil
		}
		return mcp.NewToolResultText(strings.Join(cats, "\n")), nil
	})
}
