package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var slackEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Slack notifications",
	Long: `Enable Slack notifications.

This turns on Slack notifications without changing any configuration.
Use this to re-enable notifications after using 'clonr slack disable'.

Examples:
  clonr slack enable`,
	RunE: runSlackEnable,
}

var slackDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Slack notifications",
	Long: `Temporarily disable Slack notifications.

This turns off Slack notifications without removing the configuration.
Your webhook/bot token and event settings are preserved.
Use 'clonr slack enable' to turn notifications back on.

Examples:
  clonr slack disable`,
	RunE: runSlackDisable,
}

func init() {
	slackCmd.AddCommand(slackEnableCmd)
	slackCmd.AddCommand(slackDisableCmd)
}

func runSlackEnable(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	// Check if configured
	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("slack is not configured\nSet up with: clonr slack add --webhook <url>")
	}

	if config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Slack notifications are already enabled.")
		return nil
	}

	if err := manager.Enable(); err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Slack notifications enabled.")

	return nil
}

func runSlackDisable(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	// Check if configured
	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("slack is not configured")
	}

	if !config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Slack notifications are already disabled.")
		return nil
	}

	if err := manager.Disable(); err != nil {
		return fmt.Errorf("failed to disable notifications: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Slack notifications disabled.")
	_, _ = fmt.Fprintln(os.Stdout, "Re-enable with: clonr slack enable")

	return nil
}
