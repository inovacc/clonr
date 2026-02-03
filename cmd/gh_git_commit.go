package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var ghGitCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Create a git commit",
	Long: `Record changes to the repository.

Creates a new commit containing the current contents of the index
and the given log message describing the changes.

Examples:
  clonr gh git commit -m "feat: add new feature"
  clonr gh git commit -a -m "fix: resolve bug"
  clonr gh git commit --all -m "chore: update dependencies"`,
	RunE: runGhGitCommit,
}

func init() {
	ghGitCmd.AddCommand(ghGitCommitCmd)
	ghGitCommitCmd.Flags().StringP("message", "m", "", "Commit message (required)")
	ghGitCommitCmd.Flags().BoolP("all", "a", false, "Stage all modified and deleted files")
	_ = ghGitCommitCmd.MarkFlagRequired("message")
}

func runGhGitCommit(cmd *cobra.Command, _ []string) error {
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

	return nil
}
