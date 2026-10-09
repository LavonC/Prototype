package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/worker"
)

func main() {
	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}
	config.InitLogging(cfg.Logging)

	if cfg.Mode != "worker" {
		log.Fatal().Msg("Config mode must be 'worker' to run this binary")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize Worker Controller
	controller := worker.NewController(cfg.Worker, cfg.P2P)

	// 3. Start Controller in background
	go func() {
		if err := controller.Start(ctx); err != nil {
			log.Fatal().Err(err).Msg("Worker daemon crashed")
		}
	}()

	// 4. Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Info().Msg("Received termination signal, shutting down...")
	cancel() // Triggers graceful shutdown
}
