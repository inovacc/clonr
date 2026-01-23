package grpcclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsClonrProcessRunning_InvalidPID(t *testing.T) {
	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{"zero pid", 0, false},
		{"negative pid", -1, false},
		{"very negative pid", -999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClonrProcessRunning(tt.pid); got != tt.want {
				t.Errorf("isClonrProcessRunning(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}
}

func TestIsClonrProcessRunning_NonExistentPID(t *testing.T) {
	// Use a very high PID that's unlikely to exist
	nonExistentPID := 999999999
	if isClonrProcessRunning(nonExistentPID) {
		t.Errorf("isClonrProcessRunning(%d) = true, want false for non-existent PID", nonExistentPID)
	}
}

func TestClientConfig_Fields(t *testing.T) {
	cfg := ClientConfig{
		ServerAddress:  "localhost:50051",
		TimeoutSeconds: 30,
	}

	if cfg.ServerAddress != "localhost:50051" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "localhost:50051")
	}

	if cfg.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want %d", cfg.TimeoutSeconds, 30)
	}
}

func TestClientConfig_ZeroValue(t *testing.T) {
	var cfg ClientConfig

	if cfg.ServerAddress != "" {
		t.Errorf("zero ClientConfig.ServerAddress = %q, want empty", cfg.ServerAddress)
	}

	if cfg.TimeoutSeconds != 0 {
		t.Errorf("zero ClientConfig.TimeoutSeconds = %d, want 0", cfg.TimeoutSeconds)
	}
}

func TestServerInfo_Fields(t *testing.T) {
	now := time.Now()

	info := ServerInfo{
		Address:   "localhost:8080",
		Port:      8080,
		PID:       12345,
		StartedAt: now,
	}

	if info.Address != "localhost:8080" {
		t.Errorf("Address = %q, want %q", info.Address, "localhost:8080")
	}

	if info.Port != 8080 {
		t.Errorf("Port = %d, want %d", info.Port, 8080)
	}

	if info.PID != 12345 {
		t.Errorf("PID = %d, want %d", info.PID, 12345)
	}

	if !info.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", info.StartedAt, now)
	}
}

func TestServerInfo_JSONMarshaling(t *testing.T) {
	original := ServerInfo{
		Address:   "localhost:50051",
		Port:      50051,
		PID:       12345,
		StartedAt: time.Now().Truncate(time.Second),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ServerInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Address != original.Address {
		t.Errorf("Address = %q, want %q", decoded.Address, original.Address)
	}

	if decoded.Port != original.Port {
		t.Errorf("Port = %d, want %d", decoded.Port, original.Port)
	}

	if decoded.PID != original.PID {
		t.Errorf("PID = %d, want %d", decoded.PID, original.PID)
	}
}

func TestClientConfig_JSONMarshaling(t *testing.T) {
	original := ClientConfig{
		ServerAddress:  "localhost:8080",
		TimeoutSeconds: 60,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ClientConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ServerAddress != original.ServerAddress {
		t.Errorf("ServerAddress = %q, want %q", decoded.ServerAddress, original.ServerAddress)
	}

	if decoded.TimeoutSeconds != original.TimeoutSeconds {
		t.Errorf("TimeoutSeconds = %d, want %d", decoded.TimeoutSeconds, original.TimeoutSeconds)
	}
}

func TestSaveServerAddress(t *testing.T) {
	// Create a temporary directory for the test
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".config", "clonr")
	configPath := filepath.Join(configDir, "client.json")

	// Backup existing config if present
	var backup []byte
	if data, err := os.ReadFile(configPath); err == nil {
		backup = data
	}

	// Cleanup after test
	defer func() {
		if backup != nil {
			_ = os.WriteFile(configPath, backup, 0644)
		} else {
			_ = os.Remove(configPath)
		}
	}()

	// Test saving address
	testAddr := "testhost:12345"
	if err := SaveServerAddress(testAddr); err != nil {
		t.Fatalf("SaveServerAddress() error = %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if cfg.ServerAddress != testAddr {
		t.Errorf("Saved address = %q, want %q", cfg.ServerAddress, testAddr)
	}

	if cfg.TimeoutSeconds != 30 {
		t.Errorf("Saved timeout = %d, want 30", cfg.TimeoutSeconds)
	}
}

func TestDiscoverServerAddress_EnvVariable(t *testing.T) {
	// Set environment variable
	testAddr := "envhost:99999"
	t.Setenv("CLONR_SERVER", testAddr)

	addr := discoverServerAddress()
	if addr != testAddr {
		t.Errorf("discoverServerAddress() = %q, want %q", addr, testAddr)
	}
}

func TestDiscoverServerAddress_Default(t *testing.T) {
	// Ensure no environment variable is set
	t.Setenv("CLONR_SERVER", "")

	// Remove any server.json file temporarily
	dataDir, err := os.UserCacheDir()
	if err == nil {
		serverInfoPath := filepath.Join(dataDir, "clonr", "server.json")

		if data, err := os.ReadFile(serverInfoPath); err == nil {
			defer func() {
				_ = os.WriteFile(serverInfoPath, data, 0644)
			}()

			_ = os.Remove(serverInfoPath)
		}
	}

	// When no server is running and no config, should return default
	addr := discoverServerAddress()

	// Should be default or a discovered server
	if addr == "" {
		t.Error("discoverServerAddress() returned empty string")
	}
}

func TestIsServerRunning_InvalidAddress(t *testing.T) {
	// Test with invalid/unreachable addresses
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{"empty address", "", false},
		{"unreachable port", "localhost:65000", false},
		{"invalid host", "nonexistent.invalid.host:50051", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isServerRunning(tt.address); got != tt.want {
				t.Errorf("isServerRunning(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}
