package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check default clone directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expectedDir := filepath.Join(homeDir, "clonr")
	if cfg.DefaultCloneDir != expectedDir {
		t.Errorf("DefaultCloneDir = %q, want %q", cfg.DefaultCloneDir, expectedDir)
	}

	// Check default editor
	if cfg.Editor != "code" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "code")
	}

	// Check default terminal (empty)
	if cfg.Terminal != "" {
		t.Errorf("Terminal = %q, want empty string", cfg.Terminal)
	}

	// Check default monitor interval
	if cfg.MonitorInterval != 300 {
		t.Errorf("MonitorInterval = %d, want %d", cfg.MonitorInterval, 300)
	}

	// Check default server port
	if cfg.ServerPort != 4000 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 4000)
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{
		DefaultCloneDir: "/custom/path",
		Editor:          "vim",
		Terminal:        "alacritty",
		MonitorInterval: 600,
		ServerPort:      8080,
	}

	if cfg.DefaultCloneDir != "/custom/path" {
		t.Errorf("DefaultCloneDir = %q, want %q", cfg.DefaultCloneDir, "/custom/path")
	}

	if cfg.Editor != "vim" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "vim")
	}

	if cfg.Terminal != "alacritty" {
		t.Errorf("Terminal = %q, want %q", cfg.Terminal, "alacritty")
	}

	if cfg.MonitorInterval != 600 {
		t.Errorf("MonitorInterval = %d, want %d", cfg.MonitorInterval, 600)
	}

	if cfg.ServerPort != 8080 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 8080)
	}
}

func TestConfig_ZeroValues(t *testing.T) {
	cfg := &Config{}

	if cfg.DefaultCloneDir != "" {
		t.Errorf("zero Config.DefaultCloneDir = %q, want empty", cfg.DefaultCloneDir)
	}

	if cfg.Editor != "" {
		t.Errorf("zero Config.Editor = %q, want empty", cfg.Editor)
	}

	if cfg.Terminal != "" {
		t.Errorf("zero Config.Terminal = %q, want empty", cfg.Terminal)
	}

	if cfg.MonitorInterval != 0 {
		t.Errorf("zero Config.MonitorInterval = %d, want 0", cfg.MonitorInterval)
	}

	if cfg.ServerPort != 0 {
		t.Errorf("zero Config.ServerPort = %d, want 0", cfg.ServerPort)
	}
}
