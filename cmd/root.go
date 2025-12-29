package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "clonr",
	Short: "A Git repository manager",
	Long: `Clonr is a command-line tool for managing Git repositories efficiently.
It provides an interactive interface for cloning, organizing, and working with
multiple repositories.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
