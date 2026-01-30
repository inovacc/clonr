package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:    "auth",
	Short:  "Authentication commands",
	Hidden: true, // Internal commands
}

var gitCredentialCmd = &cobra.Command{
	Use:    "git-credential",
	Short:  "Git credential helper (internal use)",
	Long:   `This command is used as a git credential helper. It is called by git automatically.`,
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE:   runGitCredential,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(gitCredentialCmd)
}

func runGitCredential(_ *cobra.Command, args []string) error {
	// Git credential helper operations: get, store, erase
	// We only support "get" for now
	operation := "get"
	if len(args) > 0 {
		operation = args[0]
	}

	if operation != "get" {
		// Silently ignore store and erase operations
		return nil
	}

	// Read credential request from stdin
	wants := make(map[string]string)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			wants[parts[0]] = parts[1]
		}
	}

	// Only handle HTTPS
	if wants["protocol"] != "https" {
		return nil
	}

	host := wants["host"]
	if host == "" {
		return nil
	}

	// Resolve token from profile or environment
	token, _, _ := core.ResolveGitHubTokenForHost("", "", host)
	if token == "" {
		// No token available, let git try other credential helpers
		return nil
	}

	// Get username from active profile
	username := "x-access-token" // Default for token-based auth

	// Output credentials
	_, _ = fmt.Fprintf(os.Stdout, "protocol=https\n")
	_, _ = fmt.Fprintf(os.Stdout, "host=%s\n", host)
	_, _ = fmt.Fprintf(os.Stdout, "username=%s\n", username)
	_, _ = fmt.Fprintf(os.Stdout, "password=%s\n", token)

	return nil
}
