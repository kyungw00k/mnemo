package transport

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/server"

	mcpserver "github.com/kyungw00k/mnemo/internal/mcp"
)

// StartStdio serves MCP over stdin/stdout and blocks until the context is cancelled or an error occurs.
func StartStdio(ctx context.Context, s *mcpserver.Server) error {
	// ServeStdio blocks until stdin is closed.
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ServeStdio(s.MCPServer())
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("stdio transport: %w", err)
		}
		return nil
	}
}
