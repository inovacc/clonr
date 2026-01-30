package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var (
	mergeNoFF    bool
	mergeSquash  bool
	mergeMessage string
)

var mergeCmd = &cobra.Command{
	Use:   "merge <branch>",
	Short: "Merge a branch into current branch",
	Long: `Merge a branch into the current branch.

Examples:
  clonr merge feature              # Merge feature branch
  clonr merge feature --no-ff      # Merge with merge commit
  clonr merge feature --squash     # Squash merge
  clonr merge feature -m "Merge"   # Merge with custom message`,
	Args: cobra.ExactArgs(1),
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	mergeCmd.Flags().BoolVar(&mergeNoFF, "no-ff", false, "Create merge commit even for fast-forward")
	mergeCmd.Flags().BoolVar(&mergeSquash, "squash", false, "Squash commits into one")
	mergeCmd.Flags().StringVarP(&mergeMessage, "message", "m", "", "Merge commit message")
}

func runMerge(_ *cobra.Command, args []string) error {
	branch := args[0]

	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	opts := git.MergeOptions{
		NoFastForward: mergeNoFF,
		Squash:        mergeSquash,
		Message:       mergeMessage,
	}

	if err := client.Merge(ctx, branch, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Merged '%s' successfully!\n", branch)

	return nil
}
