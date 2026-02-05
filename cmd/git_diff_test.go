package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunGitDiff_WithUnstagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify existing file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitDiffCmd.Flags().Set("staged", "false")
	gitDiffCmd.Flags().Set("cached", "false")
	gitDiffCmd.Flags().Set("stat", "false")
	gitDiffCmd.Flags().Set("name-only", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithStagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify and stage file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified content\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	cmd := exec.Command("git", "add", "README.md")

	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	gitDiffCmd.Flags().Set("staged", "true")
	defer gitDiffCmd.Flags().Set("staged", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithCachedFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify and stage file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	cmd := exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	_ = cmd.Run()

	gitDiffCmd.Flags().Set("cached", "true")
	defer gitDiffCmd.Flags().Set("cached", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithStatFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitDiffCmd.Flags().Set("stat", "true")
	defer gitDiffCmd.Flags().Set("stat", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithNameOnlyFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitDiffCmd.Flags().Set("name-only", "true")
	defer gitDiffCmd.Flags().Set("name-only", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithNameStatusFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitDiffCmd.Flags().Set("name-status", "true")
	defer gitDiffCmd.Flags().Set("name-status", "false")

	err := runGitDiff(gitDiffCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithCommitArg(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create another commit
	testFile := filepath.Join(repoDir, "second.txt")
	if err := os.WriteFile(testFile, []byte("second\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "second.txt")
	cmd.Dir = repoDir
	_ = cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Reset flags
	gitDiffCmd.Flags().Set("staged", "false")
	gitDiffCmd.Flags().Set("cached", "false")

	err := runGitDiff(gitDiffCmd, []string{"HEAD~1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitDiff_WithPathArg(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Modify file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}

	gitDiffCmd.Flags().Set("staged", "false")

	err := runGitDiff(gitDiffCmd, []string{"--", "README.md"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
