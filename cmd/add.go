package cmd

import (
	"fmt"
	"os"

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
			if !promptConfirm(fmt.Sprintf("Add '%s' to repositories? [y/N]: ", path)) {
				_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
				return nil
			}
		}
		id, err := core.AddRepo(path, core.AddOptions{Yes: addYes, Name: addName})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "Added: %s\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolVarP(&addYes, "yes", "y", false, "Skip confirmation prompt")
	addCmd.Flags().StringVar(&addName, "name", "", "Optional display name")
}
