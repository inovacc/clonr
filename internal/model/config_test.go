package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestDefaultConfig_Consistency(t *testing.T) {
	// Multiple calls should return same values
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	if cfg1.DefaultCloneDir != cfg2.DefaultCloneDir {
		t.Error("DefaultConfig() returns inconsistent DefaultCloneDir")
	}

	if cfg1.Editor != cfg2.Editor {
		t.Error("DefaultConfig() returns inconsistent Editor")
	}

	if cfg1.Terminal != cfg2.Terminal {
		t.Error("DefaultConfig() returns inconsistent Terminal")
	}

	if cfg1.MonitorInterval != cfg2.MonitorInterval {
		t.Error("DefaultConfig() returns inconsistent MonitorInterval")
	}

	if cfg1.ServerPort != cfg2.ServerPort {
		t.Error("DefaultConfig() returns inconsistent ServerPort")
	}
}

func TestConfig_JSONMarshaling(t *testing.T) {
	original := Config{
		DefaultCloneDir: "/custom/clone/dir",
		Editor:          "vim",
		Terminal:        "kitty",
		MonitorInterval: 600,
		ServerPort:      8080,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Config

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.DefaultCloneDir != original.DefaultCloneDir {
		t.Errorf("DefaultCloneDir = %q, want %q", decoded.DefaultCloneDir, original.DefaultCloneDir)
	}

	if decoded.Editor != original.Editor {
		t.Errorf("Editor = %q, want %q", decoded.Editor, original.Editor)
	}

	if decoded.Terminal != original.Terminal {
		t.Errorf("Terminal = %q, want %q", decoded.Terminal, original.Terminal)
	}

	if decoded.MonitorInterval != original.MonitorInterval {
		t.Errorf("MonitorInterval = %d, want %d", decoded.MonitorInterval, original.MonitorInterval)
	}

	if decoded.ServerPort != original.ServerPort {
		t.Errorf("ServerPort = %d, want %d", decoded.ServerPort, original.ServerPort)
	}
}

func TestConfig_JSONFields(t *testing.T) {
	cfg := Config{
		DefaultCloneDir: "/test/path",
		Editor:          "nano",
		Terminal:        "gnome-terminal",
		MonitorInterval: 120,
		ServerPort:      3000,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	jsonStr := string(data)

	// Verify JSON field names match struct tags
	expectedFields := []string{
		`"default_clone_dir":"/test/path"`,
		`"editor":"nano"`,
		`"terminal":"gnome-terminal"`,
		`"monitor_interval":120`,
		`"server_port":3000`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing field %q in %s", field, jsonStr)
		}
	}
}

func TestDefaultConfig_ContainsClonrDir(t *testing.T) {
	cfg := DefaultConfig()

	// DefaultCloneDir should contain "clonr"
	if !strings.Contains(cfg.DefaultCloneDir, "clonr") {
		t.Errorf("DefaultCloneDir = %q, should contain 'clonr'", cfg.DefaultCloneDir)
	}
}

func TestDefaultConfig_PositiveValues(t *testing.T) {
	cfg := DefaultConfig()

	// MonitorInterval should be positive
	if cfg.MonitorInterval <= 0 {
		t.Errorf("MonitorInterval = %d, should be positive", cfg.MonitorInterval)
	}

	// ServerPort should be in valid range
	if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
		t.Errorf("ServerPort = %d, should be between 1 and 65535", cfg.ServerPort)
	}
}
