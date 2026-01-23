package core

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inovacc/clonr/internal/grpcclient"
)

// CloneOptions configures the clone operation
type CloneOptions struct {
	Force bool // Force clone even if repo exists (removes existing)
}

// if dest dir is a dot clones into the current dir, if not,
// then clone into specified dir when dest dir not exists use default dir, saved in db

// PrepareClonePath validates the URL and determines the target path for cloning
func PrepareClonePath(args []string, opts CloneOptions) (*url.URL, string, error) {
	if len(args) < 1 {
		return nil, "", fmt.Errorf("repository URL is required")
	}

	uri, err := url.ParseRequestURI(args[0])
	if err != nil {
		return nil, "", fmt.Errorf("invalid repository URL: %w", err)
	}

	if uri.Scheme != "http" && uri.Scheme != "https" {
		return nil, "", fmt.Errorf("invalid repository URL: %s", uri.String())
	}

	if uri.Host == "" {
		return nil, "", fmt.Errorf("invalid repository URL: %s", uri.String())
	}

	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to server: %w", err)
	}

	// check for repo existence
	ok, err := client.RepoExistsByURL(uri)
	if err != nil {
		return nil, "", fmt.Errorf("error checking for repo existence: %w", err)
	}

	if ok {
		if !opts.Force {
			return nil, "", fmt.Errorf("repository already exists: %s\n\nUse --force to remove and re-clone", uri.String())
		}

		// Force mode: remove existing repo from database
		if err := client.RemoveRepoByURL(uri); err != nil {
			return nil, "", fmt.Errorf("error removing existing repo from database: %w", err)
		}

		log.Printf("Removed existing repo from database: %s\n", uri.String())
	}

	// Get config to determine default clone directory
	cfg, err := client.GetConfig()
	if err != nil {
		return nil, "", fmt.Errorf("error getting config: %w", err)
	}

	pathStr := cfg.DefaultCloneDir

	// Allow override via command-line argument
	if len(args) > 1 {
		pathStr = args[1]
	}

	if pathStr == "." || pathStr == "./" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("error getting current working directory: %w", err)
		}

		pathStr = wd
	}

	if _, err := os.Stat(pathStr); os.IsNotExist(err) {
		if err := os.MkdirAll(pathStr, os.ModePerm); err != nil {
			return nil, "", fmt.Errorf("error creating directory %s: %w", pathStr, err)
		}
	}

	absPath, err := filepath.Abs(pathStr)
	if err != nil {
		return nil, "", fmt.Errorf("error determining absolute path: %w", err)
	}

	savePath := filepath.Join(absPath, extractRepoName(uri.String()))

	// Check if target directory already exists
	if info, err := os.Stat(savePath); err == nil && info.IsDir() {
		if !opts.Force {
			return nil, "", fmt.Errorf("directory already exists: %s\n\nUse --force to remove and re-clone", savePath)
		}

		// Force mode: remove existing directory
		if err := os.RemoveAll(savePath); err != nil {
			return nil, "", fmt.Errorf("error removing existing directory: %w", err)
		}

		log.Printf("Removed existing directory: %s\n", savePath)
	}

	return uri, savePath, nil
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

// CloneRepo is the legacy function that clones and saves in one operation
func CloneRepo(args []string) error {
	uri, savePath, err := PrepareClonePath(args, CloneOptions{})
	if err != nil {
		return err
	}

	runCmd := exec.Command("git", "clone", uri.String(), savePath)

	output, err := runCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone error: %v - %s", err, string(output))
	}

	return SaveClonedRepo(uri, savePath)
}

func PullRepo(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error git pull: %v, output: %s", err, string(output))
	}

	return nil
}

func extractRepoName(url string) string {
	parts := strings.Split(url, "/")
	last := parts[len(parts)-1]

	return strings.TrimSuffix(last, ".git")
}
