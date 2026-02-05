package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var slackRemoveForce bool

var slackRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove Slack integration",
	Long: `Remove the Slack integration configuration.

This will delete the webhook URL or bot token and all event settings.
You will need to reconfigure Slack notifications after removal.

Examples:
  clonr slack remove
  clonr slack remove --force`,
	RunE: runSlackRemove,
}

func init() {
	slackCmd.AddCommand(slackRemoveCmd)
	slackRemoveCmd.Flags().BoolVarP(&slackRemoveForce, "force", "f", false, "Skip confirmation")
}

func runSlackRemove(_ *cobra.Command, _ []string) error {
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
		_, _ = fmt.Fprintln(os.Stdout, "Slack is not configured.")
		return nil
	}

	// Confirm removal
	if !slackRemoveForce {
		if !promptConfirm("Remove Slack integration? [y/N]: ") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := manager.Remove(); err != nil {
		return fmt.Errorf("failed to remove Slack integration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Slack integration removed.")

	return nil
}
