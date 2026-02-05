package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitPullCmd = &cobra.Command{
	Use:   "pull [remote] [branch]",
	Short: "Fetch and integrate with remote repository",
	Long: `Fetch from and integrate with another repository or local branch.

Uses clonr profile authentication automatically for private repositories.

Examples:
  clonr git pull
  clonr git pull origin
  clonr git pull origin main`,
	RunE: runGitPull,
}

func init() {
	gitCmd.AddCommand(gitPullCmd)
}

func runGitPull(_ *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	var remote, branch string
	if len(args) >= 1 {
		remote = args[0]
	}

	if len(args) >= 2 {
		branch = args[1]
	}

	// Show current branch
	currentBranch, err := client.CurrentBranch(ctx)
	if err == nil {
		_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Pulling into '%s'...\n"), currentBranch)
	}

	if err := client.Pull(ctx, remote, branch); err != nil {
		if git.IsAuthRequired(err) {
			return fmt.Errorf("authentication failed - check your profile token")
		}

		if git.IsConflict(err) {
			return fmt.Errorf("merge conflict detected - resolve conflicts and commit")
		}

		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Pull completed successfully!"))

	// Send pull notification (async, non-blocking)
	repoPath, _ := os.Getwd()
	go core.NotifyPull(ctx, repoPath)

	return nil
}
