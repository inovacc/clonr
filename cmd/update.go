package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Pull latest changes for all or specific repository",
	Long:  `Update repositories by pulling the latest changes from their remotes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Update command - to be implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
