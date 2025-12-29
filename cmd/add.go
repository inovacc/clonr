package cmd

import (
	"fmt"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var (
	addYes  bool
	addName string
)

var addCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Register an existing local Git repository",
	Long: `Add an existing Git repository to Clonr's management.
This allows you to track and manage repositories that were cloned outside of Clonr.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if !addYes {
			fmt.Printf("Add '%s' to repositories? [y/N]: ", path)
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		id, err := core.AddRepo(path, core.AddOptions{Yes: addYes, Name: addName})
		if err != nil {
			return err
		}
		fmt.Printf("Added: %s\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&addYes, "yes", "y", false, "Skip confirmation prompt")
	addCmd.Flags().StringVar(&addName, "name", "", "Optional display name")
}
