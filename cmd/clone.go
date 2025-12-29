package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [destination]",
	Short: "Clone a Git repository",
	Long:  `Clone a Git repository and register it with Clonr. Supports https, http, git, ssh, ftp, sftp, and git@ URLs.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL, targetPath, err := core.PrepareClonePath(args)
		if err != nil {
			return err
		}
		m := cli.NewCloneModel(repoURL.String(), targetPath)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		cloneModel := finalModel.(cli.CloneModel)
		if cloneModel.Error() != nil {
			return cloneModel.Error()
		}
		return core.SaveClonedRepo(repoURL, targetPath)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
