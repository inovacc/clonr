package grpcclient

import (
	"testing"
	"time"

	"github.com/inovacc/clonr/internal/model"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHandleGRPCError_Nil(t *testing.T) {
	err := handleGRPCError(nil)
	if err != nil {
		t.Errorf("handleGRPCError(nil) = %v, want nil", err)
	}
}

func TestHandleGRPCError_InvalidArgument(t *testing.T) {
	grpcErr := status.Error(codes.InvalidArgument, "bad input")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for InvalidArgument")
	}

	expected := "invalid input: bad input"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestHandleGRPCError_AlreadyExists(t *testing.T) {
	grpcErr := status.Error(codes.AlreadyExists, "duplicate entry")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for AlreadyExists")
	}

	expected := "already exists: duplicate entry"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestHandleGRPCError_NotFound(t *testing.T) {
	grpcErr := status.Error(codes.NotFound, "resource missing")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for NotFound")
	}

	expected := "not found: resource missing"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestHandleGRPCError_Unavailable(t *testing.T) {
	grpcErr := status.Error(codes.Unavailable, "server down")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for Unavailable")
	}

	// Should contain helpful message about starting server
	if err.Error() == "" {
		t.Error("handleGRPCError() should return non-empty error message")
	}
}

func TestHandleGRPCError_DeadlineExceeded(t *testing.T) {
	grpcErr := status.Error(codes.DeadlineExceeded, "timeout")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for DeadlineExceeded")
	}

	expected := "request timeout: timeout"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestHandleGRPCError_Canceled(t *testing.T) {
	grpcErr := status.Error(codes.Canceled, "user canceled")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for Canceled")
	}

	expected := "request canceled: user canceled"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestHandleGRPCError_DefaultCode(t *testing.T) {
	grpcErr := status.Error(codes.Internal, "internal error")

	err := handleGRPCError(grpcErr)
	if err == nil {
		t.Fatal("handleGRPCError should return error for Internal")
	}

	expected := "server error: internal error"
	if err.Error() != expected {
		t.Errorf("handleGRPCError() = %q, want %q", err.Error(), expected)
	}
}

func TestProtoToModelRepository(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	proto := &v1.Repository{
		Id:          123,
		Uid:         "test-uid",
		Url:         "https://github.com/user/repo",
		Path:        "/home/user/repo",
		Favorite:    true,
		ClonedAt:    timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now.Add(time.Hour)),
		LastChecked: timestamppb.New(now.Add(2 * time.Hour)),
	}

	repo := protoToModelRepository(proto)

	if repo.ID != 123 {
		t.Errorf("ID = %d, want %d", repo.ID, 123)
	}

	if repo.UID != "test-uid" {
		t.Errorf("UID = %q, want %q", repo.UID, "test-uid")
	}

	if repo.URL != "https://github.com/user/repo" {
		t.Errorf("URL = %q, want %q", repo.URL, "https://github.com/user/repo")
	}

	if repo.Path != "/home/user/repo" {
		t.Errorf("Path = %q, want %q", repo.Path, "/home/user/repo")
	}

	if !repo.Favorite {
		t.Error("Favorite = false, want true")
	}

	if !repo.ClonedAt.Equal(now) {
		t.Errorf("ClonedAt = %v, want %v", repo.ClonedAt, now)
	}

	if !repo.UpdatedAt.Equal(now.Add(time.Hour)) {
		t.Errorf("UpdatedAt = %v, want %v", repo.UpdatedAt, now.Add(time.Hour))
	}

	if !repo.LastChecked.Equal(now.Add(2 * time.Hour)) {
		t.Errorf("LastChecked = %v, want %v", repo.LastChecked, now.Add(2*time.Hour))
	}
}

func TestProtoToModelConfig(t *testing.T) {
	proto := &v1.Config{
		DefaultCloneDir: "/custom/dir",
		Editor:          "nvim",
		Terminal:        "alacritty",
		MonitorInterval: 600,
		ServerPort:      8080,
	}

	cfg := protoToModelConfig(proto)

	if cfg.DefaultCloneDir != "/custom/dir" {
		t.Errorf("DefaultCloneDir = %q, want %q", cfg.DefaultCloneDir, "/custom/dir")
	}

	if cfg.Editor != "nvim" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "nvim")
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

func TestModelToProtoConfig(t *testing.T) {
	cfg := &model.Config{
		DefaultCloneDir: "/test/path",
		Editor:          "code",
		Terminal:        "kitty",
		MonitorInterval: 300,
		ServerPort:      50051,
	}

	proto := modelToProtoConfig(cfg)

	if proto.GetDefaultCloneDir() != "/test/path" {
		t.Errorf("DefaultCloneDir = %q, want %q", proto.GetDefaultCloneDir(), "/test/path")
	}

	if proto.GetEditor() != "code" {
		t.Errorf("Editor = %q, want %q", proto.GetEditor(), "code")
	}

	if proto.GetTerminal() != "kitty" {
		t.Errorf("Terminal = %q, want %q", proto.GetTerminal(), "kitty")
	}

	if proto.GetMonitorInterval() != 300 {
		t.Errorf("MonitorInterval = %d, want %d", proto.GetMonitorInterval(), 300)
	}

	if proto.GetServerPort() != 50051 {
		t.Errorf("ServerPort = %d, want %d", proto.GetServerPort(), 50051)
	}
}

func TestClientConfigStruct(t *testing.T) {
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

func TestServerInfoStruct(t *testing.T) {
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
