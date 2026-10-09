package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/analyzer"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/requester"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]

	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}
	config.InitLogging(cfg.Logging)

	if cfg.Mode != "requester" && cfg.Mode != "auto" {
		log.Warn().Msg("Config mode is not 'requester', but running CLI anyway")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize Requester Client
	client := requester.NewClient(cfg.Requester, cfg.P2P)

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Interrupt received, shutting down...")
		cancel()
	}()

	switch command {
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: unicompute run <project_dir>")
			os.Exit(1)
		}
		projectDir := os.Args[2]
		
		if err := runJob(ctx, client, cfg, projectDir); err != nil {
			log.Fatal().Err(err).Msg("Job execution failed")
		}
	
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("UniCompute CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  unicompute run <path>    Package and distribute a task to the network")
}

func runJob(ctx context.Context, client *requester.Client, cfg *config.AppConfig, projectDir string) error {
	// 1. Connect to coordinator
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to coordinator: %w", err)
	}

	// 2. Package the project
	packager := analyzer.NewPackager(cfg.Requester.MaxCapsuleSizeMB)
	
	outDir := filepath.Join(os.TempDir(), "unicompute_out_"+uuid.New().String()[:8])
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(outDir)
	
	manifest, archivePath, err := packager.PackageProject(projectDir, outDir)
	if err != nil {
		return fmt.Errorf("failed to package project: %w", err)
	}

	jobID := "job-" + uuid.New().String()[:8]
	log.Info().Str("job_id", jobID).Str("lang", manifest.DetectedLanguage).Msg("Project packaged successfully")

	// 3. Submit
	_, err = client.SubmitTask(ctx, jobID, manifest, archivePath)
	if err != nil {
		return err
	}

	// 4. Wait for completion
	job := client.GetJob(jobID)
	if job == nil {
		return fmt.Errorf("job tracking lost locally")
	}

	log.Info().Msg("Waiting for execution to complete...")
	if err := job.Wait(ctx); err != nil {
		return err
	}
	
	log.Info().Msg("Job completed successfully!")
	return nil
}
