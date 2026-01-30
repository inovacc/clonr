// Package git provides a git client with credential helper support.
// Pattern inspired by github.com/cli/cli
package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Client wraps git operations with authentication support
type Client struct {
	ClonrPath string    // Path to clonr executable (for credential helper)
	RepoDir   string    // Repository directory
	GitPath   string    // Path to git executable
	Stderr    io.Writer
	Stdin     io.Reader
	Stdout    io.Writer
	mu        sync.Mutex
}

// NewClient creates a new git client
func NewClient() *Client {
	gitPath, _ := exec.LookPath("git")
	clonrPath, _ := os.Executable()

	return &Client{
		ClonrPath: clonrPath,
		GitPath:   gitPath,
		Stderr:    os.Stderr,
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
	}
}

// NewClientForRepo creates a client for a specific repository
func NewClientForRepo(repoDir string) *Client {
	c := NewClient()
	c.RepoDir = repoDir
	return c
}

// CredentialPattern represents a pattern for credential matching
type CredentialPattern struct {
	allMatching bool
	pattern     string
}

// AllMatchingCredentialsPattern matches all hosts
var AllMatchingCredentialsPattern = CredentialPattern{allMatching: true}

// CredentialPatternFromHost creates a pattern for a specific host
func CredentialPatternFromHost(host string) CredentialPattern {
	return CredentialPattern{
		pattern: fmt.Sprintf("https://%s", host),
	}
}

// CredentialPatternFromGitURL derives a credential pattern from a git URL
func CredentialPatternFromGitURL(gitURL string) (CredentialPattern, error) {
	u, err := ParseURL(gitURL)
	if err != nil {
		return CredentialPattern{}, err
	}

	return CredentialPatternFromHost(u.Host), nil
}

// Command creates a git command without authentication
// Note: Do not set Stdout/Stderr if you plan to use CombinedOutput()
func (c *Client) Command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.GitPath, args...)

	if c.RepoDir != "" {
		cmd.Dir = c.RepoDir
	}

	return cmd
}

// CommandInteractive creates a git command with stdio attached for interactive use
func (c *Client) CommandInteractive(ctx context.Context, args ...string) *exec.Cmd {
	cmd := c.Command(ctx, args...)
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	return cmd
}

// AuthenticatedCommand creates a git command with credential helper configured
// Note: Do not set Stdout/Stderr if you plan to use CombinedOutput()
func (c *Client) AuthenticatedCommand(ctx context.Context, pattern CredentialPattern, args ...string) *exec.Cmd {
	credHelper := fmt.Sprintf("!%q auth git-credential", c.ClonrPath)

	var preArgs []string

	if pattern.allMatching {
		// Clear existing credential helpers and set ours for all hosts
		preArgs = []string{
			"-c", "credential.helper=",
			"-c", fmt.Sprintf("credential.helper=%s", credHelper),
		}
	} else {
		// Set credential helper for specific host only
		preArgs = []string{
			"-c", fmt.Sprintf("credential.%s.helper=", pattern.pattern),
			"-c", fmt.Sprintf("credential.%s.helper=%s", pattern.pattern, credHelper),
		}
	}

	allArgs := append(preArgs, args...)
	cmd := exec.CommandContext(ctx, c.GitPath, allArgs...)

	if c.RepoDir != "" {
		cmd.Dir = c.RepoDir
	}

	return cmd
}

// Clone clones a repository with authentication
func (c *Client) Clone(ctx context.Context, cloneURL, targetPath string) error {
	pattern, err := CredentialPatternFromGitURL(cloneURL)
	if err != nil {
		// Fallback to all-matching pattern
		pattern = AllMatchingCredentialsPattern
	}

	cmd := c.AuthenticatedCommand(ctx, pattern, "clone", cloneURL, targetPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{
			Stderr: string(output),
			err:    err,
		}
	}

	return nil
}

// Pull pulls changes with authentication
func (c *Client) Pull(ctx context.Context, remote, branch string) error {
	args := []string{"pull"}
	if remote != "" {
		args = append(args, remote)
		if branch != "" {
			args = append(args, branch)
		}
	}

	cmd := c.AuthenticatedCommand(ctx, AllMatchingCredentialsPattern, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if err := cmd.Run(); err != nil {
		return &GitError{err: err}
	}

	return nil
}

// Push pushes changes with authentication
func (c *Client) Push(ctx context.Context, remote, branch string, opts PushOptions) error {
	args := []string{"push"}

	if opts.SetUpstream {
		args = append(args, "-u")
	}

	if opts.Force {
		args = append(args, "--force")
	}

	if opts.Tags {
		args = append(args, "--tags")
	}

	if remote != "" {
		args = append(args, remote)
		if branch != "" {
			args = append(args, branch)
		}
	}

	cmd := c.AuthenticatedCommand(ctx, AllMatchingCredentialsPattern, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if err := cmd.Run(); err != nil {
		return &GitError{err: err}
	}

	return nil
}

// Fetch fetches changes with authentication
func (c *Client) Fetch(ctx context.Context, remote, refspec string) error {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
		if refspec != "" {
			args = append(args, refspec)
		}
	}

	cmd := c.AuthenticatedCommand(ctx, AllMatchingCredentialsPattern, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{
			Stderr: string(output),
			err:    err,
		}
	}

	return nil
}

// Commit creates a commit
func (c *Client) Commit(ctx context.Context, message string, opts CommitOptions) error {
	if opts.All {
		// Stage all changes first
		addCmd := exec.CommandContext(ctx, c.GitPath, "add", "-A")
		if c.RepoDir != "" {
			addCmd.Dir = c.RepoDir
		}
		if _, err := addCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
	}

	args := []string{"commit", "-m", message}
	cmd := exec.CommandContext(ctx, c.GitPath, args...)
	if c.RepoDir != "" {
		cmd.Dir = c.RepoDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{
			Stderr: string(output),
			err:    err,
		}
	}

	return nil
}

// Tag creates a git tag
func (c *Client) Tag(ctx context.Context, name, message string) error {
	var args []string

	if message != "" {
		args = []string{"tag", "-a", name, "-m", message}
	} else {
		args = []string{"tag", name}
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{
			Stderr: string(output),
			err:    err,
		}
	}

	return nil
}

// GetRemoteURL returns the URL for a remote
func (c *Client) GetRemoteURL(ctx context.Context, remote string) (string, error) {
	cmd := c.Command(ctx, "remote", "get-url", remote)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// IsRepository checks if the current directory is a git repository
func (c *Client) IsRepository(ctx context.Context) bool {
	cmd := c.Command(ctx, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// Status returns the git status
func (c *Client) Status(ctx context.Context) (string, error) {
	cmd := c.Command(ctx, "status", "--short")
	cmd.Stdout = nil // Capture output

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// CurrentBranch returns the current branch name
func (c *Client) CurrentBranch(ctx context.Context) (string, error) {
	cmd := c.Command(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Stdout = nil

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// PushOptions configures push behavior
type PushOptions struct {
	SetUpstream bool
	Force       bool
	Tags        bool
}

// CommitOptions configures commit behavior
type CommitOptions struct {
	All bool // Stage all modified files before committing
}

// GitError represents a git command error
type GitError struct {
	ExitCode int
	Stderr   string
	err      error
}

func (e *GitError) Error() string {
	if e.Stderr == "" {
		return fmt.Errorf("git command failed: %w", e.err).Error()
	}
	return fmt.Sprintf("git command failed: %s", strings.TrimSpace(e.Stderr))
}

func (e *GitError) Unwrap() error {
	return e.err
}

// ParseURL parses a git URL, including SCP-like syntax
func ParseURL(rawURL string) (*url.URL, error) {
	// Handle SCP-like syntax: git@github.com:owner/repo.git
	if !strings.Contains(rawURL, "://") && strings.Contains(rawURL, ":") {
		if strings.HasPrefix(rawURL, "git@") {
			// Convert to ssh://git@host/path
			rawURL = "ssh://" + strings.Replace(rawURL, ":", "/", 1)
		}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Normalize schemes
	switch u.Scheme {
	case "git+https":
		u.Scheme = "https"
	case "git+ssh":
		u.Scheme = "ssh"
	}

	// Extract path without .git suffix for consistency
	u.Path = strings.TrimSuffix(u.Path, ".git")

	return u, nil
}

// ExtractRepoName extracts the repository name from a URL
func ExtractRepoName(rawURL string) string {
	u, err := ParseURL(rawURL)
	if err != nil {
		// Fallback to basic extraction
		parts := strings.Split(rawURL, "/")
		if len(parts) > 0 {
			return strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
		return ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// RepoRoot finds the root directory of the current git repository
func RepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}

	return strings.TrimSpace(string(output)), nil
}

// CredentialStore provides credential storage interface
type CredentialStore interface {
	Get(host string) (username, password string, err error)
}

// WriteCredential writes credentials in git credential helper format
func WriteCredential(w io.Writer, host, username, password string) error {
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "protocol=https\n")
	_, _ = fmt.Fprintf(&buf, "host=%s\n", host)
	_, _ = fmt.Fprintf(&buf, "username=%s\n", username)
	_, _ = fmt.Fprintf(&buf, "password=%s\n", password)

	_, err := w.Write(buf.Bytes())
	return err
}

// ReadCredentialRequest reads a credential request from git
func ReadCredentialRequest(r io.Reader) (map[string]string, error) {
	fields := make(map[string]string)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r)

	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			fields[parts[0]] = parts[1]
		}
	}

	return fields, nil
}

// FindGitDir finds the .git directory from a path
func FindGitDir(startPath string) (string, error) {
	current := startPath

	for {
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a git repository (or any parent)")
		}
		current = parent
	}
}
