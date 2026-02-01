package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	configEditorCmd.AddCommand(configEditorAddCmd)
	configEditorAddCmd.Flags().StringVarP(&editorAddName, "name", "n", "", "Display name of the editor (required)")
	configEditorAddCmd.Flags().StringVarP(&editorAddCommand, "command", "c", "", "Executable command (required)")
	configEditorAddCmd.Flags().StringVarP(&editorAddIcon, "icon", "i", "", "Optional icon for display")

	_ = configEditorAddCmd.MarkFlagRequired("name")
	_ = configEditorAddCmd.MarkFlagRequired("command")

	configEditorCmd.AddCommand(configEditorListCmd)
	configEditorListCmd.Flags().BoolVarP(&editorListAll, "all", "a", false, "Show all editors (including not installed)")
	configEditorListCmd.Flags().BoolVar(&editorListInstalled, "installed", false, "Show only installed editors (default)")

	configEditorCmd.AddCommand(configEditorRemoveCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage clonr configuration",
	Long: `Commands for managing clonr configuration.

Available Commands:
  editor    Manage custom editors`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

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

var (
	editorAddName       string
	editorAddCommand    string
	editorAddIcon       string
	editorListAll       bool
	editorListInstalled bool
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

func runConfigEditorRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := core.RemoveCustomEditor(name); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Removed editor: %s\n", name)

	return nil
}
