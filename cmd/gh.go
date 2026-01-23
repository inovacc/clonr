package cmd

import (
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "GitHub operations for repositories",
	Long: `Interact with GitHub issues, PRs, actions, and releases.

Available Commands:
  issues    Manage GitHub issues (list, create)
  pr        Check pull request status
  actions   Check GitHub Actions workflow status
  release   Manage GitHub releases (create, download)

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Authentication:
  Uses GitHub token from (in priority order):
  1. --token flag
  2. GITHUB_TOKEN environment variable
  3. GH_TOKEN environment variable
  4. gh CLI authentication`,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

// addGHCommonFlags adds flags common to all gh subcommands
func addGHCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("token", "", "GitHub token (default: auto-detect)")
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().Bool("json", false, "Output as JSON")
}
