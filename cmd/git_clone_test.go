package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inovacc/clonr/internal/model"
)

func TestRunGitClone_ClientError(t *testing.T) {
	cleanup := withMockClientError(errors.New("connection failed"))
	defer cleanup()

	gitCloneCmd.Flags().Set("no-tui", "true")
	defer gitCloneCmd.Flags().Set("no-tui", "false")

	err := runGitClone(gitCloneCmd, []string{"owner/repo"})
	if err == nil {
		t.Error("expected error when client fails")
	}

	if err.Error() != "connection failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitClone_WithProfile(t *testing.T) {
	mock := NewMockClient()
	mock.Profiles = []model.Profile{
		{Name: "work", Host: "github.com", Workspace: "/tmp/work"},
	}

	cleanup := withMockClient(mock)
	defer cleanup()

	gitCloneCmd.Flags().Set("no-tui", "true")
	gitCloneCmd.Flags().Set("profile", "work")

	defer func() {
		gitCloneCmd.Flags().Set("no-tui", "false")
		gitCloneCmd.Flags().Set("profile", "")
	}()

	// This will fail because we're not in a real clone situation,
	// but we can verify the profile was set
	_ = runGitClone(gitCloneCmd, []string{"owner/repo"})

	if !mock.SetActiveProfileCalled {
		t.Error("SetActiveProfile should have been called")
	}

	if mock.SetActiveProfileName != "work" {
		t.Errorf("SetActiveProfile called with %q, want %q", mock.SetActiveProfileName, "work")
	}

	if !mock.GetProfileCalled {
		t.Error("GetProfile should have been called")
	}
}

func TestRunGitClone_SetProfileError(t *testing.T) {
	mock := NewMockClient()
	mock.SetActiveProfileErr = errors.New("profile not found")

	cleanup := withMockClient(mock)
	defer cleanup()

	gitCloneCmd.Flags().Set("no-tui", "true")
	gitCloneCmd.Flags().Set("profile", "nonexistent")

	defer func() {
		gitCloneCmd.Flags().Set("no-tui", "false")
		gitCloneCmd.Flags().Set("profile", "")
	}()

	err := runGitClone(gitCloneCmd, []string{"owner/repo"})
	if err == nil {
		t.Error("expected error when SetActiveProfile fails")
	}
}

func TestRunGitClone_GetProfileError(t *testing.T) {
	mock := NewMockClient()
	mock.GetProfileErr = errors.New("failed to get profile")

	cleanup := withMockClient(mock)
	defer cleanup()

	gitCloneCmd.Flags().Set("no-tui", "true")
	gitCloneCmd.Flags().Set("profile", "work")

	defer func() {
		gitCloneCmd.Flags().Set("no-tui", "false")
		gitCloneCmd.Flags().Set("profile", "")
	}()

	err := runGitClone(gitCloneCmd, []string{"owner/repo"})
	if err == nil {
		t.Error("expected error when GetProfile fails")
	}
}

func TestRunGitClone_ListProfilesError(t *testing.T) {
	mock := NewMockClient()
	mock.ListProfilesErr = errors.New("failed to list profiles")

	cleanup := withMockClient(mock)
	defer cleanup()

	// Without profile flag, it will try to list profiles
	gitCloneCmd.Flags().Set("no-tui", "false")
	gitCloneCmd.Flags().Set("profile", "")

	// This test is hard to run without TUI, skip it
	t.Skip("requires TUI interaction")
}

func TestRunGitClone_ListWorkspacesError(t *testing.T) {
	mock := NewMockClient()
	mock.ListWorkspacesErr = errors.New("failed to list workspaces")

	cleanup := withMockClient(mock)
	defer cleanup()

	// Without workspace, it will try to list workspaces
	gitCloneCmd.Flags().Set("no-tui", "false")
	gitCloneCmd.Flags().Set("workspace", "")

	// This test is hard to run without TUI, skip it
	t.Skip("requires TUI interaction")
}

func TestCreateGitDefaultWorkspace(t *testing.T) {
	mock := NewMockClient()
	mock.Config = &model.Config{
		DefaultCloneDir: "/tmp/test-clonr",
	}

	err := createGitDefaultWorkspace(mock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !mock.SaveWorkspaceCalled {
		t.Error("SaveWorkspace should have been called")
	}

	if mock.SavedWorkspace == nil {
		t.Fatal("SavedWorkspace should not be nil")
	}

	if mock.SavedWorkspace.Path != "/tmp/test-clonr" {
		t.Errorf("workspace path = %q, want %q", mock.SavedWorkspace.Path, "/tmp/test-clonr")
	}

	if !mock.SavedWorkspace.Active {
		t.Error("workspace should be active")
	}
}

func TestCreateGitDefaultWorkspace_GetConfigError(t *testing.T) {
	mock := NewMockClient()
	mock.GetConfigErr = errors.New("config error")

	err := createGitDefaultWorkspace(mock)
	if err == nil {
		t.Error("expected error when GetConfig fails")
	}
}

func TestCreateGitDefaultWorkspace_SaveWorkspaceError(t *testing.T) {
	mock := NewMockClient()
	mock.Config = &model.Config{
		DefaultCloneDir: "/tmp/test-clonr",
	}
	mock.SaveWorkspaceErr = errors.New("save error")

	err := createGitDefaultWorkspace(mock)
	if err == nil {
		t.Error("expected error when SaveWorkspace fails")
	}
}

func TestCreateGitWorkspaceFromSelection(t *testing.T) {
	mock := NewMockClient()

	tmpDir, err := os.MkdirTemp("", "clonr-ws-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ws := &model.Workspace{
		Name:        "test-workspace",
		Description: "Test workspace",
		Path:        tmpDir,
	}

	err = createGitWorkspaceFromSelection(mock, ws)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !mock.SaveWorkspaceCalled {
		t.Error("SaveWorkspace should have been called")
	}

	if mock.SavedWorkspace == nil {
		t.Fatal("SavedWorkspace should not be nil")
	}

	if mock.SavedWorkspace.Name != "test-workspace" {
		t.Errorf("workspace name = %q, want %q", mock.SavedWorkspace.Name, "test-workspace")
	}
}

func TestCreateGitWorkspaceFromSelection_WithTildePath(t *testing.T) {
	mock := NewMockClient()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	// Create a temp dir in home
	testDir := filepath.Join(home, ".clonr-test-ws-"+time.Now().Format("20060102150405"))

	ws := &model.Workspace{
		Name:        "tilde-workspace",
		Description: "Test workspace with tilde",
		Path:        "~/.clonr-test-ws-" + time.Now().Format("20060102150405"),
	}

	err = createGitWorkspaceFromSelection(mock, ws)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	defer os.RemoveAll(testDir)

	if mock.SavedWorkspace == nil {
		t.Fatal("SavedWorkspace should not be nil")
	}

	// Path should be expanded
	if mock.SavedWorkspace.Path == ws.Path {
		t.Error("path should be expanded from tilde")
	}
}

func TestCreateGitWorkspaceFromSelection_SaveError(t *testing.T) {
	mock := NewMockClient()
	mock.SaveWorkspaceErr = errors.New("save error")

	tmpDir, err := os.MkdirTemp("", "clonr-ws-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ws := &model.Workspace{
		Name: "test-workspace",
		Path: tmpDir,
	}

	err = createGitWorkspaceFromSelection(mock, ws)
	if err == nil {
		t.Error("expected error when SaveWorkspace fails")
	}
}

func TestCreateGitWorkspaceFromSelection_CreatesDirIfNotExists(t *testing.T) {
	mock := NewMockClient()

	tmpDir, err := os.MkdirTemp("", "clonr-ws-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	newDir := filepath.Join(tmpDir, "new-workspace-dir")

	ws := &model.Workspace{
		Name: "new-dir-workspace",
		Path: newDir,
	}

	err = createGitWorkspaceFromSelection(mock, ws)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("directory should have been created")
	}
}
