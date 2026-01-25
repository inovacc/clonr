package cmd

import (
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage GitHub authentication profiles",
	Long: `Manage GitHub authentication profiles for clonr.

Each profile stores GitHub credentials using OAuth device flow authentication.
Tokens are stored securely in the system keyring when available.

Available Commands:
  add          Create a new profile with GitHub OAuth
  list         List all profiles
  use          Set the active profile
  remove       Delete a profile
  status       Show current profile information

Examples:
  clonr profile add work
  clonr profile list
  clonr profile use work
  clonr profile status
  clonr profile remove old-profile`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand provided, show help
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
}
