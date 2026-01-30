package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
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
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	opts := git.CommitOptions{
		All: commitAll,
	}

	if err := client.Commit(ctx, commitMessage, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Commit created successfully!")
	return nil
}
