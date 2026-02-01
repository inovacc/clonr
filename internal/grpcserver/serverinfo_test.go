package grpcserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inovacc/clonr/internal/application"
)

func TestGetServerInfoPath(t *testing.T) {
	path, err := getServerInfoPath()
	if err != nil {
		t.Fatalf("getServerInfoPath() error = %v", err)
	}

	if path == "" {
		t.Error("getServerInfoPath() returned empty string")
	}

	// Should end with server.json
	if filepath.Base(path) != "server.json" {
		t.Errorf("getServerInfoPath() = %q, want to end with server.json", path)
	}

	// Should contain clonr directory
	if filepath.Base(filepath.Dir(path)) != application.AppName {
		t.Errorf("getServerInfoPath() parent dir = %q, want clonr", filepath.Base(filepath.Dir(path)))
	}
}

func TestReadServerInfo_NoFile(t *testing.T) {
	// Ensure no server.json exists
	path, err := getServerInfoPath()
	if err != nil {
		t.Fatalf("getServerInfoPath() error = %v", err)
	}

	// Remove if exists
	_ = os.Remove(path)

	info, err := ReadServerInfo()
	if err != ErrNoServerInfo {
		t.Errorf("ReadServerInfo() error = %v, want ErrNoServerInfo", err)
	}

	if info != nil {
		t.Error("ReadServerInfo() returned non-nil info when file doesn't exist")
	}
}

func TestWriteAndReadServerInfo(t *testing.T) {
	// Clean up before and after
	path, _ := getServerInfoPath()
	_ = os.Remove(path)

	defer func() {
		_ = os.Remove(path)
	}()

	testPort := 55555

	// Write server info
	if err := WriteServerInfo(testPort); err != nil {
		t.Fatalf("WriteServerInfo() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("WriteServerInfo() did not create file")
	}

	// Read server info
	info, err := ReadServerInfo()
	if err != nil {
		t.Fatalf("ReadServerInfo() error = %v", err)
	}

	// Verify fields
	if info.Port != testPort {
		t.Errorf("ServerInfo.Port = %d, want %d", info.Port, testPort)
	}

	expectedAddr := "localhost:55555"
	if info.Address != expectedAddr {
		t.Errorf("ServerInfo.Address = %q, want %q", info.Address, expectedAddr)
	}

	if info.PID <= 0 {
		t.Errorf("ServerInfo.PID = %d, want > 0", info.PID)
	}

	if info.StartedAt.IsZero() {
		t.Error("ServerInfo.StartedAt is zero")
	}

	// StartedAt should be recent
	if time.Since(info.StartedAt) > time.Minute {
		t.Error("ServerInfo.StartedAt is not recent")
	}
}

func TestRemoveServerInfo(t *testing.T) {
	path, _ := getServerInfoPath()

	// Write server info first
	if err := WriteServerInfo(50051); err != nil {
		t.Fatalf("WriteServerInfo() error = %v", err)
	}

	// Verify exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("server.json should exist before removal")
	}

	// Remove
	RemoveServerInfo()

	// Verify removed
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("RemoveServerInfo() did not remove the file")
	}
}

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
			if got := IsClonrProcessRunning(tt.pid); got != tt.want {
				t.Errorf("IsClonrProcessRunning(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}
}

func TestIsClonrProcessRunning_NonExistentPID(t *testing.T) {
	// Use a very high PID that's unlikely to exist
	nonExistentPID := 999999999

	if IsClonrProcessRunning(nonExistentPID) {
		t.Errorf("IsClonrProcessRunning(%d) = true, want false for non-existent PID", nonExistentPID)
	}
}

func TestIsServerRunning_NoServerInfo(t *testing.T) {
	// Ensure no server.json exists
	path, _ := getServerInfoPath()
	_ = os.Remove(path)

	info := IsServerRunning()
	if info != nil {
		t.Error("IsServerRunning() should return nil when no server.json exists")
	}
}

func TestIsServerRunning_StaleServerInfo(t *testing.T) {
	path, _ := getServerInfoPath()

	defer func() {
		_ = os.Remove(path)
	}()

	// Write server info with a non-existent PID
	staleInfo := ServerInfo{
		Address:   "localhost:50051",
		Port:      50051,
		PID:       999999999, // Non-existent PID
		StartedAt: time.Now().Add(-time.Hour),
	}

	data, err := json.MarshalIndent(staleInfo, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal stale info: %v", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write stale server info: %v", err)
	}

	// IsServerRunning should return nil and clean up stale file
	info := IsServerRunning()
	if info != nil {
		t.Error("IsServerRunning() should return nil for stale server info")
	}

	// File should be removed
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("IsServerRunning() should clean up stale server.json")
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

func TestErrNoServerInfo(t *testing.T) {
	if ErrNoServerInfo == nil {
		t.Error("ErrNoServerInfo should not be nil")
	}

	expectedMsg := "no server info file"
	if ErrNoServerInfo.Error() != expectedMsg {
		t.Errorf("ErrNoServerInfo.Error() = %q, want %q", ErrNoServerInfo.Error(), expectedMsg)
	}
}
