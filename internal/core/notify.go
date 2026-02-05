package core

import (
	"context"
	"os/exec"
	"strings"

	"github.com/inovacc/clonr/internal/git"
	"github.com/inovacc/clonr/internal/notify"
)

// NotifyPush sends a notification for a push event.
func NotifyPush(ctx context.Context, repoPath, remote, branch string) {
	event := notify.NewEvent(notify.EventPush)

	// Get repository info
	if repoURL, err := getRemoteURLFromPath(repoPath, remote); err == nil {
		event.WithRepository(repoURL)
		event.WithURL(repoURL)
	}

	event.WithBranch(branch)

	// Get commit info
	client := git.NewClient()
	if sha, err := client.GetHead(ctx); err == nil {
		event.Commit = sha
	}

	if msg, err := getCommitMessage(repoPath, "HEAD"); err == nil {
		event.CommitMessage = msg
	}

	// Get author
	if author, err := getGitUser(); err == nil {
		event.WithAuthor(author)
	}

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyClone sends a notification for a clone event.
func NotifyClone(ctx context.Context, repoURL, targetPath string) {
	event := notify.NewEvent(notify.EventClone).
		WithRepository(repoURL).
		WithURL(repoURL).
		WithExtra("path", targetPath)

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyPull sends a notification for a pull event.
func NotifyPull(ctx context.Context, repoPath string) {
	event := notify.NewEvent(notify.EventPull)

	// Get repository info
	if repoURL, err := getRemoteURLFromPath(repoPath, "origin"); err == nil {
		event.WithRepository(repoURL)
	}

	// Get current branch
	client := git.NewClient()
	if branch, err := client.CurrentBranch(ctx); err == nil {
		event.WithBranch(branch)
	}

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyCommit sends a notification for a commit event.
func NotifyCommit(ctx context.Context, repoPath, sha, message string) {
	event := notify.NewEvent(notify.EventCommit).
		WithCommit(sha, message)

	// Get repository info
	if repoURL, err := getRemoteURLFromPath(repoPath, "origin"); err == nil {
		event.WithRepository(repoURL)
	}

	// Get author
	if author, err := getGitUser(); err == nil {
		event.WithAuthor(author)
	}

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyCIFail sends a notification for a CI failure event.
func NotifyCIFail(ctx context.Context, repo, workflowURL, errorMsg string) {
	event := notify.NewEvent(notify.EventCIFail).
		WithRepository(repo).
		WithURL(workflowURL).
		WithError(errorMsg)

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyCIPass sends a notification for a CI pass event.
func NotifyCIPass(ctx context.Context, repo, workflowURL string) {
	event := notify.NewEvent(notify.EventCIPass).
		WithRepository(repo).
		WithURL(workflowURL)

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// NotifyError sends a notification for an error event.
func NotifyError(ctx context.Context, repo, errorMsg string) {
	event := notify.NewEvent(notify.EventError).
		WithRepository(repo).
		WithError(errorMsg)

	// Get profile context
	if profile, workspace := getCurrentProfileContext(); profile != "" {
		event.WithProfile(profile).WithWorkspace(workspace)
	}

	notify.Send(ctx, event)
}

// getRemoteURLFromPath gets the remote URL for a repository.
func getRemoteURLFromPath(repoPath, remote string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", remote)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getCommitMessage gets the commit message for a ref.
func getCommitMessage(repoPath, ref string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%s", ref)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getGitUser gets the current git user.
func getGitUser() (string, error) {
	cmd := exec.Command("git", "config", "--get", "user.name")

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// getCurrentProfileContext gets the current profile and workspace names.
func getCurrentProfileContext() (profile, workspace string) {
	pm, err := NewProfileManager()
	if err != nil {
		return "", ""
	}

	p, err := pm.GetActiveProfile()
	if err != nil || p == nil {
		return "", ""
	}

	return p.Name, p.Workspace
}
