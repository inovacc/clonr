package grpcserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ServerInfo contains information about a running server
type ServerInfo struct {
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

// WriteServerInfo writes server information to the local data directory
func WriteServerInfo(port int) error {
	// Use OS-appropriate local data directory
	// Windows: C:\Users\<user>\AppData\Local\clonr
	// Linux: ~/.local/share/clonr
	// macOS: ~/Library/Application Support/clonr
	dataDir, err := os.UserCacheDir() // This gives us AppData\Local on Windows
	if err != nil {
		return fmt.Errorf("failed to get local data directory: %w", err)
	}

	clonrDir := filepath.Join(dataDir, "clonr")
	if err := os.MkdirAll(clonrDir, 0755); err != nil {
		return fmt.Errorf("failed to create clonr directory: %w", err)
	}

	info := ServerInfo{
		Address:   fmt.Sprintf("localhost:%d", port),
		Port:      port,
		PID:       os.Getpid(),
		StartedAt: time.Now(),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server info: %w", err)
	}

	serverInfoPath := filepath.Join(clonrDir, "server.json")
	if err := os.WriteFile(serverInfoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write server info file: %w", err)
	}

	return nil
}

// RemoveServerInfo removes the server info file (called when server stops)
func RemoveServerInfo() {
	dataDir, err := os.UserCacheDir()
	if err != nil {
		return // Ignore errors on cleanup
	}

	serverInfoPath := filepath.Join(dataDir, "clonr", "server.json")
	_ = os.Remove(serverInfoPath) // Ignore errors
}
