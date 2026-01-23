package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// ClientConfig holds client configuration for connecting to the server
type ClientConfig struct {
	ServerAddress  string `json:"server_address"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// ServerInfo contains information about a running server (matches grpcserver.ServerInfo)
type ServerInfo struct {
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

// discoverServerAddress determines the server address to connect to
// Priority:
// 1. CLONR_SERVER environment variable (if set, use it directly)
// 2. ~/.config/clonr/server.json (written by running server)
// 3. Probe common ports for a running server (50051-50055)
// 4. ~/.config/clonr/client.json config file
// 5. Default: localhost:50051
func discoverServerAddress() (string, error) {
	// 1. Check environment variable - if explicitly set, trust it
	if addr := os.Getenv("CLONR_SERVER"); addr != "" {
		return addr, nil
	}

	// 2. Check server info file (written by server when it starts)
	// Location: AppData\Local\clonr on Windows, ~/.local/share/clonr on Linux, ~/Library/Application Support/clonr on macOS
	dataDir, err := os.UserCacheDir()
	if err == nil {
		serverInfoPath := filepath.Join(dataDir, "clonr", "server.json")
		if data, err := os.ReadFile(serverInfoPath); err == nil {
			var info ServerInfo
			if err := json.Unmarshal(data, &info); err == nil {
				// Verify the server is actually running
				if isServerRunning(info.Address) {
					return info.Address, nil
				}
				// Server info exists but server not running - clean up stale file
				_ = os.Remove(serverInfoPath)
			}
		}
	}

	// 3. Probe common ports to find a running server
	commonPorts := []int{50051, 50052, 50053, 50054, 50055}
	for _, port := range commonPorts {
		addr := fmt.Sprintf("localhost:%d", port)
		if isServerRunning(addr) {
			return addr, nil
		}
	}

	// 4. Check client config file (still in .config for backwards compatibility)
	homeDir, homeErr := os.UserHomeDir()
	if homeErr == nil {
		configPath := filepath.Join(homeDir, ".config", "clonr", "client.json")
		if data, err := os.ReadFile(configPath); err == nil {
			var cfg ClientConfig
			if err := json.Unmarshal(data, &cfg); err == nil && cfg.ServerAddress != "" {
				// Verify the configured server is actually running
				if isServerRunning(cfg.ServerAddress) {
					return cfg.ServerAddress, nil
				}
			}
		}
	}

	// 5. Default fallback
	return "localhost:50051", nil
}

// isServerRunning checks if a gRPC server is running at the given address
func isServerRunning(address string) bool {
	// First, quick TCP port check
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()

	// Port is open, now verify it's actually a healthy gRPC server using health check (per guide)
	grpcConn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false
	}
	defer grpcConn.Close()

	// Use standard health check protocol instead of custom Ping
	healthClient := healthpb.NewHealthClient(grpcConn)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return false
	}

	return resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
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
