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
	Long: `Clone a Git repository and register it with Clonr. Supports https, http, git, ssh, ftp, sftp, and git@ URLs.

If no destination is specified, the repository is cloned to the default clone directory
configured in settings (default: ~/clonr). Use 'clonr configure' to change this.

Use --force to remove and re-clone if the repository already exists in the database or the target directory already exists.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		opts := core.CloneOptions{
			Force: force,
		}

		repoURL, targetPath, err := core.PrepareClonePath(args, opts)
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
	cloneCmd.Flags().BoolP("force", "f", false, "Force clone even if repository/directory already exists (removes existing)")
}
