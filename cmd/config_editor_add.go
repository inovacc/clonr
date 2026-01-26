package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var (
	editorAddName    string
	editorAddCommand string
	editorAddIcon    string
)

var configEditorAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new custom editor",
	Long: `Add a new custom editor to the configuration.

The editor will be available in the editor selection list when using 'clonr repo edit'.

Examples:
  clonr config editor add --name "VS Code Insiders" --command code-insiders
  clonr config editor add --name "My Editor" --command myeditor --icon "📝"`,
	RunE: runConfigEditorAdd,
}

func init() {
	configEditorCmd.AddCommand(configEditorAddCmd)
	configEditorAddCmd.Flags().StringVarP(&editorAddName, "name", "n", "", "Display name of the editor (required)")
	configEditorAddCmd.Flags().StringVarP(&editorAddCommand, "command", "c", "", "Executable command (required)")
	configEditorAddCmd.Flags().StringVarP(&editorAddIcon, "icon", "i", "", "Optional icon for display")

	_ = configEditorAddCmd.MarkFlagRequired("name")
	_ = configEditorAddCmd.MarkFlagRequired("command")
}

func runConfigEditorAdd(cmd *cobra.Command, args []string) error {
	editor := model.Editor{
		Name:    editorAddName,
		Command: editorAddCommand,
		Icon:    editorAddIcon,
	}

	if err := core.AddCustomEditor(editor); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Added editor: %s (%s)\n", editor.Name, editor.Command)

	return nil
}
