package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/config"
	"github.com/kyungw00k/mnemo/internal/db"
	"github.com/kyungw00k/mnemo/internal/service"
)

// Server wraps the mcp-go MCPServer and provides a unified entry point.
type Server struct {
	mcpServer *server.MCPServer
}

// NewServer creates a new MCP server with all tools registered.
// extractSvc may be nil if AutoExtract is not enabled.
func NewServer(
	cfg *config.Config,
	memSvc *service.MemoryService,
	noteSvc *service.NoteService,
	dbConn *db.DBConn,
	hostID string,
	extractSvc *service.ExtractService,
) *Server {
	s := server.NewMCPServer(
		"mnemo",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	registerMemoryTools(s, memSvc, hostID)
	registerNoteTools(s, noteSvc, hostID)
	registerDBTools(s, dbConn)
	registerExtraTools(s, cfg, memSvc, noteSvc, extractSvc, hostID)

	return &Server{mcpServer: s}
}

// MCPServer returns the underlying mcp-go MCPServer instance.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}
