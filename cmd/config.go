package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage clonr configuration",
	Long: `Commands for managing clonr configuration.

Available Commands:
  editor    Manage custom editors`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
