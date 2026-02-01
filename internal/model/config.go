package model

import (
	"os"
	"path/filepath"

	"github.com/inovacc/clonr/internal/application"
)

// Editor represents a custom editor configuration.
type Editor struct {
	// Name is the display name of the editor (e.g., "VS Code")
	Name string `json:"name"`

	// Command is the executable command (e.g., "code")
	Command string `json:"command"`

	// Icon is an optional icon for display (e.g., "󰨞")
	Icon string `json:"icon,omitempty"`
}

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

	// CustomEditors is a list of user-defined editors
	CustomEditors []Editor `json:"custom_editors,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	// Get a user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return Config{
		DefaultCloneDir: filepath.Join(homeDir, application.AppName),
		Editor:          "code", // VS Code as default
		Terminal:        "",
		MonitorInterval: 300, // 5 minutes
		ServerPort:      4000,
	}
}
