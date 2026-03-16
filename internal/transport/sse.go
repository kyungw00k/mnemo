package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"

	mcpserver "github.com/kyungw00k/mnemo/internal/mcp"
)

// StartSSE starts an HTTP server that serves MCP over SSE on the given port.
// It also exposes a /health endpoint. It shuts down gracefully when ctx is cancelled.
func StartSSE(ctx context.Context, s *mcpserver.Server, port string) error {
	sseServer := server.NewSSEServer(
		s.MCPServer(),
		server.WithBaseURL("http://localhost:"+port),
	)

	mux := http.NewServeMux()

	// Health check endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"transport": "sse",
		})
	})

	// Delegate all other requests to the SSE server.
	mux.Handle("/", sseServer)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start server in background.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("SSE server listening on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("SSE server: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown.
		shutdownCtx := context.Background()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("SSE server shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
