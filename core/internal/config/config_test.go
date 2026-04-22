package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.PluginsDir == "" {
		t.Fatal("expected PluginsDir to be set")
	}
	if cfg.DataDir == "" {
		t.Fatal("expected DataDir to be set")
	}
	if cfg.NATS.Port != 4222 {
		t.Fatalf("expected NATS port 4222, got %d", cfg.NATS.Port)
	}
	if cfg.NATS.ServerName != "kognis-core" {
		t.Fatalf("expected NATS server name kognis-core, got %s", cfg.NATS.ServerName)
	}
	if cfg.Supervisor.HeartbeatIntervalSec != 10 {
		t.Fatalf("expected heartbeat 10, got %d", cfg.Supervisor.HeartbeatIntervalSec)
	}
	if cfg.Supervisor.RegistrationTimeoutSec != 5 {
		t.Fatalf("expected registration timeout 5, got %d", cfg.Supervisor.RegistrationTimeoutSec)
	}
	if cfg.Supervisor.ShutdownGracePeriodSec != 30 {
		t.Fatalf("expected shutdown grace 30, got %d", cfg.Supervisor.ShutdownGracePeriodSec)
	}
	if cfg.Supervisor.MaxRestartAttempts != 5 {
		t.Fatalf("expected max restarts 5, got %d", cfg.Supervisor.MaxRestartAttempts)
	}
}

func TestDataDirectoryCreated(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	stat, err := os.Stat(cfg.DataDir)
	if err != nil {
		t.Fatalf("data directory %s not created: %v", cfg.DataDir, err)
	}
	if !stat.IsDir() {
		t.Fatalf("%s is not a directory", cfg.DataDir)
	}
}

func TestPathsAreConsistent(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedDataDir := filepath.Join(home, ".kognis")
	if cfg.DataDir != expectedDataDir {
		t.Fatalf("expected DataDir %s, got %s", expectedDataDir, cfg.DataDir)
	}
	if cfg.PluginsDir != filepath.Join(expectedDataDir, "plugins") {
		t.Fatalf("expected PluginsDir %s, got %s", filepath.Join(expectedDataDir, "plugins"), cfg.PluginsDir)
	}
	if cfg.NATS.DataDir != filepath.Join(expectedDataDir, "nats") {
		t.Fatalf("expected NATS DataDir %s, got %s", filepath.Join(expectedDataDir, "nats"), cfg.NATS.DataDir)
	}
}