package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var repoOpenCmd = &cobra.Command{
	Use:   "open [path]",
	Short: "Open repository folder in file manager",
	Long: `Open a repository folder in the system's default file manager.

If no path is provided, an interactive list will be shown to select a repository.

Examples:
  clonr repo open                    # Interactive selection
  clonr repo open ~/projects/myrepo  # Open specific path
  clonr repo open --favorites        # Show only favorites`,
	RunE: runRepoOpen,
}

func init() {
	repoCmd.AddCommand(repoOpenCmd)
	repoOpenCmd.Flags().Bool("favorites", false, "Show only favorite repositories")
}

func runRepoOpen(cmd *cobra.Command, args []string) error {
	// If path provided directly, open it
	if len(args) > 0 {
		path := args[0]

		if err := core.OpenInFileManager(path); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(os.Stdout, "Opened %s in file manager\n", path)

		return nil
	}

	// Interactive selection
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

	if err := core.OpenInFileManager(selected.Path); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Opened %s in file manager\n", selected.Path)

	return nil
}
