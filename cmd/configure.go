package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var (
	showConfig  bool
	resetConfig bool
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure clonr settings",
	Long:  `Interactively configure Clonr settings such as default clone directory, editor, and server port.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showConfig {
			return core.ShowConfig()
		}

		if resetConfig {
			return core.ResetConfig()
		}

		if err := core.ShowConfig(); err != nil {
			fmt.Println("No configuration found, using defaults.")
		}

		fmt.Println("\nStarting interactive configuration...")

		m, err := cli.NewConfigureModel()
		if err != nil {
			return err
		}

		p := tea.NewProgram(&m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		configModel := finalModel.(cli.ConfigureModel)
		if configModel.Err != nil {
			return configModel.Err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().BoolVarP(&showConfig, "show", "s", false, "Show current configuration")
	configureCmd.Flags().BoolVarP(&resetConfig, "reset", "r", false, "Reset configuration to defaults")
}
