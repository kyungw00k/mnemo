package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/kyungw00k/mnemo/internal/config"
	"github.com/kyungw00k/mnemo/internal/dashboard"
	"github.com/kyungw00k/mnemo/internal/db"
	mcpserver "github.com/kyungw00k/mnemo/internal/mcp"
	"github.com/kyungw00k/mnemo/internal/migrations"
	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
	"github.com/kyungw00k/mnemo/internal/transport"
)

// RunDashboard handles: mnemo dashboard [--port N]
// Starts an SSE server and opens the dashboard in the browser.
func RunDashboard(args []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			if _, err := strconv.Atoi(args[i+1]); err == nil {
				os.Setenv("SSE_PORT", args[i+1])
			}
			i++
		}
	}

	os.Setenv("TRANSPORT", "sse")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo dashboard: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	var conn *db.DBConn
	if cfg.IsSQLite() {
		conn, err = db.NewSQLite(cfg.SQLitePath())
	} else {
		conn, err = db.NewPostgres(ctx, cfg.DBUrl)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo dashboard: db: %v\n", err)
		os.Exit(1)
	}

	if cfg.IsSQLite() {
		if sqFS, fsErr := migrations.SQLiteFS(); fsErr == nil {
			db.Migrate(ctx, conn, sqFS, cfg.EmbeddingDimensions) //nolint:errcheck
		}
	} else {
		if pgFS, fsErr := migrations.PostgresFS(); fsErr == nil {
			db.Migrate(ctx, conn, pgFS, cfg.EmbeddingDimensions) //nolint:errcheck
		}
	}

	embSvc := service.NewEmbeddingService(cfg)
	memRepo := repository.NewMemoryRepository(conn)
	noteRepo := repository.NewNoteRepository(conn)
	memSvc := service.NewMemoryService(memRepo, embSvc, cfg.MemoryTTLDays)
	noteSvc := service.NewNoteService(noteRepo, embSvc, cfg.MemoryTTLDays)

	mcpSrv := mcpserver.NewServer(cfg, memSvc, noteSvc, conn, cfg.HostID, nil)
	dash := dashboard.NewServer(memSvc, noteSvc, conn)

	url := fmt.Sprintf("http://localhost:%s", cfg.SSEPort)
	log.Printf("mnemo dashboard: %s (Ctrl+C to stop)", url)
	go openBrowser(url)

	errCh := make(chan error, 1)
	go func() {
		errCh <- transport.StartSSE(ctx, mcpSrv, cfg.SSEPort, dash)
	}()

	select {
	case <-ctx.Done():
		log.Println("mnemo dashboard: stopped")
	case err := <-errCh:
		log.Printf("mnemo dashboard: %v", err)
	}
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	exec.Command(cmd, url).Start() //nolint:errcheck
}
