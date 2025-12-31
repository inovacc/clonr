package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var favoriteCmd = &cobra.Command{
	Use:   "favorite",
	Short: "Mark a repository as favorite",
	Long:  `Interactively select a repository to mark as favorite. Favorited repositories can be quickly accessed and filtered.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := cli.NewRepoList(false)
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
			if err := core.SetFavoriteByURL(selected.URL, true); err != nil {
				return err
			}
			fmt.Printf("✓ Marked %s as favorite\n", selected.URL)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(favoriteCmd)
}
