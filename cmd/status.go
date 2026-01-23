package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show git status of repositories",
	Long:  `Display the git status of all managed repositories or a specific repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprintln(os.Stdout, "Status command - to be implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
