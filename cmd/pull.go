package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var pullToken string

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull changes from remote repository",
	Long: `Pull changes from remote repository using profile authentication.

The command automatically uses the active profile token for authentication
with private repositories.

Examples:
  clonr pull
  clonr pull --token ghp_xxx`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringVar(&pullToken, "token", "", "GitHub token for authentication (uses active profile if not specified)")
}

func runPull(_ *cobra.Command, _ []string) error {
	// Check if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Get remote URL to determine host
	remoteOutput, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(remoteOutput))
	host := extractHostFromURL(remoteURL)

	// Resolve token
	token, _, _ := core.ResolveGitHubTokenForHost(pullToken, "", host)

	// If we have a token and it's an HTTPS URL, use credential helper
	if token != "" && strings.HasPrefix(remoteURL, "https://") {
		return pullWithToken(remoteURL, token)
	}

	// Standard pull without token injection
	cmd := exec.Command("git", "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

func pullWithToken(remoteURL, token string) error {
	// Temporarily set the remote URL with token
	authURL := injectTokenIntoGitURL(remoteURL, token)

	// Set the authenticated URL temporarily
	if err := exec.Command("git", "remote", "set-url", "origin", authURL).Run(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w", err)
	}

	// Ensure we restore the original URL
	defer func() {
		_ = exec.Command("git", "remote", "set-url", "origin", remoteURL).Run()
	}()

	// Pull
	cmd := exec.Command("git", "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

func extractHostFromURL(url string) string {
	// Handle https://github.com/... or https://token@github.com/...
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove token if present (token@host)
	if idx := strings.Index(url, "@"); idx != -1 {
		url = url[idx+1:]
	}

	// Get just the host part
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}

	return url
}

func injectTokenIntoGitURL(rawURL, token string) string {
	var scheme, rest string

	if strings.HasPrefix(rawURL, "https://") {
		scheme = "https://"
		rest = strings.TrimPrefix(rawURL, "https://")
	} else if strings.HasPrefix(rawURL, "http://") {
		scheme = "http://"
		rest = strings.TrimPrefix(rawURL, "http://")
	} else {
		return rawURL
	}

	// Remove existing credentials if present (user:pass@host or user@host)
	if idx := strings.Index(rest, "@"); idx != -1 {
		// Check if @ appears before the first /
		slashIdx := strings.Index(rest, "/")
		if slashIdx == -1 || idx < slashIdx {
			rest = rest[idx+1:]
		}
	}

	return scheme + token + "@" + rest
}
