package cmd

import (
	"github.com/spf13/cobra"
)

var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Manage Slack notifications",
	Long: `Configure and manage Slack notifications for clonr events.

Clonr can send notifications to Slack when various events occur, such as:
- Push operations
- Clone operations
- CI/CD status changes
- Errors

You can use either a webhook URL (simpler setup) or a bot token (richer features).

Available Commands:
  add      Add Slack webhook or bot integration
  list     Show current Slack configuration
  remove   Remove Slack integration
  test     Send a test notification
  enable   Enable Slack notifications
  disable  Disable Slack notifications

Examples:
  clonr slack add --webhook https://hooks.slack.com/services/...
  clonr slack add --webhook https://hooks.slack.com/services/... --channel "#dev"
  clonr slack list
  clonr slack test
  clonr slack disable`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(slackCmd)
}
