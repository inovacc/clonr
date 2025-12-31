package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nerdsCmd = &cobra.Command{
	Use:   "nerds [name]",
	Short: "Display repository statistics",
	Long:  `Show detailed statistics and metrics for all repositories or a specific repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Nerds command - to be implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(nerdsCmd)
}
