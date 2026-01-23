package core

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type CoreSection struct {
	RepositoryFormatVersion int  `ini:"repositoryformatversion"`
	FileMode                bool `ini:"filemode"`
	Bare                    bool `ini:"bare"`
}

type RemoteSection struct {
	URL   string `ini:"url"`
	Fetch string `ini:"fetch"`
}

type BranchSection struct {
	Remote string `ini:"remote"`
	Merge  string `ini:"merge"`
}

type GitConfig struct {
	Core   CoreSection              `ini:"core"`
	Remote map[string]RemoteSection `ini:"remote"`
	Branch map[string]BranchSection `ini:"branch"`
}

func newGitConfig(path string) (*GitConfig, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	gitConfig := GitConfig{
		Remote: make(map[string]RemoteSection),
		Branch: make(map[string]BranchSection),
	}

	if err := cfg.Section("core").MapTo(&gitConfig.Core); err != nil {
		return nil, err
	}

	for _, sec := range cfg.Sections() {
		if sec.HasKey("url") && sec.HasKey("fetch") {
			name := sec.Name()[len(`remote "`) : len(sec.Name())-1] // extraer "origin"

			var remote RemoteSection

			if err := sec.MapTo(&remote); err != nil {
				return nil, err
			}

			gitConfig.Remote[name] = remote
		}
	}

	for _, sec := range cfg.Sections() {
		if sec.HasKey("remote") && sec.HasKey("merge") {
			name := sec.Name()[len(`branch "`) : len(sec.Name())-1] // extraer "main"

			var branch BranchSection

			if err := sec.MapTo(&branch); err != nil {
				return nil, err
			}

			gitConfig.Branch[name] = branch
		}
	}

	return &gitConfig, nil
}

type DotGit struct {
	*GitConfig

	Path string
	URL  *url.URL
}

func dotGitCheck(path string) (*DotGit, error) {
	base := filepath.Base(path)
	if base != ".git" {
		return nil, fmt.Errorf("not a git repo: %s", path)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(abs, "config")

	cfg, err := newGitConfig(configFile)
	if err != nil {
		return nil, err
	}

	u, err := gitHubURL(cfg.Remote["origin"].URL)
	if err != nil {
		return nil, err
	}

	return &DotGit{
		Path:      abs,
		URL:       u,
		GitConfig: cfg,
	}, nil
}

func gitHubURL(repoPath string) (*url.URL, error) {
	s := strings.TrimSpace(repoPath)
	if s == "" {
		// create a local repo
		return &url.URL{
			Scheme: "file",
			Path:   ".",
		}, nil
	}

	// Detect SCP-like syntax: user@host:path
	// e.g., git@github.com:owner/repo.git
	if isSCPLike(s) {
		userHost, repo, _ := strings.Cut(s, ":")

		user, host, ok := strings.Cut(userHost, "@")

		if !ok || user == "" || host == "" || repo == "" {
			return nil, fmt.Errorf("invalid SSH repository URL: %s", repoPath)
		}

		// url.URL Path should start with a leading slash.
		if !strings.HasPrefix(repo, "/") {
			repo = "/" + repo
		}

		return &url.URL{
			Scheme: "ssh",
			User:   url.User(user),
			Host:   host,
			Path:   repo,
		}, nil
	}

	// Handle standard URL forms.
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %s", repoPath)
	}

	if u.Scheme == "" {
		return nil, fmt.Errorf("unsupported repository URL (missing scheme): %s", repoPath)
	}

	// Normalize schemes we accept.
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		// ok
	case "ssh", "git+ssh":
		// normalize git+ssh -> ssh
		u.Scheme = "ssh"
		if u.Host == "" {
			return nil, fmt.Errorf("invalid SSH repository URL (missing host): %s", repoPath)
		}

		if u.Path == "" || u.Path == "/" {
			return nil, fmt.Errorf("invalid SSH repository URL (missing path): %s", repoPath)
		}

		// Ensure leading slash for consistency
		if !strings.HasPrefix(u.Path, "/") {
			u.Path = "/" + u.Path
		}
	case "git":
		// git:// protocol (read-only; rare nowadays)
		if u.Host == "" {
			return nil, fmt.Errorf("invalid git:// repository URL (missing host): %s", repoPath)
		}
	default:
		return nil, fmt.Errorf("unsupported repository URL scheme %q: %s", u.Scheme, repoPath)
	}

	return u, nil
}

func isSCPLike(s string) bool {
	// Must contain a single '@' before the first ':' and not start with a scheme.
	if i := strings.IndexByte(s, ':'); i > 0 {
		if strings.IndexByte(s[:i], '@') > 0 && !hasSchemePrefix(s) {
			return true
		}
	}

	return false
}

func hasSchemePrefix(s string) bool {
	l := strings.ToLower(s)

	return strings.HasPrefix(l, "http://") ||
		strings.HasPrefix(l, "https://") ||
		strings.HasPrefix(l, "ssh://") ||
		strings.HasPrefix(l, "git://") ||
		strings.HasPrefix(l, "git+ssh://")
}

// DetectRepository detects the GitHub owner/repo from various sources.
// Priority:
//  1. Explicit argument (owner/repo format)
//  2. --repo flag value
//  3. Current directory's git config (remote origin)
//
// Returns owner, repo, and any error encountered.
func DetectRepository(arg, repoFlag string) (owner, repo string, err error) {
	// 1. Check explicit argument first
	if arg != "" {
		return parseOwnerRepo(arg)
	}

	// 2. Check --repo flag
	if repoFlag != "" {
		return parseOwnerRepo(repoFlag)
	}

	// 3. Try to detect from current directory
	return detectFromCurrentDir()
}

// parseOwnerRepo parses an "owner/repo" string or a full GitHub URL
func parseOwnerRepo(s string) (owner, repo string, err error) {
	s = strings.TrimSpace(s)

	// Check if it's a full URL
	if strings.Contains(s, "github.com") {
		return parseGitHubURL(s)
	}

	// Simple owner/repo format
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %q (expected owner/repo)", s)
	}

	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid repository format: %q (owner and repo cannot be empty)", s)
	}

	return owner, repo, nil
}

// detectFromCurrentDir attempts to detect owner/repo from the current directory's git config
func detectFromCurrentDir() (owner, repo string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	gitDir := filepath.Join(cwd, ".git")

	info, err := os.Stat(gitDir)
	if err != nil {
		return "", "", fmt.Errorf("not a git repository (no .git directory found)")
	}

	if !info.IsDir() {
		return "", "", fmt.Errorf("not a git repository (.git is not a directory)")
	}

	configFile := filepath.Join(gitDir, "config")
	cfg, err := newGitConfig(configFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read git config: %w", err)
	}

	origin, ok := cfg.Remote["origin"]
	if !ok || origin.URL == "" {
		return "", "", fmt.Errorf("no origin remote found in git config")
	}

	// Parse the origin URL to extract owner/repo
	owner, repo, err = parseGitHubURL(origin.URL)
	if err != nil {
		return "", "", fmt.Errorf("origin is not a GitHub repository: %w", err)
	}

	return owner, repo, nil
}

// GetRepoFullName returns the full "owner/repo" name
func GetRepoFullName(owner, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}
