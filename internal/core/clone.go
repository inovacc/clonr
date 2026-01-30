package core

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inovacc/clonr/internal/giturl"
	"github.com/inovacc/clonr/internal/grpcclient"
)

// CloneOptions configures the clone operation
type CloneOptions struct {
	Force    bool     // Force clone even if repo exists (removes existing)
	GitArgs  []string // Additional git clone arguments
	Protocol string   // Preferred protocol (https or ssh), empty for auto-detect
}

// CloneResult contains the result of a clone operation
type CloneResult struct {
	Repository *giturl.Repository
	CloneURL   string
	TargetPath string
	GitArgs    []string
}

// PrepareClone parses clone arguments and prepares for cloning.
// Supports multiple input formats like gh repo clone:
//   - "repo" (uses current GitHub user)
//   - "owner/repo"
//   - "https://github.com/owner/repo"
//   - "https://github.com/owner/repo/blob/main/file.go#L10" (strips extra path)
//   - "git@github.com:owner/repo.git"
//
// Arguments format: <repository> [<directory>] [-- <gitflags>...]
// Note: Cobra strips the "--" separator, so args after repo that start with "-"
// are treated as git flags.
func PrepareClone(args []string, opts CloneOptions) (*CloneResult, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("repository argument required")
	}

	// Parse arguments: <repository> [<directory>] [<gitflags>...]
	// After Cobra processing, "--" is stripped so we detect git flags by "-" prefix
	repoArg := args[0]
	remaining := args[1:]

	var (
		gitArgs   []string
		targetDir string
	)

	// Separate directory from git flags
	// If an arg starts with "-", it and all following args are git flags
	for i, arg := range remaining {
		if strings.HasPrefix(arg, "-") {
			gitArgs = remaining[i:]
			break
		}

		if i == 0 {
			targetDir = arg
		}
	}

	// Merge with options git args
	gitArgs = append(opts.GitArgs, gitArgs...)

	// Get the current GitHub user for shorthand resolution
	currentUser := getGitHubUsername()

	// Determine protocol
	protocol := opts.Protocol
	if protocol == "" {
		protocol = "https"
	}

	// If it's a URL, detect protocol from URL
	if giturl.IsURL(repoArg) {
		u, err := giturl.Parse(repoArg)
		if err == nil && u.Scheme == "ssh" {
			protocol = "ssh"
		}
	}

	// Parse the repository
	repo, err := giturl.ParseRepository(repoArg, currentUser)
	if err != nil {
		return nil, err
	}

	cloneURL := repo.CloneURL(protocol)

	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// Build canonical URL for database operations
	canonicalURL, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", repo.Host, repo.Owner, repo.Name))
	if err != nil {
		return nil, fmt.Errorf("error building canonical URL: %w", err)
	}

	// Check for repo existence in a database
	ok, err := client.RepoExistsByURL(canonicalURL)
	if err != nil {
		return nil, fmt.Errorf("error checking for repo existence: %w", err)
	}

	if ok {
		if !opts.Force {
			return nil, fmt.Errorf("repository already exists: %s\n\nUse --force to remove and re-clone", repo.FullName())
		}

		// Force mode: remove existing repo from database
		if err := client.RemoveRepoByURL(canonicalURL); err != nil {
			return nil, fmt.Errorf("error removing existing repo from database: %w", err)
		}

		log.Printf("Removed existing repo from database: %s\n", repo.FullName())
	}

	// Get config to determine default clone directory
	cfg, err := client.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting config: %w", err)
	}

	// Determine a target path
	var savePath string

	switch {
	case targetDir == "":
		// No target specified - use the default clone directory
		savePath = filepath.Join(cfg.DefaultCloneDir, repo.Name)
	case filepath.IsAbs(targetDir):
		// Absolute path - use directly
		savePath = targetDir
	case targetDir == "." || targetDir == "./":
		// Current directory - clone repo into cwd
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting current working directory: %w", err)
		}

		savePath = filepath.Join(wd, repo.Name)
	default:
		// Relative path - resolve to absolute
		absPath, err := filepath.Abs(targetDir)
		if err != nil {
			return nil, fmt.Errorf("error determining absolute path: %w", err)
		}

		savePath = absPath
	}

	// Create a parent directory if it doesn't exist
	parentDir := filepath.Dir(savePath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("error creating directory %s: %w", parentDir, err)
		}
	}

	// Check if the target directory already exists
	if info, err := os.Stat(savePath); err == nil && info.IsDir() {
		if !opts.Force {
			return nil, fmt.Errorf("directory already exists: %s\n\nUse --force to remove and re-clone", savePath)
		}

		// Force mode: remove the existing directory
		if err := os.RemoveAll(savePath); err != nil {
			return nil, fmt.Errorf("error removing existing directory: %w", err)
		}

		log.Printf("Removed existing directory: %s\n", savePath)
	}

	return &CloneResult{
		Repository: repo,
		CloneURL:   cloneURL,
		TargetPath: savePath,
		GitArgs:    gitArgs,
	}, nil
}

// getGitHubUsername tries to get the current GitHub username from git config or gh CLI
func getGitHubUsername() string {
	// Try gh CLI first
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")

	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// Fall back to git config
	cmd = exec.Command("git", "config", "--get", "github.user")

	output, err = cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// PrepareClonePath validates the URL and determines the target path for cloning.
//
// Deprecated: Use PrepareClone instead.
func PrepareClonePath(args []string, opts CloneOptions) (*url.URL, string, error) {
	result, err := PrepareClone(args, opts)
	if err != nil {
		return nil, "", err
	}

	// Convert to URL for backwards compatibility
	uri, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", result.Repository.Host, result.Repository.Owner, result.Repository.Name))
	if err != nil {
		return nil, "", err
	}

	return uri, result.TargetPath, nil
}

// SaveClonedRepo saves the successfully cloned repository to the database
func SaveClonedRepo(uri *url.URL, savePath string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := client.SaveRepo(uri, savePath); err != nil {
		return fmt.Errorf("error saving repo to database: %w", err)
	}

	// Fetch and save GitHub issues (non-blocking, errors are logged but don't fail clone)
	token := GetGitHubToken()
	if token != "" {
		_ = FetchAndSaveIssues(uri.String(), savePath, FetchIssuesOptions{
			Token: token,
		})
	}

	// Gather and save git statistics using git-nerds (non-blocking)
	_ = FetchAndSaveGitStats(uri.String(), savePath, FetchGitStatsOptions{
		IncludeTemporal: true,
		IncludeBranches: true,
	})

	log.Printf("Cloned repo at %s\n", savePath)

	return nil
}

// SaveClonedRepoFromResult saves the repository using CloneResult
func SaveClonedRepoFromResult(result *CloneResult) error {
	uri, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", result.Repository.Host, result.Repository.Owner, result.Repository.Name))
	if err != nil {
		return fmt.Errorf("error building URL: %w", err)
	}

	return SaveClonedRepo(uri, result.TargetPath)
}

// CloneRepo is the legacy function that clones and saves in one operation
func CloneRepo(args []string) error {
	return CloneRepoWithOptions(args, CloneOptions{})
}

// CloneRepoWithOptions clones a repository with the specified options (non-TUI mode)
func CloneRepoWithOptions(args []string, opts CloneOptions) error {
	result, err := PrepareClone(args, opts)
	if err != nil {
		return err
	}

	// Build git clone command
	// +2 for cloneURL and targetPath
	gitArgs := make([]string, 0, 1+len(result.GitArgs)+2)
	gitArgs = append(gitArgs, "clone")
	gitArgs = append(gitArgs, result.GitArgs...)
	gitArgs = append(gitArgs, result.CloneURL, result.TargetPath)

	runCmd := exec.Command("git", gitArgs...)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("git clone error: %w", err)
	}

	return SaveClonedRepoFromResult(result)
}

func PullRepo(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error git pull: %v, output: %s", err, string(output))
	}

	return nil
}
