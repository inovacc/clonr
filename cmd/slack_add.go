package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var (
	slackAddWebhook  string
	slackAddBotToken string
	slackAddChannel  string
)

var slackAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Slack webhook or bot integration",
	Long: `Configure Slack notifications using either a webhook URL or bot token.

Webhook Integration (Recommended for simple use):
  - Create an Incoming Webhook in your Slack workspace
  - Go to: https://api.slack.com/apps → Create New App → Incoming Webhooks
  - Copy the webhook URL and use --webhook flag

Bot Integration (For advanced features):
  - Create a Slack App with a bot user
  - Go to: https://api.slack.com/apps → Create New App → OAuth & Permissions
  - Add scopes: chat:write, chat:write.public
  - Install to workspace and copy the Bot User OAuth Token

The --channel flag specifies the default channel for notifications.
If not specified, the webhook's default channel is used.

Examples:
  clonr slack add --webhook https://hooks.slack.com/services/T00/B00/xxx
  clonr slack add --webhook https://hooks.slack.com/services/T00/B00/xxx --channel "#dev"
  clonr slack add --bot-token xoxb-xxx-xxx --channel "#notifications"`,
	RunE: runSlackAdd,
}

func init() {
	slackCmd.AddCommand(slackAddCmd)

	slackAddCmd.Flags().StringVar(&slackAddWebhook, "webhook", "", "Slack webhook URL")
	slackAddCmd.Flags().StringVar(&slackAddBotToken, "bot-token", "", "Slack bot token (xoxb-...)")
	slackAddCmd.Flags().StringVar(&slackAddChannel, "channel", "", "Default channel for notifications")
}

func runSlackAdd(_ *cobra.Command, _ []string) error {
	if slackAddWebhook == "" && slackAddBotToken == "" {
		return fmt.Errorf("either --webhook or --bot-token is required")
	}

	if slackAddWebhook != "" && slackAddBotToken != "" {
		return fmt.Errorf("specify either --webhook or --bot-token, not both")
	}

	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	if slackAddWebhook != "" {
		if err := manager.AddWebhook(slackAddWebhook, slackAddChannel); err != nil {
			return fmt.Errorf("failed to add webhook: %w", err)
		}

		_, _ = fmt.Fprintln(os.Stdout, "Slack webhook added successfully!")
		if slackAddChannel != "" {
			_, _ = fmt.Fprintf(os.Stdout, "Default channel: %s\n", slackAddChannel)
		}
		_, _ = fmt.Fprintln(os.Stdout, "\nTest with: clonr slack test")
		return nil
	}

	if slackAddBotToken != "" {
		if slackAddChannel == "" {
			return fmt.Errorf("--channel is required when using bot token")
		}

		if err := manager.AddBot(slackAddBotToken, slackAddChannel); err != nil {
			return fmt.Errorf("failed to add bot: %w", err)
		}

		_, _ = fmt.Fprintln(os.Stdout, "Slack bot added successfully!")
		_, _ = fmt.Fprintf(os.Stdout, "Default channel: %s\n", slackAddChannel)
		_, _ = fmt.Fprintln(os.Stdout, "\nTest with: clonr slack test")
		return nil
	}

	return nil
}
