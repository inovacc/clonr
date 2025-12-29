package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/spf13/cobra"
)

var favoritesOnly bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively list all repositories",
	Long:  `Display all managed repositories in an interactive list. Use arrow keys to navigate and Enter to select actions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := cli.NewRepoList(favoritesOnly)
		if err != nil {
			return err
		}
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&favoritesOnly, "favorites", false, "Show only favorite repositories")
}
