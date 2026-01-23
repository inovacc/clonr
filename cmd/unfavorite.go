package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var unfavoriteCmd = &cobra.Command{
	Use:   "unfavorite",
	Short: "Remove favorite mark from a repository",
	Long:  `Interactively select a favorite repository to unfavorite. Shows only repositories that are currently marked as favorites.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := cli.NewRepoList(true)
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
		if selected != nil {
			if err := core.SetFavoriteByURL(selected.URL, false); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "✓ Removed favorite from %s\n", selected.URL)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unfavoriteCmd)
}
