// Package config provides configuration loading for UniCompute.
//
// Configuration is loaded from (in priority order):
//  1. Command-line flags
//  2. Environment variables (UNICOMPUTE_ prefix)
//  3. Configuration file (~/.unicompute/config.yaml)
//  4. Default values
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	// AppName is the application name used for config file lookup.
	AppName = "unicompute"

	// EnvPrefix is the environment variable prefix.
	EnvPrefix = "UNICOMPUTE"

	// ConfigFileName is the default config file name (without extension).
	ConfigFileName = "config"
)

// Load reads configuration from file, environment, and returns a validated AppConfig.
func Load(cfgFile string) (*AppConfig, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType("yaml")
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Config file search paths.
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(homeDir, "."+AppName))
		}
		v.AddConfigPath(".")
	}

	// Attempt to read config file (optional).
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is acceptable — we use defaults.
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return cfg, nil
}

// InitLogging configures the global logger based on LoggingConfig.
func InitLogging(cfg LoggingConfig) {
	// Set log level.
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output writer.
	var writer *os.File
	if cfg.OutputPath != "" {
		f, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file %s: %v\n", cfg.OutputPath, err)
			writer = os.Stderr
		} else {
			writer = f
		}
	} else {
		writer = os.Stderr
	}

	// Set format.
	if cfg.Format == "console" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: writer}).
			With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(writer).
			With().Timestamp().Logger()
	}

	if cfg.EnableCaller {
		log.Logger = log.Logger.With().Caller().Logger()
	}
}
