package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "clonr",
	Short: "clonr - A Git wrapper to clone, monitor, and manage repositories.",
	Long:  `clonr is a command-line tool to efficiently clone, monitor, and manage multiple Git repositories.`,
	// RunE: func(cmd *cobra.Command, args []string) error {
	// 	p := tea.NewProgram(NewMenuModel())
	// 	if _, err := p.Run(); err != nil {
	// 		return err
	// 	}
	// 	return nil
	// },
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
