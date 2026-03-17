package cli

import (
	"context"

	"github.com/kyungw00k/mnemo/internal/config"
	"github.com/kyungw00k/mnemo/internal/db"
	"github.com/kyungw00k/mnemo/internal/migrations"
	"github.com/kyungw00k/mnemo/internal/repository"
	"github.com/kyungw00k/mnemo/internal/service"
)

type svcs struct {
	cfg        *config.Config
	memSvc     *service.MemoryService
	noteSvc    *service.NoteService
	extractSvc *service.ExtractService // non-nil only when ENABLE_AUTO_EXTRACT=true
}

// initSvcs loads config, connects to DB, runs migrations, and returns services.
// Each CLI subcommand call is short-lived; SQLite WAL mode handles concurrent access safely.
func initSvcs(ctx context.Context) (*svcs, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	var conn *db.DBConn
	if cfg.IsSQLite() {
		conn, err = db.NewSQLite(cfg.SQLitePath())
	} else {
		conn, err = db.NewPostgres(ctx, cfg.DBUrl)
	}
	if err != nil {
		return nil, err
	}

	if cfg.IsSQLite() {
		sqFS, fsErr := migrations.SQLiteFS()
		if fsErr != nil {
			return nil, fsErr
		}
		if migErr := db.Migrate(ctx, conn, sqFS, cfg.EmbeddingDimensions); migErr != nil {
			return nil, migErr
		}
	} else {
		pgFS, fsErr := migrations.PostgresFS()
		if fsErr != nil {
			return nil, fsErr
		}
		if migErr := db.Migrate(ctx, conn, pgFS, cfg.EmbeddingDimensions); migErr != nil {
			return nil, migErr
		}
	}

	embSvc := service.NewEmbeddingService(cfg)
	memRepo := repository.NewMemoryRepository(conn)
	noteRepo := repository.NewNoteRepository(conn)
	memSvc := service.NewMemoryService(memRepo, embSvc, cfg.MemoryTTLDays)
	noteSvc := service.NewNoteService(noteRepo, embSvc, cfg.MemoryTTLDays)

	s := &svcs{cfg: cfg, memSvc: memSvc, noteSvc: noteSvc}
	if cfg.AutoExtractEnabled() {
		s.extractSvc = service.NewExtractService(cfg)
	}
	return s, nil
}
