package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var (
	editorListAll       bool
	editorListInstalled bool
)

var configEditorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all editors",
	Long: `List all available editors (default + custom).

By default, shows only installed editors. Use --all to show all editors.

Examples:
  clonr config editor list              # List installed editors
  clonr config editor list --all        # List all editors (including not installed)
  clonr config editor list --installed  # List only installed editors`,
	RunE: runConfigEditorList,
}

func init() {
	configEditorCmd.AddCommand(configEditorListCmd)
	configEditorListCmd.Flags().BoolVarP(&editorListAll, "all", "a", false, "Show all editors (including not installed)")
	configEditorListCmd.Flags().BoolVar(&editorListInstalled, "installed", false, "Show only installed editors (default)")
}

func runConfigEditorList(cmd *cobra.Command, args []string) error {
	editors, err := core.GetAllEditors()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Available Editors:")
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Get custom editor names for marking
	customEditors, err := core.GetCustomEditors()
	if err != nil {
		return err
	}

	customNames := make(map[string]bool)
	for _, e := range customEditors {
		customNames[e.Name] = true
	}

	for _, editor := range editors {
		installed := core.IsEditorInstalled(editor.Command)

		if !editorListAll && !installed {
			continue
		}

		status := "✓"
		if !installed {
			status = "✗"
		}

		customMark := ""
		if customNames[editor.Name] {
			customMark = " [custom]"
		}

		icon := ""
		if editor.Icon != "" {
			icon = editor.Icon + " "
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s %s%s (%s)%s\n", status, icon, editor.Name, editor.Command, customMark)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	_, _ = fmt.Fprintln(os.Stdout, "Legend: ✓ installed, ✗ not installed, [custom] user-added")

	return nil
}
