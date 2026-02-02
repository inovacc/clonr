package cmd

import (
	"os"

	"github.com/inovacc/clonr/internal/application"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   application.AppName,
	Short: "A Git repository manager",
	Long: `Clonr is a command-line tool for managing Git repositories efficiently.
It provides an interactive interface for cloning, organizing, and working with
multiple repositories.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// GetRootCmd returns the root command for introspection purposes.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	// Global flags can be added here if needed
}
