package cmd

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove repository from management",
	Long:    `Interactively select a repository to remove from Clonr's management. This only removes the repository from Clonr's database; the files remain on disk.`,
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
			fmt.Printf("Removing repository: %s\n", selected.URL)
			return core.RemoveRepo(selected.URL)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
