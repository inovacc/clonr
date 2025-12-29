package cmd

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/database"
	"github.com/spf13/cobra"
	"os/exec"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a repository in your configured editor",
	Long:  `Interactively select a repository to open in your configured editor. The editor can be configured using the 'clonr configure' command.`,
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
		if selected == nil {
			return nil
		}
		db := database.GetDB()
		cfg, err := db.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get config: %w", err)
		}
		if cfg.Editor == "" {
			return fmt.Errorf("no editor configured. Run 'clonr configure' to set an editor")
		}
		fmt.Printf("Opening %s in %s...\n", selected.Path, cfg.Editor)
		execCmd := exec.Command(cfg.Editor, selected.Path)
		if err := execCmd.Start(); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		fmt.Printf("✓ Opened %s\n", selected.URL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
