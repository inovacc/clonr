package grpcserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ErrNoServerInfo indicates no server info file exists
var ErrNoServerInfo = errors.New("no server info file")

// ServerInfo contains information about a running server
type ServerInfo struct {
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

// getServerInfoPath returns the path to the server.json file
func getServerInfoPath() (string, error) {
	dataDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get local data directory: %w", err)
	}

	return filepath.Join(dataDir, "clonr", "server.json"), nil
}

// ReadServerInfo reads the server info file if it exists
func ReadServerInfo() (*ServerInfo, error) {
	path, err := getServerInfoPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoServerInfo
		}

		return nil, fmt.Errorf("failed to read server info: %w", err)
	}

	var info ServerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse server info: %w", err)
	}

	return &info, nil
}

// IsProcessRunning checks if a process with the given PID is still running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))

	return err == nil
}

// IsServerRunning checks if a clonr server is already running
// Returns the server info if running, nil otherwise
func IsServerRunning() *ServerInfo {
	info, err := ReadServerInfo()
	if err != nil {
		return nil
	}

	// Check if the process is still alive
	if IsProcessRunning(info.PID) {
		return info
	}

	// Process is dead, clean up stale server info
	RemoveServerInfo()

	return nil
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
