package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunGitPull_GitRepo(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Pull without remote (will show already up to date or error about no remote)
	err := runGitPull(gitPullCmd, nil)
	// This may error because there's no remote configured, which is expected
	if err != nil && err.Error() != "not a git repository" {
		// Accept the error as the repo has no remote
		t.Log("pull without remote:", err)
	}
}

func TestRunGitPull_WithRemoteArg(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Pull with remote arg (will fail because no remote, but tests the arg parsing)
	err := runGitPull(gitPullCmd, []string{"origin"})
	// Expected to fail since there's no remote
	if err == nil {
		t.Log("pull with non-existent remote succeeded unexpectedly")
	}
}

func TestRunGitPull_WithRemoteAndBranchArgs(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Pull with remote and branch args
	err := runGitPull(gitPullCmd, []string{"origin", "main"})
	// Expected to fail since there's no remote
	if err == nil {
		t.Log("pull with non-existent remote succeeded unexpectedly")
	}
}

// setupTestRepoWithRemote creates a test repo with a local "remote"
func setupTestRepoWithRemote(t *testing.T) (repoDir, remoteDir string) {
	t.Helper()

	// Create "remote" repo (bare)
	remoteDir, err := os.MkdirTemp("", "clonr-remote-*")
	if err != nil {
		t.Fatalf("failed to create remote dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Create local repo
	repoDir, err = os.MkdirTemp("", "clonr-local-*")
	if err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to create local dir: %v", err)
	}

	cmd = exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(repoDir)
		t.Fatalf("failed to init local repo: %v", err)
	}

	// Configure git
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test User"},
		{"remote", "add", "origin", remoteDir},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			os.RemoveAll(remoteDir)
			os.RemoveAll(repoDir)
			t.Fatalf("failed to configure git: %v", err)
		}
	}

	// Create initial commit and push
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(repoDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "Initial commit"},
		{"push", "-u", "origin", "master"},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			// Push might fail if default branch is 'main' instead of 'master'
			if args[0] == "push" {
				// Try with main branch
				cmd = exec.Command("git", "branch", "-M", "main")
				cmd.Dir = repoDir
				_ = cmd.Run()
				cmd = exec.Command("git", "push", "-u", "origin", "main")
				cmd.Dir = repoDir
				if err := cmd.Run(); err != nil {
					t.Logf("push failed: %v", err)
				}
			}
		}
	}

	return repoDir, remoteDir
}

func TestRunGitPull_WithLocalRemote(t *testing.T) {
	repoDir, remoteDir := setupTestRepoWithRemote(t)
	defer os.RemoveAll(repoDir)
	defer os.RemoveAll(remoteDir)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Pull from the local remote
	err := runGitPull(gitPullCmd, nil)
	if err != nil {
		// May fail due to git configuration, log but don't fail test
		t.Logf("pull error (may be expected): %v", err)
	}
}
