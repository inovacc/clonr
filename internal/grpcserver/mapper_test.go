package grpcserver

import (
	"testing"
	"time"

	"github.com/inovacc/clonr/internal/model"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestModelToProtoRepository(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	repo := &model.Repository{
		ID:          123,
		UID:         "test-uid-456",
		URL:         "https://github.com/user/repo",
		Path:        "/home/user/repos/repo",
		Favorite:    true,
		ClonedAt:    now,
		UpdatedAt:   now.Add(time.Hour),
		LastChecked: now.Add(2 * time.Hour),
	}

	proto := ModelToProtoRepository(repo)

	if proto.GetId() != 123 {
		t.Errorf("Id = %d, want %d", proto.GetId(), 123)
	}

	if proto.GetUid() != "test-uid-456" {
		t.Errorf("Uid = %q, want %q", proto.GetUid(), "test-uid-456")
	}

	if proto.GetUrl() != "https://github.com/user/repo" {
		t.Errorf("Url = %q, want %q", proto.GetUrl(), "https://github.com/user/repo")
	}

	if proto.GetPath() != "/home/user/repos/repo" {
		t.Errorf("Path = %q, want %q", proto.GetPath(), "/home/user/repos/repo")
	}

	if !proto.GetFavorite() {
		t.Error("Favorite = false, want true")
	}

	if !proto.GetClonedAt().AsTime().Equal(now) {
		t.Errorf("ClonedAt = %v, want %v", proto.GetClonedAt().AsTime(), now)
	}

	if !proto.GetUpdatedAt().AsTime().Equal(now.Add(time.Hour)) {
		t.Errorf("UpdatedAt = %v, want %v", proto.GetUpdatedAt().AsTime(), now.Add(time.Hour))
	}

	if !proto.GetLastChecked().AsTime().Equal(now.Add(2 * time.Hour)) {
		t.Errorf("LastChecked = %v, want %v", proto.GetLastChecked().AsTime(), now.Add(2*time.Hour))
	}
}

func TestModelToProtoRepository_Nil(t *testing.T) {
	proto := ModelToProtoRepository(nil)

	if proto != nil {
		t.Error("ModelToProtoRepository(nil) should return nil")
	}
}

func TestProtoToModelRepository(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	proto := &v1.Repository{
		Id:          456,
		Uid:         "proto-uid-789",
		Url:         "https://gitlab.com/org/project",
		Path:        "/opt/repos/project",
		Favorite:    false,
		ClonedAt:    timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now.Add(time.Hour)),
		LastChecked: timestamppb.New(now.Add(2 * time.Hour)),
	}

	repo := ProtoToModelRepository(proto)

	if repo.ID != 456 {
		t.Errorf("ID = %d, want %d", repo.ID, 456)
	}

	if repo.UID != "proto-uid-789" {
		t.Errorf("UID = %q, want %q", repo.UID, "proto-uid-789")
	}

	if repo.URL != "https://gitlab.com/org/project" {
		t.Errorf("URL = %q, want %q", repo.URL, "https://gitlab.com/org/project")
	}

	if repo.Path != "/opt/repos/project" {
		t.Errorf("Path = %q, want %q", repo.Path, "/opt/repos/project")
	}

	if repo.Favorite {
		t.Error("Favorite = true, want false")
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

func TestProtoToModelRepository_Nil(t *testing.T) {
	repo := ProtoToModelRepository(nil)

	if repo.ID != 0 {
		t.Errorf("ID = %d, want 0 for nil input", repo.ID)
	}

	if repo.UID != "" {
		t.Errorf("UID = %q, want empty for nil input", repo.UID)
	}
}

func TestModelToProtoConfig(t *testing.T) {
	cfg := &model.Config{
		DefaultCloneDir: "/home/user/repos",
		Editor:          "nvim",
		Terminal:        "kitty",
		MonitorInterval: 600,
		ServerPort:      8080,
	}

	proto := ModelToProtoConfig(cfg)

	if proto.GetDefaultCloneDir() != "/home/user/repos" {
		t.Errorf("DefaultCloneDir = %q, want %q", proto.GetDefaultCloneDir(), "/home/user/repos")
	}

	if proto.GetEditor() != "nvim" {
		t.Errorf("Editor = %q, want %q", proto.GetEditor(), "nvim")
	}

	if proto.GetTerminal() != "kitty" {
		t.Errorf("Terminal = %q, want %q", proto.GetTerminal(), "kitty")
	}

	if proto.GetMonitorInterval() != 600 {
		t.Errorf("MonitorInterval = %d, want %d", proto.GetMonitorInterval(), 600)
	}

	if proto.GetServerPort() != 8080 {
		t.Errorf("ServerPort = %d, want %d", proto.GetServerPort(), 8080)
	}
}

func TestModelToProtoConfig_Nil(t *testing.T) {
	proto := ModelToProtoConfig(nil)

	if proto != nil {
		t.Error("ModelToProtoConfig(nil) should return nil")
	}
}

func TestProtoToModelConfig(t *testing.T) {
	proto := &v1.Config{
		DefaultCloneDir: "/custom/path",
		Editor:          "code",
		Terminal:        "alacritty",
		MonitorInterval: 300,
		ServerPort:      50051,
	}

	cfg := ProtoToModelConfig(proto)

	if cfg.DefaultCloneDir != "/custom/path" {
		t.Errorf("DefaultCloneDir = %q, want %q", cfg.DefaultCloneDir, "/custom/path")
	}

	if cfg.Editor != "code" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "code")
	}

	if cfg.Terminal != "alacritty" {
		t.Errorf("Terminal = %q, want %q", cfg.Terminal, "alacritty")
	}

	if cfg.MonitorInterval != 300 {
		t.Errorf("MonitorInterval = %d, want %d", cfg.MonitorInterval, 300)
	}

	if cfg.ServerPort != 50051 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 50051)
	}
}

func TestProtoToModelConfig_Nil(t *testing.T) {
	cfg := ProtoToModelConfig(nil)

	if cfg != nil {
		t.Error("ProtoToModelConfig(nil) should return nil")
	}
}

func TestRoundTripRepository(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	original := &model.Repository{
		ID:          999,
		UID:         "roundtrip-uid",
		URL:         "https://github.com/test/roundtrip",
		Path:        "/test/roundtrip",
		Favorite:    true,
		ClonedAt:    now,
		UpdatedAt:   now,
		LastChecked: now,
	}

	// Convert to proto and back
	proto := ModelToProtoRepository(original)
	result := ProtoToModelRepository(proto)

	if result.ID != original.ID {
		t.Errorf("ID roundtrip: got %d, want %d", result.ID, original.ID)
	}

	if result.UID != original.UID {
		t.Errorf("UID roundtrip: got %q, want %q", result.UID, original.UID)
	}

	if result.URL != original.URL {
		t.Errorf("URL roundtrip: got %q, want %q", result.URL, original.URL)
	}

	if result.Path != original.Path {
		t.Errorf("Path roundtrip: got %q, want %q", result.Path, original.Path)
	}

	if result.Favorite != original.Favorite {
		t.Errorf("Favorite roundtrip: got %v, want %v", result.Favorite, original.Favorite)
	}

	if !result.ClonedAt.Equal(original.ClonedAt) {
		t.Errorf("ClonedAt roundtrip: got %v, want %v", result.ClonedAt, original.ClonedAt)
	}
}

func TestRoundTripConfig(t *testing.T) {
	original := &model.Config{
		DefaultCloneDir: "/test/config",
		Editor:          "vim",
		Terminal:        "xterm",
		MonitorInterval: 120,
		ServerPort:      9999,
	}

	// Convert to proto and back
	proto := ModelToProtoConfig(original)
	result := ProtoToModelConfig(proto)

	if result.DefaultCloneDir != original.DefaultCloneDir {
		t.Errorf("DefaultCloneDir roundtrip: got %q, want %q", result.DefaultCloneDir, original.DefaultCloneDir)
	}

	if result.Editor != original.Editor {
		t.Errorf("Editor roundtrip: got %q, want %q", result.Editor, original.Editor)
	}

	if result.Terminal != original.Terminal {
		t.Errorf("Terminal roundtrip: got %q, want %q", result.Terminal, original.Terminal)
	}

	if result.MonitorInterval != original.MonitorInterval {
		t.Errorf("MonitorInterval roundtrip: got %d, want %d", result.MonitorInterval, original.MonitorInterval)
	}

	if result.ServerPort != original.ServerPort {
		t.Errorf("ServerPort roundtrip: got %d, want %d", result.ServerPort, original.ServerPort)
	}
}
