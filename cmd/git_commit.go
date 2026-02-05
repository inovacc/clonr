package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a git commit",
	Long: `Record changes to the repository.

Creates a new commit containing the current contents of the index
and the given log message describing the changes.

Examples:
  clonr git commit -m "feat: add new feature"
  clonr git commit -a -m "fix: resolve bug"
  clonr git commit --all -m "chore: update dependencies"`,
	RunE: runGitCommit,
}

func init() {
	gitCmd.AddCommand(gitCommitCmd)
	gitCommitCmd.Flags().StringP("message", "m", "", "Commit message (required)")
	gitCommitCmd.Flags().BoolP("all", "a", false, "Stage all modified and deleted files")
	_ = gitCommitCmd.MarkFlagRequired("message")
}

func runGitCommit(cmd *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	message, _ := cmd.Flags().GetString("message")
	all, _ := cmd.Flags().GetBool("all")

	opts := git.CommitOptions{
		All: all,
	}

	if err := client.Commit(ctx, message, opts); err != nil {
		if git.IsNothingToCommit(err) {
			_, _ = fmt.Fprintln(os.Stdout, "nothing to commit, working tree clean")
			return nil
		}

		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Commit created successfully!"))

	// Show the commit that was just created
	head, err := client.GetShortHead(ctx)
	if err == nil {
		_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("  commit: %s\n"), head)
	}

	// Send commit notification (async, non-blocking)
	repoPath, _ := os.Getwd()
	go core.NotifyCommit(ctx, repoPath, head, message)

	return nil
}
