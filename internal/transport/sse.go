package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"

	"github.com/kyungw00k/mnemo/internal/dashboard"
	mcpserver "github.com/kyungw00k/mnemo/internal/mcp"
)

// StartSSE starts an HTTP server that serves MCP over SSE on the given port.
// It also exposes a /health endpoint and a dashboard UI. It shuts down gracefully when ctx is cancelled.
func StartSSE(ctx context.Context, s *mcpserver.Server, port string, dash *dashboard.Server) error {
	sseServer := server.NewSSEServer(
		s.MCPServer(),
		server.WithBaseURL("http://localhost:"+port),
	)

	mux := http.NewServeMux()

	// Health check endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"status":    "ok",
			"transport": "sse",
		})
	})

	// Dashboard REST API endpoints (registered before catch-all so they take priority).
	// Note: /api/notes/ must be registered before /api/notes for proper routing.
	// Note: /api/memories/ must be registered before /api/memories for proper routing.
	mux.HandleFunc("/api/stats", dash.HandleStats)
	mux.HandleFunc("/api/memories/", dash.HandleMemoryDetail)
	mux.HandleFunc("/api/memories", dash.HandleMemories)
	mux.HandleFunc("/api/notes/", dash.HandleNoteDetail)
	mux.HandleFunc("/api/notes", dash.HandleNotes)
	mux.HandleFunc("/api/search", dash.HandleSearch)
	mux.HandleFunc("/api/graph", dash.HandleGraph)

	// Dashboard static assets.
	mux.Handle("/assets/", dash.FileServer())

	// Delegate all other requests to the SSE server, but serve dashboard index at root.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodGet {
			dash.HandleIndex(w, r)
			return
		}
		sseServer.ServeHTTP(w, r)
	})

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
