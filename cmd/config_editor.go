package cmd

import (
	"github.com/spf13/cobra"
)

var configEditorCmd = &cobra.Command{
	Use:   "editor",
	Short: "Manage custom editors",
	Long: `Commands for managing custom editors in clonr.

Custom editors are added to the list of available editors when using 'clonr repo edit'.

Available Commands:
  add       Add a new custom editor
  remove    Remove a custom editor
  list      List all editors (default + custom)

Examples:
  clonr config editor add --name "My Editor" --command myeditor
  clonr config editor remove "My Editor"
  clonr config editor list`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	configCmd.AddCommand(configEditorCmd)
}
