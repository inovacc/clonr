package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var removeURL string

var removeCmd = &cobra.Command{
	Use:     "remove [url]",
	Aliases: []string{"rm"},
	Short:   "Remove repository from management",
	Long: `Remove a repository from Clonr's management. This only removes the repository
from Clonr's database; the files remain on disk.

You can specify the repository URL as an argument or use the interactive list.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Non-interactive mode: URL provided as argument or flag
		url := removeURL
		if len(args) > 0 {
			url = args[0]
		}

		if url != "" {
			fmt.Printf("Removing repository: %s\n", url)
			if err := core.RemoveRepo(url); err != nil {
				return fmt.Errorf("failed to remove repository: %w", err)
			}
			fmt.Println("Repository removed successfully")
			return nil
		}

		// Interactive mode
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
	removeCmd.Flags().StringVar(&removeURL, "url", "", "Repository URL to remove (non-interactive)")
	rootCmd.AddCommand(removeCmd)
}
