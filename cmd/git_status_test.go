package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunGitStatus_WithStagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create and stage a new file
	testFile := filepath.Join(repoDir, "staged.txt")
	if err := os.WriteFile(testFile, []byte("staged content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "staged.txt")

	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")
	gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_WithUnstagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify existing file without staging
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")
	gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_WithUntrackedFiles(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create untracked file
	testFile := filepath.Join(repoDir, "untracked.txt")
	if err := os.WriteFile(testFile, []byte("untracked content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")
	gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_WithBranchFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")

	gitStatusCmd.Flags().Set("branch", "true")
	defer gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_MixedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create staged file
	stagedFile := filepath.Join(repoDir, "staged.txt")
	if err := os.WriteFile(stagedFile, []byte("staged\n"), 0644); err != nil {
		t.Fatalf("failed to create staged file: %v", err)
	}

	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Modify existing file (unstaged)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Create untracked file
	untrackedFile := filepath.Join(repoDir, "untracked.txt")
	if err := os.WriteFile(untrackedFile, []byte("untracked\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	gitStatusCmd.Flags().Set("short", "false")
	gitStatusCmd.Flags().Set("porcelain", "false")
	gitStatusCmd.Flags().Set("branch", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitStatus_ShortWithChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create untracked file
	testFile := filepath.Join(repoDir, "new.txt")
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	gitStatusCmd.Flags().Set("short", "true")

	gitStatusCmd.Flags().Set("porcelain", "false")
	defer gitStatusCmd.Flags().Set("short", "false")

	err := runGitStatus(gitStatusCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
