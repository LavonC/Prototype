package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/coordinator"
	"github.com/unicompute/unicompute/internal/storage"
)

func main() {
	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	config.InitLogging(cfg.Logging)

	if cfg.Mode != "coordinator" {
		log.Fatal().Msg("Config mode must be 'coordinator' to run this binary")
	}

	// 2. Init Database (PostgreSQL typically for coordinator, SQLite for demo)
	db, err := storage.NewSQLiteDB(cfg.Storage.DSN, cfg.Storage.MigrationsDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Run Migrations
	if err := db.Migrate(ctx); err != nil {
		log.Fatal().Err(err).Msg("Database migration failed")
	}

	// 4. Initialize Middleware & Server
	mw := coordinator.NewMiddleware(cfg.Security)
	server := coordinator.NewServer(cfg.Coordinator, db, mw)

	// 5. Start Server in background
	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatal().Err(err).Msg("Server crashed")
		}
	}()

	// 6. Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Info().Msg("Received termination signal, shutting down...")
	cancel() // Triggers graceful shutdown in server.Start
}
