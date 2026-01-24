package cmd

import (
	"github.com/spf13/cobra"
)

var pmCmd = &cobra.Command{
	Use:   "pm",
	Short: "Project management tool integrations",
	Long: `Interact with project management tools like Jira, ZenHub, and Linear.

Available Platforms:
  jira          Atlassian Jira (Cloud and Server)
  zenhub        ZenHub (GitHub-integrated project management)
  linear        Linear (issue tracking)

Project Detection:
  Commands auto-detect the project from repository context when possible,
  or you can specify it explicitly.

Authentication:
  Each platform uses its own authentication:

  Jira:
    1. --token flag
    2. JIRA_API_TOKEN environment variable
    3. ATLASSIAN_TOKEN environment variable
    4. ~/.config/clonr/jira.json config file

  ZenHub:
    1. --token flag
    2. ZENHUB_TOKEN environment variable
    3. ~/.config/clonr/zenhub.json config file

  Linear:
    1. --token flag
    2. LINEAR_API_KEY environment variable
    3. ~/.config/clonr/linear.json config file`,
}

func init() {
	rootCmd.AddCommand(pmCmd)
}

// addPMCommonFlags adds flags common to all pm subcommands
func addPMCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("token", "", "API token (default: auto-detect)")
	cmd.Flags().Bool("json", false, "Output as JSON")
}
