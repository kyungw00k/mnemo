package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyungw00k/mnemo/internal/cli"
	"github.com/kyungw00k/mnemo/internal/config"
	"github.com/kyungw00k/mnemo/internal/dashboard"
	"github.com/kyungw00k/mnemo/internal/db"
	mcpserver "github.com/kyungw00k/mnemo/internal/mcp"
	"github.com/kyungw00k/mnemo/internal/migrations"
	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
	"github.com/kyungw00k/mnemo/internal/transport"
)

func main() {
	// All logs MUST go to stderr — stdout is reserved for MCP JSON-RPC.
	log.SetOutput(os.Stderr)

	// Subcommand routing — short-lived, no server started.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hook":
			cli.RunHook(os.Args[2:])
			return
		case "search":
			cli.RunSearch(os.Args[2:])
			return
		case "save":
			cli.RunSave(os.Args[2:])
			return
		case "dashboard":
			cli.RunDashboard(os.Args[2:])
			return
		}
	}

	// 1. Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Set up context with graceful shutdown on SIGTERM/SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// 2. Connect to the database.
	var conn *db.DBConn
	if cfg.IsSQLite() {
		conn, err = db.NewSQLite(cfg.SQLitePath())
		if err != nil {
			log.Fatalf("connect sqlite: %v", err)
		}
	} else {
		conn, err = db.NewPostgres(ctx, cfg.DBUrl)
		if err != nil {
			log.Fatalf("connect postgres: %v", err)
		}
	}

	// 3. Run migrations.
	var migrationsErr error

	if cfg.IsSQLite() {
		sqFS, err := migrations.SQLiteFS()
		if err != nil {
			log.Fatalf("get sqlite migrations: %v", err)
		}
		migrationsErr = db.Migrate(ctx, conn, sqFS, cfg.EmbeddingDimensions)
	} else {
		pgFS, err := migrations.PostgresFS()
		if err != nil {
			log.Fatalf("get postgres migrations: %v", err)
		}
		migrationsErr = db.Migrate(ctx, conn, pgFS, cfg.EmbeddingDimensions)
	}
	if migrationsErr != nil {
		log.Fatalf("run migrations: %v", migrationsErr)
	}

	// 4. Create services.
	embSvc := service.NewEmbeddingService(cfg)
	memRepo := repository.NewMemoryRepository(conn)
	noteRepo := repository.NewNoteRepository(conn)
	memSvc := service.NewMemoryService(memRepo, embSvc, cfg.MemoryTTLDays)
	noteSvc := service.NewNoteService(noteRepo, embSvc, cfg.MemoryTTLDays)

	// Phase 13: create ExtractService only if AutoExtract is enabled.
	var extractSvc *service.ExtractService
	if cfg.AutoExtractEnabled() {
		extractSvc = service.NewExtractService(cfg)
	}

	// 5. Auto-install Claude Code hooks when Claude Code is detected.
	cli.MaybeAutoInstall()

	// 6. Create MCP server and dashboard.
	mcpSrv := mcpserver.NewServer(cfg, memSvc, noteSvc, conn, cfg.HostID, extractSvc)
	dash := dashboard.NewServer(memSvc, noteSvc, conn)

	// Phase 13: start background TTL cleanup goroutine if TTL is enabled.
	if cfg.TTLEnabled() {
		interval, err := time.ParseDuration(cfg.MemoryTTLCleanupInterval)
		if err != nil {
			log.Printf("invalid MEMORY_TTL_CLEANUP_INTERVAL %q (using 1h): %v", cfg.MemoryTTLCleanupInterval, err)
			interval = time.Hour
		}
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if n, err := memSvc.Cleanup(ctx); err != nil {
						log.Printf("ttl cleanup memories: %v", err)
					} else if n > 0 {
						log.Printf("ttl cleanup: deleted %d expired memories", n)
					}
					if n, err := noteSvc.Cleanup(ctx); err != nil {
						log.Printf("ttl cleanup notes: %v", err)
					} else if n > 0 {
						log.Printf("ttl cleanup: deleted %d expired notes", n)
					}
				}
			}
		}()
	}

	// 7. Start transports based on TRANSPORT config.
	errCh := make(chan error, 2)

	switch cfg.Transport {
	case "stdio":
		go func() {
			if err := transport.StartStdio(ctx, mcpSrv); err != nil {
				errCh <- err
			}
		}()

	case "sse":
		go func() {
			if err := transport.StartSSE(ctx, mcpSrv, cfg.SSEPort, dash); err != nil {
				errCh <- err
			}
		}()

	default: // "both"
		go func() {
			if err := transport.StartSSE(ctx, mcpSrv, cfg.SSEPort, dash); err != nil {
				errCh <- err
			}
		}()
		go func() {
			if err := transport.StartStdio(ctx, mcpSrv); err != nil {
				errCh <- err
			}
		}()
	}

	// 8. Wait for shutdown signal or transport error.
	select {
	case <-ctx.Done():
		log.Println("shutting down mnemo...")
	case err := <-errCh:
		log.Printf("transport error: %v", err)
	}
}
