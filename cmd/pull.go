package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [remote] [branch]",
	Short: "Pull changes from remote repository",
	Long: `Pull changes from remote repository using profile authentication.

The command automatically uses the active profile token via credential helper
for authentication with private repositories.

Examples:
  clonr pull
  clonr pull origin main`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(_ *cobra.Command, args []string) error {
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

	// Use authenticated command via credential helper
	if err := client.Pull(ctx, remote, branch); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Pull completed successfully!")

	return nil
}
