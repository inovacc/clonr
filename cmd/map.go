package cmd

import (
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map [directory]",
	Short: "Scan directory for existing Git repositories",
	Long:  `Recursively scan a directory to find existing Git repositories and register them with Clonr for management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return core.MapRepos(args)
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
