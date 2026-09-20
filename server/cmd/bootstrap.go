package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

// mustLoadConfig loads the server config from the environment, exiting on failure.
func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	return cfg
}

// bootstrapStore opens the SQLite database, runs migrations, and derives the at-rest encryption key.
func bootstrapStore(cfg *config.Config) (*storage.Store, []byte) {
	db, err := storage.OpenDB(cfg.DBPath)
	if err != nil {
		fail(fmt.Errorf("open db: %w", err))
	}
	backupDir := filepath.Join(filepath.Dir(cfg.DBPath), "backups")
	if err := storage.MigrateWithBackup(db, backupDir, version.Version); err != nil {
		fail(fmt.Errorf("migrate db: %w", err))
	}
	encKey := crypto.DeriveKey(cfg.AuthSecret)
	return storage.New(db, encKey), encKey
}

// bootstrapBus constructs the in-process event bus wired to the store's dedupe and dead-letter tables.
func bootstrapBus(logger *slog.Logger, store *storage.Store) *inprocess.Bus {
	return inprocess.New(inprocess.Options{
		Logger:          logger,
		DedupeStore:     store.ProcessedEvents,
		DeadLetterStore: store.DeadLetters,
	})
}

// resolvePendingUpgrade applies the boot-time instance-upgrade resolution (instance-upgrade spec): called from
// startBackgroundWorkers once runnerSvc exists, after bootstrapStore already ran the migrations this depends on.
// There is no runner callback confirming a stack survived its own restart, so the booted version is the only
// signal — a matching pending/started record completes, anything else unresolved past 15 minutes fails.
func resolvePendingUpgrade(ctx context.Context, runnerSvc *runner.Service, logger *slog.Logger) {
	if err := runnerSvc.ResolvePendingUpgrade(ctx); err != nil {
		logger.Warn("resolve pending instance upgrade failed", "error", err)
	}
}
