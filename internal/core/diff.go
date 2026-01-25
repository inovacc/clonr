package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiffOptions configures diff behavior.
type DiffOptions struct {
	Staged   bool // Show staged changes (--cached)
	Stat     bool // Show diffstat instead of full diff
	NameOnly bool // Show only file names
}

// DiffResult holds parsed diff information.
type DiffResult struct {
	RepoPath   string   `json:"repo_path"`
	RepoURL    string   `json:"repo_url,omitempty"`
	HasChanges bool     `json:"has_changes"`
	Files      []string `json:"files,omitempty"`
	Stats      string   `json:"stats,omitempty"`
	Diff       string   `json:"diff,omitempty"`
}

// GetDiff returns git diff for a repository.
func GetDiff(repoPath string, opts DiffOptions) (*DiffResult, error) {
	if err := validateGitRepo(repoPath); err != nil {
		return nil, err
	}

	result := &DiffResult{
		RepoPath: repoPath,
	}

	// Get changed files first to determine if there are changes
	files, err := GetDiffFiles(repoPath, opts.Staged)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	result.Files = files
	result.HasChanges = len(files) > 0

	if !result.HasChanges {
		return result, nil
	}

	// Build git diff command based on options
	args := []string{"-C", repoPath, "diff"}

	if opts.Staged {
		args = append(args, "--cached")
	}

	if opts.NameOnly {
		// Already have files from GetDiffFiles, no need for full diff
		return result, nil
	}

	if opts.Stat {
		args = append(args, "--stat")

		output, err := runGitCommand(args...)
		if err != nil {
			return nil, fmt.Errorf("failed to get diff stat: %w", err)
		}

		result.Stats = strings.TrimSpace(output)

		return result, nil
	}

	// Get full diff
	output, err := runGitCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	result.Diff = output

	return result, nil
}

// GetDiffFiles returns list of changed files.
func GetDiffFiles(repoPath string, staged bool) ([]string, error) {
	args := []string{"-C", repoPath, "diff", "--name-only"}

	if staged {
		args = append(args, "--cached")
	}

	output, err := runGitCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}, nil
	}

	files := strings.Split(output, "\n")

	return files, nil
}

// GetDiffSummary returns a quick summary of changes.
func GetDiffSummary(repoPath string, staged bool) (string, error) {
	args := []string{"-C", repoPath, "diff", "--shortstat"}

	if staged {
		args = append(args, "--cached")
	}

	output, err := runGitCommand(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get diff summary: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// validateGitRepo checks if the path is a valid git repository.
func validateGitRepo(repoPath string) error {
	args := []string{"-C", repoPath, "rev-parse", "--git-dir"}

	_, err := runGitCommand(args...)
	if err != nil {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	return nil
}

// runGitCommand executes a git command and returns the output.
func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w - %s", err, string(output))
	}

	return string(output), nil
}
