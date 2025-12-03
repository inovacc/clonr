package model

import (
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	// DefaultCloneDir is the default directory where repositories will be cloned
	DefaultCloneDir string `json:"default_clone_dir"`

	// Editor is the default editor to open repositories
	Editor string `json:"editor"`

	// Terminal is the default terminal application
	Terminal string `json:"terminal"`

	// MonitorInterval is the interval in seconds for monitoring repositories
	MonitorInterval int `json:"monitor_interval"`

	// ServerPort is the port for the API server
	ServerPort int `json:"server_port"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	// Get a user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	defaultCloneDir := filepath.Join(homeDir, "clonr")

	return Config{
		DefaultCloneDir: defaultCloneDir,
		Editor:          "code", // VS Code as default
		Terminal:        "",
		MonitorInterval: 300, // 5 minutes
		ServerPort:      4000,
	}
}
