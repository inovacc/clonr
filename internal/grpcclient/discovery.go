package grpcclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClientConfig holds client configuration for connecting to the server
type ClientConfig struct {
	ServerAddress  string `json:"server_address"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// discoverServerAddress determines the server address to connect to
// Priority:
// 1. CLONR_SERVER environment variable
// 2. ~/clonr/config.json config file
// 3. Default: localhost: 50051
func discoverServerAddress() (string, error) {
	// 1. Check environment variable
	if addr := os.Getenv("CLONR_SERVER"); addr != "" {
		return addr, nil
	}

	// 2. Check the config file
	homeDir, err := os.UserConfigDir()
	if err == nil {
		configPath := filepath.Join(homeDir, "clonr", "config.json")
		if data, err := os.ReadFile(configPath); err == nil {
			var cfg ClientConfig
			if err := json.Unmarshal(data, &cfg); err == nil && cfg.ServerAddress != "" {
				return cfg.ServerAddress, nil
			}
		}
	}

	// 3. Default
	return "localhost:50051", nil
}

// SaveServerAddress saves the server address to the config file
func SaveServerAddress(address string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "clonr")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := ClientConfig{
		ServerAddress:  address,
		TimeoutSeconds: 30,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(configDir, "client.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
