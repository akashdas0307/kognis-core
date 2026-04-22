package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// NATSConfig holds NATS server configuration.
type NATSConfig struct {
	Port       int
	DataDir    string
	ServerName string
}

// SupervisorConfig holds plugin supervisor configuration.
type SupervisorConfig struct {
	HeartbeatIntervalSec int
	RegistrationTimeoutSec int
	ShutdownGracePeriodSec int
	MaxRestartAttempts int
}

// Config holds all core daemon configuration.
type Config struct {
	Path       string
	PluginsDir string
	DataDir    string
	NATS       NATSConfig
	Supervisor SupervisorConfig
}

// Load reads configuration from default locations and environment.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determine home directory: %w", err)
	}

	dataDir := filepath.Join(home, ".kognis")
	cfg := &Config{
		Path:       filepath.Join(dataDir, "config.yaml"),
		PluginsDir:  filepath.Join(dataDir, "plugins"),
		DataDir:     dataDir,
		NATS: NATSConfig{
			Port:       4222,
			DataDir:    filepath.Join(dataDir, "nats"),
			ServerName: "kognis-core",
		},
		Supervisor: SupervisorConfig{
			HeartbeatIntervalSec:   10,
			RegistrationTimeoutSec: 5,
			ShutdownGracePeriodSec: 30,
			MaxRestartAttempts:     5,
		},
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data directory %s: %w", dataDir, err)
	}

	return cfg, nil
}