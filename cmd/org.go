package cmd

import (
	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage GitHub organizations",
	Long: `Commands for working with GitHub organizations.

Available Commands:
  list    List your GitHub organizations
  mirror  Mirror all repositories from an organization`,
}

func init() {
	rootCmd.AddCommand(orgCmd)
}
