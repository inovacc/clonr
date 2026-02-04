package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunGitLog_WithLimitFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create multiple commits
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(repoDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoDir
		_ = cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Commit "+string(rune('0'+i)))
		cmd.Dir = repoDir
		_ = cmd.Run()
	}

	gitLogCmd.Flags().Set("limit", "3")
	gitLogCmd.Flags().Set("oneline", "false")
	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("limit", "10")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_WithAuthorFilter(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitLogCmd.Flags().Set("author", "Test User")
	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("author", "")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_WithGrepFilter(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitLogCmd.Flags().Set("grep", "Initial")
	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("grep", "")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_EmptyResult(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Search for non-existent author
	gitLogCmd.Flags().Set("author", "NonExistentAuthor12345")
	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("author", "")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_OnelineEmptyResult(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitLogCmd.Flags().Set("oneline", "true")
	gitLogCmd.Flags().Set("author", "NonExistentAuthor12345")
	gitLogCmd.Flags().Set("json", "false")
	defer func() {
		gitLogCmd.Flags().Set("oneline", "false")
		gitLogCmd.Flags().Set("author", "")
	}()

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitLog_WithAllFlag(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create another branch with a commit
	cmd := exec.Command("git", "branch", "other-branch")
	cmd.Dir = repoDir
	_ = cmd.Run()

	gitLogCmd.Flags().Set("all", "true")
	gitLogCmd.Flags().Set("json", "false")
	defer gitLogCmd.Flags().Set("all", "false")

	err := runGitLog(gitLogCmd, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
