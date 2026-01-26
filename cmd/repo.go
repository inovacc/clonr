package cmd

import (
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Repository operations",
	Long: `Commands for opening and editing repositories.

Available Commands:
  open    Open repository folder in file manager
  edit    Open repository in selected editor`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
