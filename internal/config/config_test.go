package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ServerAddr != "127.0.0.1:7373" {
		t.Errorf("Expected ServerAddr '127.0.0.1:7373', got '%s'", cfg.ServerAddr)
	}
	if cfg.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers 4, got %d", cfg.MaxWorkers)
	}
	if cfg.DefaultAgent != "codex" {
		t.Errorf("Expected DefaultAgent 'codex', got '%s'", cfg.DefaultAgent)
	}
}

func TestResolveAgent(t *testing.T) {
	cfg := DefaultConfig()
	tmpDir := t.TempDir()

	// Test explicit agent takes precedence
	agent := ResolveAgent("claude-code", tmpDir, cfg)
	if agent != "claude-code" {
		t.Errorf("Expected 'claude-code', got '%s'", agent)
	}

	// Test empty explicit falls back to global config
	agent = ResolveAgent("", tmpDir, cfg)
	if agent != "codex" {
		t.Errorf("Expected 'codex' (from global), got '%s'", agent)
	}

	// Test per-repo config
	repoConfig := filepath.Join(tmpDir, ".roborev.toml")
	os.WriteFile(repoConfig, []byte(`agent = "claude-code"`), 0644)

	agent = ResolveAgent("", tmpDir, cfg)
	if agent != "claude-code" {
		t.Errorf("Expected 'claude-code' (from repo config), got '%s'", agent)
	}

	// Explicit still takes precedence over repo config
	agent = ResolveAgent("codex", tmpDir, cfg)
	if agent != "codex" {
		t.Errorf("Expected 'codex' (explicit), got '%s'", agent)
	}
}

func TestSaveAndLoadGlobal(t *testing.T) {
	// Use temp home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	cfg.DefaultAgent = "claude-code"
	cfg.MaxWorkers = 8

	err := SaveGlobal(cfg)
	if err != nil {
		t.Fatalf("SaveGlobal failed: %v", err)
	}

	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal failed: %v", err)
	}

	if loaded.DefaultAgent != "claude-code" {
		t.Errorf("Expected DefaultAgent 'claude-code', got '%s'", loaded.DefaultAgent)
	}
	if loaded.MaxWorkers != 8 {
		t.Errorf("Expected MaxWorkers 8, got %d", loaded.MaxWorkers)
	}
}
