package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReauthorOptions contains options for rewriting git author history.
type ReauthorOptions struct {
	// OldEmail is the email address to replace
	OldEmail string
	// NewEmail is the new email address
	NewEmail string
	// NewName is the new author/committer name (optional, keeps existing if empty)
	NewName string
	// RepoPath is the path to the repository (uses current directory if empty)
	RepoPath string
	// AllRefs rewrites all branches and tags when true
	AllRefs bool
}

// ReauthorResult contains the result of a reauthor operation.
type ReauthorResult struct {
	// CommitsRewritten is the number of commits that were rewritten
	CommitsRewritten int
	// TagsRewritten is the list of tags that were rewritten
	TagsRewritten []string
	// BranchesRewritten is the list of branches that were rewritten
	BranchesRewritten []string
}

// Reauthor rewrites git history to change author/committer email and name.
// This uses git filter-branch to rewrite commits matching the old email.
func Reauthor(opts ReauthorOptions) (*ReauthorResult, error) {
	if opts.OldEmail == "" {
		return nil, fmt.Errorf("old email is required")
	}

	if opts.NewEmail == "" {
		return nil, fmt.Errorf("new email is required")
	}

	repoPath := opts.RepoPath
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
	}

	// Verify it's a git repository
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Build the filter-branch env-filter script
	script := buildEnvFilterScript(opts)

	// Build the git filter-branch command
	args := []string{
		"filter-branch",
		"-f",
		"--env-filter", script,
		"--tag-name-filter", "cat",
	}

	if opts.AllRefs {
		args = append(args, "--", "--branches", "--tags")
	} else {
		args = append(args, "--", "--all")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), "FILTER_BRANCH_SQUELCH_WARNING=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git filter-branch failed: %w\nOutput: %s", err, string(output))
	}

	result := parseFilterBranchOutput(string(output))
	return result, nil
}

// buildEnvFilterScript creates the shell script for git filter-branch --env-filter.
func buildEnvFilterScript(opts ReauthorOptions) string {
	var script strings.Builder

	script.WriteString(fmt.Sprintf(`OLD_EMAIL="%s"
CORRECT_EMAIL="%s"
`, opts.OldEmail, opts.NewEmail))

	if opts.NewName != "" {
		script.WriteString(fmt.Sprintf(`CORRECT_NAME="%s"
`, opts.NewName))
	}

	// Committer check
	script.WriteString(`
if [ "$GIT_COMMITTER_EMAIL" = "$OLD_EMAIL" ]
then
    export GIT_COMMITTER_EMAIL="$CORRECT_EMAIL"
`)

	if opts.NewName != "" {
		script.WriteString(`    export GIT_COMMITTER_NAME="$CORRECT_NAME"
`)
	}

	script.WriteString(`fi
`)

	// Author check
	script.WriteString(`
if [ "$GIT_AUTHOR_EMAIL" = "$OLD_EMAIL" ]
then
    export GIT_AUTHOR_EMAIL="$CORRECT_EMAIL"
`)

	if opts.NewName != "" {
		script.WriteString(`    export GIT_AUTHOR_NAME="$CORRECT_NAME"
`)
	}

	script.WriteString(`fi
`)

	return script.String()
}

// parseFilterBranchOutput parses the output of git filter-branch to extract statistics.
func parseFilterBranchOutput(output string) *ReauthorResult {
	result := &ReauthorResult{
		TagsRewritten:     []string{},
		BranchesRewritten: []string{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Count rewritten commits from lines like "Rewrite abc123 (1/10)"
		if strings.HasPrefix(line, "Rewrite ") {
			result.CommitsRewritten++
		}

		// Parse rewritten refs
		if strings.HasPrefix(line, "Ref 'refs/heads/") && strings.Contains(line, "was rewritten") {
			branch := extractRefName(line, "refs/heads/")
			if branch != "" {
				result.BranchesRewritten = append(result.BranchesRewritten, branch)
			}
		}

		if strings.HasPrefix(line, "Ref 'refs/tags/") && strings.Contains(line, "was rewritten") {
			tag := extractRefName(line, "refs/tags/")
			if tag != "" {
				result.TagsRewritten = append(result.TagsRewritten, tag)
			}
		}
	}

	return result
}

// extractRefName extracts the ref name from a git filter-branch output line.
func extractRefName(line, prefix string) string {
	// Line format: "Ref 'refs/heads/main' was rewritten"
	start := strings.Index(line, prefix)
	if start == -1 {
		return ""
	}

	start += len(prefix)
	end := strings.Index(line[start:], "'")
	if end == -1 {
		return ""
	}

	return line[start : start+end]
}

// ListAuthors returns a list of unique author emails in the repository.
func ListAuthors(repoPath string) ([]string, error) {
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
	}

	cmd := exec.Command("git", "log", "--all", "--format=%ae", "--")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list authors: %w", err)
	}

	// Deduplicate emails
	emailSet := make(map[string]struct{})
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, email := range lines {
		email = strings.TrimSpace(email)
		if email != "" {
			emailSet[email] = struct{}{}
		}
	}

	emails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		emails = append(emails, email)
	}

	return emails, nil
}

// CountCommitsByEmail counts commits by a specific email address.
func CountCommitsByEmail(repoPath, email string) (int, error) {
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
	}

	cmd := exec.Command("git", "log", "--all", "--author="+email, "--format=%H", "--")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to count commits: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}

	return len(lines), nil
}
