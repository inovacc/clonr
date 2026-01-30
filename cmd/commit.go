package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	commitMessage string
	commitAll     bool
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a git commit",
	Long: `Create a git commit in the current repository.

Examples:
  clonr commit -m "feat: add new feature"
  clonr commit -a -m "fix: resolve bug"`,
	RunE: runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Commit message (required)")
	commitCmd.Flags().BoolVarP(&commitAll, "all", "a", false, "Stage all modified files before committing")
	_ = commitCmd.MarkFlagRequired("message")
}

func runCommit(_ *cobra.Command, _ []string) error {
	// Check if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Stage all if -a flag is provided
	if commitAll {
		cmd := exec.Command("git", "add", "-A")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
	}

	// Create commit
	cmd := exec.Command("git", "commit", "-m", commitMessage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nCommit created successfully!")
	return nil
}
