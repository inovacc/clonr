package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var repoEditCmd = &cobra.Command{
	Use:   "edit [path]",
	Short: "Open repository in selected editor",
	Long: `Open a repository in an editor of your choice.

First select a repository (or provide path), then select an editor from installed options.

Examples:
  clonr repo edit                    # Interactive selection
  clonr repo edit ~/projects/myrepo  # Edit specific path
  clonr repo edit --favorites        # Show only favorites
  clonr repo edit --editor code      # Skip editor selection`,
	RunE: runRepoEdit,
}

func init() {
	repoCmd.AddCommand(repoEditCmd)
	repoEditCmd.Flags().Bool("favorites", false, "Show only favorite repositories")
	repoEditCmd.Flags().StringP("editor", "e", "", "Editor to use (skip editor selection)")
}

func runRepoEdit(cmd *cobra.Command, args []string) error {
	var repoPath string

	// Get repository path
	if len(args) > 0 {
		repoPath = args[0]
	} else {
		// Interactive repository selection
		favoritesOnly, _ := cmd.Flags().GetBool("favorites")

		m, err := cli.NewRepoList(favoritesOnly)
		if err != nil {
			return err
		}

		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		repoModel := finalModel.(cli.RepoListModel)
		selected := repoModel.GetSelectedRepo()

		if selected == nil {
			return nil
		}

		repoPath = selected.Path
	}

	// Get editor
	editorFlag, _ := cmd.Flags().GetString("editor")

	var editorCmd string

	if editorFlag != "" {
		// Use provided editor
		if !core.IsEditorInstalled(editorFlag) {
			return fmt.Errorf("editor %q is not installed", editorFlag)
		}

		editorCmd = editorFlag
	} else {
		// Interactive editor selection
		m, err := cli.NewEditorList()
		if err != nil {
			return err
		}

		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		editorModel := finalModel.(cli.EditorListModel)
		selected := editorModel.GetSelectedEditor()

		if selected == nil {
			return nil
		}

		editorCmd = selected.Command
	}

	// Open in editor
	if err := core.OpenInEditor(editorCmd, repoPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Opened %s in %s\n", repoPath, editorCmd)

	return nil
}
