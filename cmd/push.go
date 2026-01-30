package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var (
	pushToken    string
	pushTags     bool
	pushSetUp    bool
	pushForce    bool
	pushUpstream string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push changes to remote repository",
	Long: `Push changes to remote repository using profile authentication.

The command automatically uses the active profile token for authentication
with private repositories.

Examples:
  clonr push
  clonr push --tags
  clonr push -u origin main
  clonr push --token ghp_xxx`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVar(&pushToken, "token", "", "GitHub token for authentication (uses active profile if not specified)")
	pushCmd.Flags().BoolVar(&pushTags, "tags", false, "Push all tags")
	pushCmd.Flags().BoolVarP(&pushSetUp, "set-upstream", "u", false, "Set upstream for the current branch")
	pushCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Force push")
	pushCmd.Flags().StringVar(&pushUpstream, "upstream", "", "Upstream remote/branch (e.g., origin main)")
}

func runPush(_ *cobra.Command, args []string) error {
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
	token, _, _ := core.ResolveGitHubTokenForHost(pushToken, "", host)

	// Build push arguments
	pushArgs := []string{"push"}

	if pushSetUp && len(args) >= 2 {
		pushArgs = append(pushArgs, "-u", args[0], args[1])
	} else if pushSetUp && pushUpstream != "" {
		parts := strings.Fields(pushUpstream)
		if len(parts) >= 2 {
			pushArgs = append(pushArgs, "-u", parts[0], parts[1])
		}
	}

	if pushTags {
		pushArgs = append(pushArgs, "--tags")
	}

	if pushForce {
		pushArgs = append(pushArgs, "--force")
	}

	// If we have a token and it's an HTTPS URL, use credential helper
	if token != "" && strings.HasPrefix(remoteURL, "https://") {
		return pushWithToken(remoteURL, token, pushArgs)
	}

	// Standard push without token injection
	cmd := exec.Command("git", pushArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

func pushWithToken(remoteURL, token string, pushArgs []string) error {
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

	// Push
	cmd := exec.Command("git", pushArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}
