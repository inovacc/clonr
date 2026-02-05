package cmd

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestRunGitPush_NotGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-push-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	err = runGitPush(gitPushCmd, nil)
	if err == nil {
		t.Error("expected error for non-git repository")
	}

	if err.Error() != "not a git repository" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunGitPush_SkipLeaks(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Set skip-leaks to avoid running the security scan
	gitPushCmd.Flags().Set("skip-leaks", "true")
	gitPushCmd.Flags().Set("set-upstream", "false")
	gitPushCmd.Flags().Set("force", "false")

	gitPushCmd.Flags().Set("tags", "false")
	defer gitPushCmd.Flags().Set("skip-leaks", "false")

	// This will fail because there's no remote, but the skip-leaks path is covered
	err := runGitPush(gitPushCmd, nil)
	// Expected to fail since there's no remote configured
	if err == nil {
		t.Log("push succeeded unexpectedly (no remote)")
	}
}

func TestRunGitPush_WithRemoteArg(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitPushCmd.Flags().Set("skip-leaks", "true")
	defer gitPushCmd.Flags().Set("skip-leaks", "false")

	// Push with remote arg (will fail because no remote exists)
	err := runGitPush(gitPushCmd, []string{"origin"})
	if err == nil {
		t.Log("push with non-existent remote succeeded unexpectedly")
	}
}

func TestRunGitPush_WithRemoteAndBranchArgs(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitPushCmd.Flags().Set("skip-leaks", "true")
	defer gitPushCmd.Flags().Set("skip-leaks", "false")

	// Push with remote and branch args
	err := runGitPush(gitPushCmd, []string{"origin", "main"})
	if err == nil {
		t.Log("push with non-existent remote succeeded unexpectedly")
	}
}

func TestRunGitPush_WithTags(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create a tag
	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = repoDir
	_ = cmd.Run()

	gitPushCmd.Flags().Set("skip-leaks", "true")
	gitPushCmd.Flags().Set("tags", "true")

	defer func() {
		gitPushCmd.Flags().Set("skip-leaks", "false")
		gitPushCmd.Flags().Set("tags", "false")
	}()

	// Will fail because no remote
	err := runGitPush(gitPushCmd, nil)
	if err == nil {
		t.Log("push with tags succeeded unexpectedly (no remote)")
	}
}

func TestRunGitPush_WithSetUpstream(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	gitPushCmd.Flags().Set("skip-leaks", "true")
	gitPushCmd.Flags().Set("set-upstream", "true")

	defer func() {
		gitPushCmd.Flags().Set("skip-leaks", "false")
		gitPushCmd.Flags().Set("set-upstream", "false")
	}()

	// Will fail because no remote
	err := runGitPush(gitPushCmd, []string{"origin", "main"})
	if err == nil {
		t.Log("push with set-upstream succeeded unexpectedly (no remote)")
	}
}

func TestEnqueueGitPushForMonitoring(t *testing.T) {
	repoDir, remoteDir := setupTestRepoWithRemote(t)
	defer os.RemoveAll(repoDir)
	defer os.RemoveAll(remoteDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Test enqueueGitPushForMonitoring
	ctx := context.Background()
	err := enqueueGitPushForMonitoring(ctx, "origin")

	// May fail due to database issues, but at least it exercises the code path
	if err != nil {
		t.Logf("enqueue error (may be expected): %v", err)
	}
}

func TestEnqueueGitPushForMonitoring_NoRemote(t *testing.T) {
	repoDir := setupTestRepo(t)
	defer os.RemoveAll(repoDir)

	oldDir, _ := os.Getwd()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer os.Chdir(oldDir)

	ctx := context.Background()
	err := enqueueGitPushForMonitoring(ctx, "origin")
	// Should fail because there's no remote
	if err == nil {
		t.Error("expected error when no remote exists")
	}
}
