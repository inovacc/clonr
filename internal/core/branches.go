package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// Branch represents a git branch
type Branch struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	IsRemote  bool   `json:"is_remote"`
}

// BranchListOptions configures branch listing
type BranchListOptions struct {
	IncludeRemote bool // Include remote branches
	All           bool // Show all branches (local + remote)
}

// ListBranches returns all branches for a repository at the given path
func ListBranches(repoPath string, opts BranchListOptions) ([]Branch, error) {
	args := []string{"-C", repoPath, "branch", "--no-color"}

	if opts.All {
		args = append(args, "-a")
	} else if opts.IncludeRemote {
		args = append(args, "-r")
	}

	cmd := exec.Command("git", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w - %s", err, string(output))
	}

	return parseBranches(string(output)), nil
}

// GetCurrentBranch returns the current branch name for a repository
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w - %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// CheckoutBranch switches to the specified branch
func CheckoutBranch(repoPath, branchName string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", branchName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w - %s", err, string(output))
	}

	return nil
}

// CreateBranch creates a new branch and optionally switches to it
func CreateBranch(repoPath, branchName string, checkout bool) error {
	if checkout {
		cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", branchName)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch: %w - %s", err, string(output))
		}

		return nil
	}

	cmd := exec.Command("git", "-C", repoPath, "branch", branchName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create branch: %w - %s", err, string(output))
	}

	return nil
}

// DeleteBranch deletes a branch
func DeleteBranch(repoPath, branchName string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}

	cmd := exec.Command("git", "-C", repoPath, "branch", flag, branchName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w - %s", err, string(output))
	}

	return nil
}

// parseBranches parses git branch output into Branch structs
func parseBranches(output string) []Branch {
	var branches []Branch

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		branch := Branch{}

		// Check if current branch (marked with *)
		if strings.HasPrefix(line, "* ") {
			branch.IsCurrent = true
			line = strings.TrimPrefix(line, "* ")
		}

		// Check if remote branch
		if strings.HasPrefix(line, "remotes/") {
			branch.IsRemote = true
			line = strings.TrimPrefix(line, "remotes/")
		}

		// Handle detached HEAD state
		if strings.Contains(line, "HEAD detached") || strings.Contains(line, "(HEAD detached") {
			branch.Name = "(detached HEAD)"
			branches = append(branches, branch)

			continue
		}

		// Skip symbolic references like origin/HEAD -> origin/main
		if strings.Contains(line, " -> ") {
			continue
		}

		branch.Name = line
		branches = append(branches, branch)
	}

	return branches
}
