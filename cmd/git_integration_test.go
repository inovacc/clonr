package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")

	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")

	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")

	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")

	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")

	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return tmpDir
}

func TestRunGitStatus_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to non-git directory
	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitStatus(gitStatusCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}

	if err.Error() != "not a git repository" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_GitRepo(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Reset flags for test
	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")
	gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_ShortFormat(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitStatusCmd.Flags().Set("short", "true")
	gitStatusCmd.Flags().Set("porcelain", "false")

	gitStatusCmd.Flags().Set("branch", "false")
	defer gitStatusCmd.Flags().Set("short", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_PorcelainFormat(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "true")

	gitStatusCmd.Flags().Set("branch", "false")
	defer gitStatusCmd.Flags().Set("porcelain", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitCommit_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitCommitCmd.Flags().Set("message", "test")
	defer gitCommitCmd.Flags().Set("message", "")

	err = runGitCommit(gitCommitCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}
}

func TestRunGitCommit_NothingToCommit(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitCommitCmd.Flags().Set("message", "test commit")

	gitCommitCmd.Flags().Set("all", "false")
	defer gitCommitCmd.Flags().Set("message", "")

	// Should not error when nothing to commit
	err := runGitCommit(gitCommitCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitCommit_WithChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create a new file and stage it
	testFile := filepath.Join(repoDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "newfile.txt")

	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	gitCommitCmd.Flags().Set("message", "add new file")

	gitCommitCmd.Flags().Set("all", "false")
	defer gitCommitCmd.Flags().Set("message", "")

	err := runGitCommit(gitCommitCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitLog(gitLogCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}
}

func TestRunGitLog_GitRepo(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Reset flags
	gitLogCmd.Flags().Set("limit", "10")
	gitLogCmd.Flags().Set("oneline", "false")
	gitLogCmd.Flags().Set("json", "false")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_Oneline(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitLogCmd.Flags().Set("oneline", "true")

	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("oneline", "false")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_JSON(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitLogCmd.Flags().Set("oneline", "false")

	gitLogCmd.Flags().Set("json", "true")
	defer gitLogCmd.Flags().Set("json", "false")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitDiff(gitDiffCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}
}

func TestRunGitDiff_GitRepo(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Reset flags
	gitDiffCmd.Flags().Set("staged", "false")
	gitDiffCmd.Flags().Set("cached", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_Staged(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitDiffCmd.Flags().Set("staged", "true")
	defer gitDiffCmd.Flags().Set("staged", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitBranch_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitBranch(gitBranchCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}
}

func TestRunGitBranch_ListBranches(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Reset flags
	gitBranchCmd.Flags().Set("all", "false")
	gitBranchCmd.Flags().Set("delete", "false")
	gitBranchCmd.Flags().Set("force-delete", "false")
	gitBranchCmd.Flags().Set("json", "false")

	err := runGitBranch(gitBranchCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitBranch_ListJSON(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitBranchCmd.Flags().Set("json", "true")
	defer gitBranchCmd.Flags().Set("json", "false")

	err := runGitBranch(gitBranchCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitBranch_CreateBranch(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Reset flags
	gitBranchCmd.Flags().Set("delete", "false")
	gitBranchCmd.Flags().Set("force-delete", "false")

	err := runGitBranch(gitBranchCmd, []string{"test-branch"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify branch was created
	cmd := exec.Command("git", "branch", "--list", "test-branch")
	cmd.Dir = repoDir

	output, _ := cmd.Output()
	if len(output) == 0 {
		t.Error("branch was not created")
	}
}

func TestRunGitBranch_DeleteBranch(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// First create a branch
	cmd := exec.Command("git", "branch", "delete-me")

	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	gitBranchCmd.Flags().Set("delete", "true")
	defer gitBranchCmd.Flags().Set("delete", "false")

	err := runGitBranch(gitBranchCmd, []string{"delete-me"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitBranch_DeleteRequiresName(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitBranchCmd.Flags().Set("delete", "true")
	defer gitBranchCmd.Flags().Set("delete", "false")

	err := runGitBranch(gitBranchCmd, nil)
	if err == nil {
		t.Error("expected error when deleting without branch name")
	}
}

func TestRunGitPull_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitPull(gitPullCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}
}
