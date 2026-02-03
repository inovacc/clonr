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
)

// Client wraps git operations with authentication support
type Client struct {
	ClonrPath string // Path to clonr executable (for credential helper)
	RepoDir   string // Repository directory
	GitPath   string // Path to git executable
	Stderr    io.Writer
	Stdin     io.Reader
	Stdout    io.Writer
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

	preArgs := make([]string, 0, 4+len(args))

	if pattern.allMatching {
		// Clear existing credential helpers and set ours for all hosts
		preArgs = append(preArgs,
			"-c", "credential.helper=",
			"-c", fmt.Sprintf("credential.helper=%s", credHelper),
		)
	} else {
		// Set credential helper for specific host only
		preArgs = append(preArgs,
			"-c", fmt.Sprintf("credential.%s.helper=", pattern.pattern),
			"-c", fmt.Sprintf("credential.%s.helper=%s", pattern.pattern, credHelper),
		)
	}

	preArgs = append(preArgs, args...)
	cmd := exec.CommandContext(ctx, c.GitPath, preArgs...)

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
	cmd.Stdin = c.Stdin
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
	cmd.Stdin = c.Stdin
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

// StashOptions configures stash behavior
type StashOptions struct {
	Message          string
	IncludeUntracked bool
	KeepIndex        bool
}

// CheckoutOptions configures checkout behavior
type CheckoutOptions struct {
	Create bool // Create new branch
	Force  bool // Force checkout
}

// MergeOptions configures merge behavior
type MergeOptions struct {
	NoFastForward bool
	Squash        bool
	Message       string
}

// Stash stashes changes
func (c *Client) Stash(ctx context.Context, opts StashOptions) error {
	args := []string{"stash", "push"}

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}

	if opts.KeepIndex {
		args = append(args, "--keep-index")
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Stderr: string(output), err: err}
	}

	return nil
}

// StashPop pops the latest stash
func (c *Client) StashPop(ctx context.Context) error {
	cmd := c.Command(ctx, "stash", "pop")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Stderr: string(output), err: err}
	}

	return nil
}

// StashList lists all stashes
func (c *Client) StashList(ctx context.Context) (string, error) {
	cmd := c.Command(ctx, "stash", "list")

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{err: err}
	}

	return string(output), nil
}

// StashDrop drops a stash
func (c *Client) StashDrop(ctx context.Context, stash string) error {
	args := []string{"stash", "drop"}
	if stash != "" {
		args = append(args, stash)
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Stderr: string(output), err: err}
	}

	return nil
}

// Checkout checks out a branch or commit
func (c *Client) Checkout(ctx context.Context, target string, opts CheckoutOptions) error {
	args := []string{"checkout"}

	if opts.Create {
		args = append(args, "-b")
	}

	if opts.Force {
		args = append(args, "-f")
	}

	args = append(args, target)

	cmd := c.Command(ctx, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if err := cmd.Run(); err != nil {
		return &GitError{err: err}
	}

	return nil
}

// Merge merges a branch
func (c *Client) Merge(ctx context.Context, branch string, opts MergeOptions) error {
	args := []string{"merge"}

	if opts.NoFastForward {
		args = append(args, "--no-ff")
	}

	if opts.Squash {
		args = append(args, "--squash")
	}

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	args = append(args, branch)

	cmd := c.Command(ctx, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr

	if err := cmd.Run(); err != nil {
		return &GitError{err: err}
	}

	return nil
}

// ListBranches lists branches
func (c *Client) ListBranches(ctx context.Context, all bool) ([]string, error) {
	args := []string{"branch", "--format=%(refname:short)"}
	if all {
		args = append(args, "-a")
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, &GitError{err: err}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// GitError represents a git command error
type GitError struct {
	ExitCode int
	Stderr   string
	Args     []string
	err      error
}

func (e *GitError) Error() string {
	if e.Stderr == "" {
		if len(e.Args) > 0 {
			return fmt.Sprintf("git %s: %v", strings.Join(e.Args, " "), e.err)
		}

		return fmt.Errorf("git command failed: %w", e.err).Error()
	}

	if len(e.Args) > 0 {
		return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), strings.TrimSpace(e.Stderr))
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

	for line := range strings.SplitSeq(buf.String(), "\n") {
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

// CommandModifier is a function that can modify a git command before execution
type CommandModifier func(*exec.Cmd)

// GitCommand wraps exec.Cmd with additional functionality
type GitCommand struct {
	*exec.Cmd

	args      []string
	modifiers []CommandModifier
}

// NewGitCommand creates a new GitCommand
func (c *Client) NewGitCommand(ctx context.Context, args ...string) *GitCommand {
	cmd := c.Command(ctx, args...)

	return &GitCommand{
		Cmd:  cmd,
		args: args,
	}
}

// WithEnv adds environment variables to the command
func (gc *GitCommand) WithEnv(env ...string) *GitCommand {
	gc.modifiers = append(gc.modifiers, func(cmd *exec.Cmd) {
		cmd.Env = append(cmd.Environ(), env...)
	})

	return gc
}

// WithDir sets the working directory for the command
func (gc *GitCommand) WithDir(dir string) *GitCommand {
	gc.modifiers = append(gc.modifiers, func(cmd *exec.Cmd) {
		cmd.Dir = dir
	})

	return gc
}

// WithStdio attaches stdin, stdout, stderr to the command
func (gc *GitCommand) WithStdio(stdin io.Reader, stdout, stderr io.Writer) *GitCommand {
	gc.modifiers = append(gc.modifiers, func(cmd *exec.Cmd) {
		cmd.Stdin = stdin
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	})

	return gc
}

// Apply applies all modifiers to the command
func (gc *GitCommand) Apply() *exec.Cmd {
	for _, mod := range gc.modifiers {
		mod(gc.Cmd)
	}

	return gc.Cmd
}

// Run applies modifiers and runs the command
func (gc *GitCommand) Run() error {
	gc.Apply()

	if err := gc.Cmd.Run(); err != nil {
		return &GitError{Args: gc.args, err: err}
	}

	return nil
}

// Output applies modifiers and returns the output
func (gc *GitCommand) Output() ([]byte, error) {
	gc.Apply()

	output, err := gc.Cmd.Output()
	if err != nil {
		return output, &GitError{Args: gc.args, err: err}
	}

	return output, nil
}

// CombinedOutput applies modifiers and returns combined output
func (gc *GitCommand) CombinedOutput() ([]byte, error) {
	gc.Apply()

	output, err := gc.Cmd.CombinedOutput()
	if err != nil {
		return output, &GitError{Args: gc.args, Stderr: string(output), err: err}
	}

	return output, nil
}

// LogOptions configures log output
type LogOptions struct {
	Limit   int
	Oneline bool
	All     bool
	Format  string // Custom format string
	Since   string // Show commits more recent than this date
	Until   string // Show commits older than this date
	Author  string // Filter by author
	Grep    string // Filter by commit message
}

// Commit represents a git commit
type Commit struct {
	SHA       string
	ShortSHA  string
	Author    string
	Email     string
	Date      string
	Subject   string
	Body      string
	Refs      string
}

// Log returns the commit log
func (c *Client) Log(ctx context.Context, opts LogOptions) ([]Commit, error) {
	// Use a parseable format: SHA|ShortSHA|Author|Email|Date|Subject
	format := "%H|%h|%an|%ae|%ci|%s"
	args := []string{"log", "--format=" + format}

	if opts.Limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", opts.Limit))
	}

	if opts.All {
		args = append(args, "--all")
	}

	if opts.Since != "" {
		args = append(args, "--since="+opts.Since)
	}

	if opts.Until != "" {
		args = append(args, "--until="+opts.Until)
	}

	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}

	if opts.Grep != "" {
		args = append(args, "--grep="+opts.Grep)
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, &GitError{Args: args, err: err}
	}

	commits := make([]Commit, 0)

	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 6 {
			commits = append(commits, Commit{
				SHA:      parts[0],
				ShortSHA: parts[1],
				Author:   parts[2],
				Email:    parts[3],
				Date:     parts[4],
				Subject:  parts[5],
			})
		}
	}

	return commits, nil
}

// LogOneline returns a simple one-line log output
func (c *Client) LogOneline(ctx context.Context, limit int) (string, error) {
	args := []string{"log", "--oneline"}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{Args: args, err: err}
	}

	return strings.TrimSpace(string(output)), nil
}

// DiffOptions configures diff output
type DiffOptions struct {
	Staged     bool   // Show staged changes (--cached)
	Stat       bool   // Show diffstat (--stat)
	NameOnly   bool   // Only show file names
	NameStatus bool   // Show file names and status
	Commit     string // Diff against a specific commit
	Path       string // Limit to a specific path
}

// Diff returns the diff output
func (c *Client) Diff(ctx context.Context, opts DiffOptions) (string, error) {
	args := []string{"diff"}

	if opts.Staged {
		args = append(args, "--cached")
	}

	if opts.Stat {
		args = append(args, "--stat")
	}

	if opts.NameOnly {
		args = append(args, "--name-only")
	}

	if opts.NameStatus {
		args = append(args, "--name-status")
	}

	if opts.Commit != "" {
		args = append(args, opts.Commit)
	}

	if opts.Path != "" {
		args = append(args, "--", opts.Path)
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{Args: args, err: err}
	}

	return string(output), nil
}

// BranchInfo contains detailed branch information
type BranchInfo struct {
	Name      string
	Current   bool
	Upstream  string
	LastCommit string
	Gone      bool // Upstream is gone
}

// ListBranchesDetailed lists branches with detailed information
func (c *Client) ListBranchesDetailed(ctx context.Context, all bool) ([]BranchInfo, error) {
	// Format: refname:short|HEAD|upstream:short|objectname:short|upstream:track
	format := "%(refname:short)|%(HEAD)|%(upstream:short)|%(objectname:short)|%(upstream:track)"

	args := []string{"branch", "--format=" + format}
	if all {
		args = append(args, "-a")
	}

	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, &GitError{Args: args, err: err}
	}

	branches := make([]BranchInfo, 0)

	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 5 {
			info := BranchInfo{
				Name:       parts[0],
				Current:    parts[1] == "*",
				Upstream:   parts[2],
				LastCommit: parts[3],
				Gone:       strings.Contains(parts[4], "gone"),
			}
			branches = append(branches, info)
		}
	}

	return branches, nil
}

// DeleteBranch deletes a branch
func (c *Client) DeleteBranch(ctx context.Context, name string, force bool) error {
	args := []string{"branch"}

	if force {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}

	args = append(args, name)
	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Args: args, Stderr: string(output), err: err}
	}

	return nil
}

// Remote represents a git remote
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// ListRemotes lists all configured remotes
func (c *Client) ListRemotes(ctx context.Context) ([]Remote, error) {
	cmd := c.Command(ctx, "remote", "-v")

	output, err := cmd.Output()
	if err != nil {
		return nil, &GitError{Args: []string{"remote", "-v"}, err: err}
	}

	remoteMap := make(map[string]*Remote)

	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}

		// Format: name<TAB>url (fetch|push)
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			name := parts[0]
			url := parts[1]
			kind := strings.Trim(parts[2], "()")

			if _, ok := remoteMap[name]; !ok {
				remoteMap[name] = &Remote{Name: name}
			}

			switch kind {
			case "fetch":
				remoteMap[name].FetchURL = url
			case "push":
				remoteMap[name].PushURL = url
			}
		}
	}

	remotes := make([]Remote, 0, len(remoteMap))
	for _, r := range remoteMap {
		remotes = append(remotes, *r)
	}

	return remotes, nil
}

// AddRemote adds a new remote
func (c *Client) AddRemote(ctx context.Context, name, url string) error {
	args := []string{"remote", "add", name, url}
	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Args: args, Stderr: string(output), err: err}
	}

	return nil
}

// RemoveRemote removes a remote
func (c *Client) RemoveRemote(ctx context.Context, name string) error {
	args := []string{"remote", "remove", name}
	cmd := c.Command(ctx, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &GitError{Args: args, Stderr: string(output), err: err}
	}

	return nil
}

// StatusInfo contains detailed status information
type StatusInfo struct {
	Branch        string
	Upstream      string
	Ahead         int
	Behind        int
	Staged        []FileStatus
	Unstaged      []FileStatus
	Untracked     []string
	HasChanges    bool
	HasUntracked  bool
	HasConflicts  bool
}

// FileStatus represents the status of a single file
type FileStatus struct {
	Path   string
	Status string // M=modified, A=added, D=deleted, R=renamed, C=copied, U=updated
}

// StatusPorcelain returns machine-readable status
func (c *Client) StatusPorcelain(ctx context.Context) (string, error) {
	args := []string{"status", "--porcelain", "-b"}
	cmd := c.Command(ctx, args...)

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{Args: args, err: err}
	}

	return string(output), nil
}

// GetHead returns the current HEAD reference
func (c *Client) GetHead(ctx context.Context) (string, error) {
	cmd := c.Command(ctx, "rev-parse", "HEAD")

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{Args: []string{"rev-parse", "HEAD"}, err: err}
	}

	return strings.TrimSpace(string(output)), nil
}

// GetShortHead returns the short form of current HEAD
func (c *Client) GetShortHead(ctx context.Context) (string, error) {
	cmd := c.Command(ctx, "rev-parse", "--short", "HEAD")

	output, err := cmd.Output()
	if err != nil {
		return "", &GitError{Args: []string{"rev-parse", "--short", "HEAD"}, err: err}
	}

	return strings.TrimSpace(string(output)), nil
}
