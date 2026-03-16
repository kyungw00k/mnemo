package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/db"
)

func registerDBTools(s *server.MCPServer, conn *db.DBConn) {
	// db_query
	dbQueryTool := mcp.NewTool("db_query",
		mcp.WithDescription("Execute a SELECT query against the database (max 100 rows)"),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SELECT statement to execute")),
	)
	s.AddTool(dbQueryTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sqlStr := strings.TrimSpace(req.GetString("sql", ""))
		if sqlStr == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		// Only allow SELECT statements.
		upper := strings.ToUpper(sqlStr)
		if !strings.HasPrefix(upper, "SELECT") {
			return mcp.NewToolResultError("only SELECT statements are allowed"), nil
		}

		// Enforce LIMIT 100 if no LIMIT clause present.
		if !strings.Contains(upper, "LIMIT") {
			sqlStr += " LIMIT 100"
		}

		rows, err := conn.Query(ctx, sqlStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get columns failed: %v", err)), nil
		}

		var resultRows []map[string]any
		for rows.Next() {
			if len(resultRows) >= 100 {
				break
			}
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("scan row failed: %v", err)), nil
			}
			row := make(map[string]any, len(columns))
			for i, col := range columns {
				row[col] = values[i]
			}
			resultRows = append(resultRows, row)
		}
		if err := rows.Err(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("rows iteration error: %v", err)), nil
		}

		data, err := json.MarshalIndent(resultRows, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal results: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	// db_execute
	dbExecuteTool := mcp.NewTool("db_execute",
		mcp.WithDescription("Execute an INSERT, UPDATE, or DELETE statement against the database"),
		mcp.WithString("sql", mcp.Required(), mcp.Description("INSERT, UPDATE, or DELETE statement")),
	)
	s.AddTool(dbExecuteTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sqlStr := strings.TrimSpace(req.GetString("sql", ""))
		if sqlStr == "" {
			return mcp.NewToolResultError("sql is required"), nil
		}

		// Determine the first word (the statement type).
		firstWord := strings.ToUpper(strings.Fields(sqlStr)[0])

		// Allowlist: only INSERT, UPDATE, DELETE are permitted.
		switch firstWord {
		case "INSERT", "UPDATE", "DELETE":
			// allowed
		default:
			return mcp.NewToolResultError(
				fmt.Sprintf("Only INSERT, UPDATE, DELETE are allowed. Got: %q", firstWord),
			), nil
		}

		result, err := conn.Exec(ctx, sqlStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("execute failed: %v", err)), nil
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return mcp.NewToolResultText("executed successfully (rows affected: unknown)"), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("executed successfully, rows affected: %d", affected)), nil
	})
}
