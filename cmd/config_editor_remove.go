package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var configEditorRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a custom editor",
	Long: `Remove a custom editor from the configuration.

Only custom editors can be removed. Default editors cannot be removed.

Examples:
  clonr config editor remove "My Editor"
  clonr config editor remove "VS Code Insiders"`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigEditorRemove,
}

func init() {
	configEditorCmd.AddCommand(configEditorRemoveCmd)
}

func runConfigEditorRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := core.RemoveCustomEditor(name); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Removed editor: %s\n", name)

	return nil
}
