// tools is a CLI for database maintenance operations.
//
// Usage:
//
//	tools backfill  [--batch-size N]  backfill resources into the event log
//	tools parity                      check event log vs projection parity
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"selfsystems/internal/config"
	"selfsystems/internal/eventstore"
	"selfsystems/internal/migration"
	postgresrepo "selfsystems/internal/repository/postgres"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	db, store, cleanup, err := openDB(cfg)
	if err != nil {
		slog.Error("open db", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	switch os.Args[1] {
	case "backfill":
		runBackfill(db, store, os.Args[2:])
	case "parity":
		runParity(db, store, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runBackfill(db *sql.DB, store eventstore.Store, args []string) {
	fs := flag.NewFlagSet("backfill", flag.ExitOnError)
	batchSize := fs.Int("batch-size", 500, "events per transaction")
	correlationID := fs.String("correlation-id", "", "custom correlation ID (auto-generated if empty)")
	_ = fs.Parse(args)

	slog.Info("starting resource backfill", "batch_size", *batchSize)

	result, err := migration.RunResourceBackfill(context.Background(), db, store, migration.BackfillConfig{
		BatchSize:     *batchSize,
		CorrelationID: *correlationID,
		OnProgress: func(processed, total int) {
			slog.Info("backfill progress", "processed", processed, "total", total)
		},
	})
	if err != nil {
		slog.Error("backfill failed", "error", err)
		os.Exit(1)
	}

	slog.Info("backfill complete",
		"processed", result.Processed,
		"skipped", result.Skipped,
		"correlation_id", result.CorrelationID,
		"duration", result.Duration.String(),
	)
}

func runParity(db *sql.DB, store eventstore.Store, args []string) {
	fs := flag.NewFlagSet("parity", flag.ExitOnError)
	_ = fs.Parse(args)

	slog.Info("running parity check")

	report, err := migration.CheckResourceParity(context.Background(), db, store)
	if err != nil {
		slog.Error("parity check failed", "error", err)
		os.Exit(1)
	}

	fmt.Println(migration.FormatReport(report))

	if !report.IsClean() {
		os.Exit(2)
	}
}

func openDB(cfg config.Config) (*sql.DB, eventstore.Store, func(), error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Database.Type)) {
	case "", "sqlite":
		db, err := sqliterepo.Open(cfg.Database.Path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open sqlite: %w", err)
		}
		return db, eventstore.NewSQLiteStore(db), func() { _ = db.Close() }, nil
	case "postgres", "postgresql":
		db, err := postgresrepo.Open(cfg.Database.URL)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("open postgres: %w", err)
		}
		return db, eventstore.NewPostgresStore(db), func() { _ = db.Close() }, nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported database type %q", cfg.Database.Type)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Self Systems Tools

Usage:
  tools <subcommand> [flags]

Subcommands:
  backfill   Seed pre-existing resources into the event log as ResourceImported events.
             Flags: --batch-size N (default 500), --correlation-id ID
  parity     Compare the live resources projection against the event log.
             Exit 0 = clean, 2 = divergences found.

Config loaded from config/config.default.yml and SS_ environment variables.
`)
}
