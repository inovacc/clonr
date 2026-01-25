package cmd

import (
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "GitHub operations for repositories",
	Long: `Interact with GitHub issues, PRs, actions, releases, and contributors.

Available Commands:
  issues        Manage GitHub issues (list, create, close)
  pr            Check pull request status
  actions       Check GitHub Actions workflow status
  release       Manage GitHub releases (create, download)
  contributors  View contributors and their activity journey

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Authentication:
  Uses GitHub token from (in priority order):
  1. --token flag
  2. --profile flag (clonr profile token)
  3. GITHUB_TOKEN environment variable
  4. GH_TOKEN environment variable
  5. Active clonr profile token
  6. gh CLI authentication`,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

// addGHCommonFlags adds flags common to all gh subcommands
func addGHCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("token", "", "GitHub token (default: auto-detect)")
	cmd.Flags().String("profile", "", "Use token from specified profile")
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().Bool("json", false, "Output as JSON")
}
