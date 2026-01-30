package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var (
	pushTags     bool
	pushSetUp    bool
	pushForce    bool
)

var pushCmd = &cobra.Command{
	Use:   "push [remote] [branch]",
	Short: "Push changes to remote repository",
	Long: `Push changes to remote repository using profile authentication.

The command automatically uses the active profile token via credential helper
for authentication with private repositories.

Examples:
  clonr push
  clonr push --tags
  clonr push -u origin main
  clonr push origin feature-branch`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushTags, "tags", false, "Push all tags")
	pushCmd.Flags().BoolVarP(&pushSetUp, "set-upstream", "u", false, "Set upstream for the current branch")
	pushCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Force push")
}

func runPush(_ *cobra.Command, args []string) error {
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

	opts := git.PushOptions{
		SetUpstream: pushSetUp,
		Force:       pushForce,
		Tags:        pushTags,
	}

	// Use authenticated command via credential helper
	if err := client.Push(ctx, remote, branch, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Push completed successfully!")
	return nil
}
