package core

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inovacc/clonr/internal/database"
)

// if dest dir is a dot clones into the current dir, if not,
// then clone into specified dir when dest dir not exists use default dir, saved in db

// PrepareClonePath validates the URL and determines the target path for cloning
func PrepareClonePath(args []string) (*url.URL, string, error) {
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

	db := database.GetDB()

	// check for repo existence
	ok, err := db.RepoExistsByURL(uri)
	if err != nil {
		return nil, "", fmt.Errorf("error checking for repo existence: %w", err)
	}

	if ok {
		return nil, "", fmt.Errorf("repository already exists: %s", uri.String())
	}

	// Get config to determine default clone directory
	cfg, err := db.GetConfig()
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

	return uri, savePath, nil
}

// SaveClonedRepo saves the successfully cloned repository to the database
func SaveClonedRepo(uri *url.URL, savePath string) error {
	db := database.GetDB()

	if err := db.SaveRepo(uri, savePath); err != nil {
		return fmt.Errorf("error saving repo to database: %w", err)
	}

	log.Printf("Cloned repo at %s\n", savePath)

	return nil
}

// CloneRepo is the legacy function that clones and saves in one operation
func CloneRepo(args []string) error {
	uri, savePath, err := PrepareClonePath(args)
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
