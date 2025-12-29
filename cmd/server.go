package cmd

import (
	"github.com/inovacc/clonr/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Long:  `Start the Clonr HTTP API server for remote repository management.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.StartServer(args)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
