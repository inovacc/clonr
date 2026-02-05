package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
	"github.com/spf13/cobra"
)

// slackCmd is the top-level slack command
var slackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slack workspace integration",
	Long: `Manage Slack workspace integration for reading and sending messages.

Slack credentials are stored securely using profile encryption
(TPM-backed when available).

Authentication Commands:
  add          Add Slack integration via OAuth or bot token
  remove       Remove Slack integration
  status       Show Slack integration status
  accounts     List all Slack accounts

Operation Commands:
  channels     List Slack channels
  messages     Read messages from a channel
  search       Search for messages
  thread       View thread replies
  users        List workspace users

Notification Commands:
  notify       Manage Slack notifications (webhooks)

Examples:
  clonr slack add --client-id <id> --client-secret <secret>
  clonr slack add --token xoxb-xxxx
  clonr slack status
  clonr slack channels
  clonr slack messages --channel general
  clonr slack search "deployment"`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// slackAddCmd adds Slack integration via OAuth or token
var slackAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Slack integration",
	Long: `Add Slack integration via OAuth or bot token.

By default, this starts an OAuth flow where you:
1. A browser window opens to Slack authorization
2. Authorize clonr for your workspace
3. The bot token is automatically saved

Alternatively, provide a bot token directly with --token to skip OAuth.

Prerequisites for OAuth:
  1. Create a Slack App at https://api.slack.com/apps
  2. Add OAuth redirect URL: http://localhost:8338/slack/callback
  3. Add required Bot Token Scopes:
     - channels:read, channels:history
     - groups:read, groups:history
     - search:read, users:read
  4. Get Client ID and Client Secret from "Basic Information"

Examples:
  clonr slack add --client-id <id> --client-secret <secret>
  clonr slack add --token xoxb-xxxxxxxxxxxx
  SLACK_CLIENT_ID=xxx SLACK_CLIENT_SECRET=yyy clonr slack add`,
	RunE: runSlackAdd,
}

// slackRemoveCmd removes Slack integration
var slackRemoveCmd = &cobra.Command{
	Use:     "remove [account-name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove Slack integration",
	Long: `Remove Slack integration from the profile.

If no account name is specified, removes from the active profile.
Use --force to skip confirmation.

Examples:
  clonr slack remove
  clonr slack remove work-slack
  clonr slack remove --force`,
	RunE: runSlackRemove,
}

// slackStatusCmd shows Slack integration status
var slackStatusCmd = &cobra.Command{
	Use:   "status [account-name]",
	Short: "Show Slack integration status",
	Long: `Show Slack integration status.

Displays workspace information, connection status, and configured channels.

Examples:
  clonr slack status
  clonr slack status work-slack
  clonr slack status --json`,
	RunE: runSlackStatus,
}

// slackAccountsCmd lists all Slack accounts
var slackAccountsCmd = &cobra.Command{
	Use:     "accounts",
	Aliases: []string{"list", "ls"},
	Short:   "List all Slack accounts",
	Long: `List all configured Slack accounts across all profiles.

Examples:
  clonr slack accounts
  clonr slack accounts --json`,
	RunE: runSlackAccounts,
}

// slackChannelsCmd lists Slack channels
var slackChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List Slack channels",
	Long: `List channels in the Slack workspace.

Shows:
  - Channel name and ID
  - Member count
  - Topic/purpose
  - Public/private status

Examples:
  clonr slack channels
  clonr slack channels --private
  clonr slack channels --json`,
	RunE: runSlackChannels,
}

// slackMessagesCmd reads messages from a channel
var slackMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Read messages from a channel",
	Long: `Read recent messages from a Slack channel.

Shows:
  - Message author and timestamp
  - Message text
  - Reactions and thread reply counts
  - Attachments and files

Examples:
  clonr slack messages --channel general
  clonr slack messages --channel C01234567 --limit 50
  clonr slack messages --channel dev --since 24h
  clonr slack messages --channel general --json`,
	RunE: runSlackMessages,
}

// slackSearchCmd searches for messages
var slackSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for messages",
	Long: `Search for messages in Slack.

Supports Slack search modifiers:
  in:#channel          Search in specific channel
  from:@user           Search from specific user
  has:link             Messages with links
  has:reaction         Messages with reactions
  before:YYYY-MM-DD    Before date
  after:YYYY-MM-DD     After date

Examples:
  clonr slack search "deployment error"
  clonr slack search "in:#dev bug"
  clonr slack search "from:@john error" --limit 50
  clonr slack search "has:reaction" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSlackSearch,
}

// slackThreadCmd views thread replies
var slackThreadCmd = &cobra.Command{
	Use:   "thread",
	Short: "View thread replies",
	Long: `View replies to a message thread.

Requires the channel and the parent message timestamp (ts).
You can find the timestamp from the message list or search results.

Examples:
  clonr slack thread --channel general --ts 1234567890.123456
  clonr slack thread --channel C01234567 --ts 1234567890.123456 --json`,
	RunE: runSlackThread,
}

// slackUsersCmd lists workspace users
var slackUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List workspace users",
	Long: `List users in the Slack workspace.

Shows:
  - User name and display name
  - Real name and email
  - Status and timezone
  - Admin/bot status

Examples:
  clonr slack users
  clonr slack users --limit 50
  clonr slack users --json`,
	RunE: runSlackUsers,
}

// slackNotifyCmd manages Slack notifications (webhooks)
var slackNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Manage Slack notifications",
	Long: `Manage Slack webhook notifications for clonr events.

Clonr can send notifications to Slack when various events occur:
  - Push operations
  - Clone operations
  - CI/CD status changes
  - Errors

Subcommands:
  add      Add webhook for notifications
  remove   Remove webhook
  list     Show notification config
  test     Send test notification
  enable   Enable notifications
  disable  Disable notifications

Examples:
  clonr slack notify add --webhook https://hooks.slack.com/services/...
  clonr slack notify list
  clonr slack notify test`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// slackNotifyAddCmd adds webhook for notifications
var slackNotifyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Slack webhook for notifications",
	Long: `Configure Slack notifications using a webhook URL.

Create an Incoming Webhook in your Slack workspace:
  1. Go to: https://api.slack.com/apps
  2. Create New App → Incoming Webhooks
  3. Copy the webhook URL

Examples:
  clonr slack notify add --webhook https://hooks.slack.com/services/T00/B00/xxx
  clonr slack notify add --webhook https://hooks.slack.com/services/... --channel "#dev"`,
	RunE: runSlackNotifyAdd,
}

// slackNotifyRemoveCmd removes webhook
var slackNotifyRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"rm"},
	Short:   "Remove Slack webhook",
	Long: `Remove the Slack webhook configuration.

Examples:
  clonr slack notify remove
  clonr slack notify remove --force`,
	RunE: runSlackNotifyRemove,
}

// slackNotifyListCmd shows notification config
var slackNotifyListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"status"},
	Short:   "Show notification configuration",
	Long: `Display the current Slack notification configuration.

Examples:
  clonr slack notify list
  clonr slack notify list --json`,
	RunE: runSlackNotifyList,
}

// slackNotifyTestCmd sends test notification
var slackNotifyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test notification",
	Long: `Send a test notification to verify webhook configuration.

Examples:
  clonr slack notify test
  clonr slack notify test --channel "#testing"`,
	RunE: runSlackNotifyTest,
}

// slackNotifyEnableCmd enables notifications
var slackNotifyEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable Slack notifications",
	RunE:  runSlackNotifyEnable,
}

// slackNotifyDisableCmd disables notifications
var slackNotifyDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable Slack notifications",
	RunE:  runSlackNotifyDisable,
}

func init() {
	rootCmd.AddCommand(slackCmd)

	// Add subcommands
	slackCmd.AddCommand(slackAddCmd)
	slackCmd.AddCommand(slackRemoveCmd)
	slackCmd.AddCommand(slackStatusCmd)
	slackCmd.AddCommand(slackAccountsCmd)
	slackCmd.AddCommand(slackChannelsCmd)
	slackCmd.AddCommand(slackMessagesCmd)
	slackCmd.AddCommand(slackSearchCmd)
	slackCmd.AddCommand(slackThreadCmd)
	slackCmd.AddCommand(slackUsersCmd)
	slackCmd.AddCommand(slackNotifyCmd)

	// Notify subcommands
	slackNotifyCmd.AddCommand(slackNotifyAddCmd)
	slackNotifyCmd.AddCommand(slackNotifyRemoveCmd)
	slackNotifyCmd.AddCommand(slackNotifyListCmd)
	slackNotifyCmd.AddCommand(slackNotifyTestCmd)
	slackNotifyCmd.AddCommand(slackNotifyEnableCmd)
	slackNotifyCmd.AddCommand(slackNotifyDisableCmd)

	// Add command flags
	slackAddCmd.Flags().String("client-id", "", "Slack App Client ID")
	slackAddCmd.Flags().String("client-secret", "", "Slack App Client Secret")
	slackAddCmd.Flags().StringP("token", "t", "", "Bot token (skip OAuth flow)")
	slackAddCmd.Flags().StringP("refresh-token", "r", "", "Refresh token for token rotation")
	slackAddCmd.Flags().StringP("channel", "c", "#general", "Default channel")
	slackAddCmd.Flags().Int("port", 8338, "Local callback server port for OAuth")
	slackAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	slackAddCmd.Flags().String("name", "slack", "Name for the Slack account")
	slackAddCmd.Flags().StringP("profile", "p", "", "Profile to add to (default: active)")

	// Remove command flags
	slackRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	slackRemoveCmd.Flags().StringP("profile", "p", "", "Profile to remove from")

	// Status command flags
	slackStatusCmd.Flags().Bool("json", false, "Output as JSON")
	slackStatusCmd.Flags().StringP("profile", "p", "", "Profile to check")

	// Accounts command flags
	slackAccountsCmd.Flags().Bool("json", false, "Output as JSON")

	// Channels flags
	slackChannelsCmd.Flags().StringP("token", "t", "", "Bot token (overrides stored)")
	slackChannelsCmd.Flags().Bool("json", false, "Output as JSON")
	slackChannelsCmd.Flags().Bool("private", false, "Include private channels")
	slackChannelsCmd.Flags().Bool("archived", false, "Include archived channels")
	slackChannelsCmd.Flags().Int("limit", 100, "Maximum channels to return")
	slackChannelsCmd.Flags().StringP("account", "a", "", "Slack account to use")

	// Messages flags
	slackMessagesCmd.Flags().StringP("token", "t", "", "Bot token (overrides stored)")
	slackMessagesCmd.Flags().Bool("json", false, "Output as JSON")
	slackMessagesCmd.Flags().StringP("channel", "c", "", "Channel name or ID (required)")
	slackMessagesCmd.Flags().Int("limit", 20, "Maximum messages to return")
	slackMessagesCmd.Flags().String("since", "", "Show messages since duration (e.g., 24h, 7d)")
	slackMessagesCmd.Flags().String("before", "", "Show messages before timestamp")
	slackMessagesCmd.Flags().StringP("account", "a", "", "Slack account to use")
	_ = slackMessagesCmd.MarkFlagRequired("channel")

	// Search flags
	slackSearchCmd.Flags().StringP("token", "t", "", "Bot token (overrides stored)")
	slackSearchCmd.Flags().Bool("json", false, "Output as JSON")
	slackSearchCmd.Flags().Int("limit", 20, "Maximum results")
	slackSearchCmd.Flags().Int("page", 1, "Page number")
	slackSearchCmd.Flags().String("sort", "timestamp", "Sort by: score or timestamp")
	slackSearchCmd.Flags().String("dir", "desc", "Sort direction: asc or desc")
	slackSearchCmd.Flags().StringP("account", "a", "", "Slack account to use")

	// Thread flags
	slackThreadCmd.Flags().StringP("token", "t", "", "Bot token (overrides stored)")
	slackThreadCmd.Flags().Bool("json", false, "Output as JSON")
	slackThreadCmd.Flags().StringP("channel", "c", "", "Channel name or ID (required)")
	slackThreadCmd.Flags().String("ts", "", "Parent message timestamp (required)")
	slackThreadCmd.Flags().Int("limit", 100, "Maximum replies")
	slackThreadCmd.Flags().StringP("account", "a", "", "Slack account to use")
	_ = slackThreadCmd.MarkFlagRequired("channel")
	_ = slackThreadCmd.MarkFlagRequired("ts")

	// Users flags
	slackUsersCmd.Flags().StringP("token", "t", "", "Bot token (overrides stored)")
	slackUsersCmd.Flags().Bool("json", false, "Output as JSON")
	slackUsersCmd.Flags().Int("limit", 100, "Maximum users to return")
	slackUsersCmd.Flags().StringP("account", "a", "", "Slack account to use")

	// Notify add flags
	slackNotifyAddCmd.Flags().String("webhook", "", "Slack webhook URL")
	slackNotifyAddCmd.Flags().String("channel", "", "Default channel for notifications")

	// Notify remove flags
	slackNotifyRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Notify list flags
	slackNotifyListCmd.Flags().Bool("json", false, "Output as JSON")

	// Notify test flags
	slackNotifyTestCmd.Flags().String("channel", "", "Override default channel for test")
}

// =============================================================================
// Authentication Commands
// =============================================================================

func runSlackAdd(cmd *cobra.Command, _ []string) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	token, _ := cmd.Flags().GetString("token")
	refreshToken, _ := cmd.Flags().GetString("refresh-token")
	channel, _ := cmd.Flags().GetString("channel")
	port, _ := cmd.Flags().GetInt("port")
	scopes, _ := cmd.Flags().GetString("scopes")
	accountName, _ := cmd.Flags().GetString("name")
	profileName, _ := cmd.Flags().GetString("profile")

	// Get profile manager
	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get target profile
	var profile *model.Profile
	if profileName != "" {
		profile, err = pm.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileName, err)
		}
	} else {
		profile, err = pm.GetActiveProfile()
		if err != nil {
			return fmt.Errorf("no active profile; create one first with: clonr profile add <name>")
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Adding Slack to profile %q\n\n", profile.Name)

	// If token provided directly, skip OAuth
	if token != "" {
		return slackAddWithToken(pm, profile, token, refreshToken, channel, accountName)
	}

	// Try environment variables if flags not provided
	if clientID == "" {
		clientID = os.Getenv("SLACK_CLIENT_ID")
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("SLACK_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`Slack Client ID and Client Secret are required for OAuth

Provide them via flags:
  clonr slack add --client-id <id> --client-secret <secret>

Or via environment variables:
  export SLACK_CLIENT_ID=<your-client-id>
  export SLACK_CLIENT_SECRET=<your-client-secret>

Or provide a bot token directly:
  clonr slack add --token xoxb-xxxxxxxxxxxx

To get OAuth credentials:
  1. Go to https://api.slack.com/apps
  2. Create a new app or select an existing one
  3. Go to "Basic Information" > "App Credentials"
  4. Copy the Client ID and Client Secret

Also configure OAuth redirect URL:
  1. Go to "OAuth & Permissions"
  2. Add redirect URL: http://localhost:%d/slack/callback`, port)
	}

	// Run OAuth flow
	config := slack.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Port:         port,
		Scopes:       scopes,
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Starting Slack OAuth flow..."))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "A browser window will open for authorization.")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Waiting for authorization (timeout: 5 minutes)..."))
	_, _ = fmt.Fprintln(os.Stdout, "")

	result, err := slack.RunOAuthFlow(cmd.Context(), config, core.OpenBrowser)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Authorization successful!"))
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Display connection info
	_, _ = fmt.Fprintf(os.Stdout, "  Workspace:  %s (%s)\n", result.Team.Name, result.Team.ID)
	_, _ = fmt.Fprintf(os.Stdout, "  Bot User:   %s\n", result.BotUserID)
	_, _ = fmt.Fprintf(os.Stdout, "  App ID:     %s\n", result.AppID)
	_, _ = fmt.Fprintf(os.Stdout, "  Scopes:     %s\n", result.Scope)
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Create NotifyChannel for Slack
	notifyChannel := &model.NotifyChannel{
		ID:   accountName,
		Type: model.ChannelSlack,
		Name: fmt.Sprintf("Slack - %s", result.Team.Name),
		Config: map[string]string{
			"bot_token":       result.AccessToken,
			"default_channel": channel,
			"workspace_id":    result.Team.ID,
			"workspace_name":  result.Team.Name,
			"bot_user_id":     result.BotUserID,
			"app_id":          result.AppID,
		},
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Slack credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Slack added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Slack commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack channels")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack messages --channel general")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack search \"keyword\"")

	return nil
}

func slackAddWithToken(pm *core.ProfileManager, profile *model.Profile, token, refreshToken, channel, accountName string) error {
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Validating bot token..."))

	// Create a temporary client to validate the token
	client := slack.NewClient(token, slack.ClientOptions{})

	authResult, err := client.AuthTest(context.Background())
	if err != nil {
		return fmt.Errorf("invalid bot token: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Token validated!"))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "  Workspace:  %s (%s)\n", authResult.Team, authResult.TeamID)
	_, _ = fmt.Fprintf(os.Stdout, "  Bot User:   %s (%s)\n", authResult.User, authResult.UserID)
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Create NotifyChannel for Slack
	config := map[string]string{
		"bot_token":       token,
		"default_channel": channel,
		"workspace_id":    authResult.TeamID,
		"workspace_name":  authResult.Team,
		"bot_user_id":     authResult.UserID,
	}

	// Add refresh token if provided
	if refreshToken != "" {
		config["refresh_token"] = refreshToken
	}

	notifyChannel := &model.NotifyChannel{
		ID:        accountName,
		Type:      model.ChannelSlack,
		Name:      fmt.Sprintf("Slack - %s", authResult.Team),
		Config:    config,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Slack credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Slack added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Slack commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack channels")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack messages --channel general")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack search \"keyword\"")

	return nil
}

func runSlackRemove(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	profileName, _ := cmd.Flags().GetString("profile")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get target profile
	var profile *model.Profile
	if profileName != "" {
		profile, err = pm.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileName, err)
		}
	} else {
		profile, err = pm.GetActiveProfile()
		if err != nil {
			return fmt.Errorf("no active profile")
		}
	}

	// Get account name from args or find by type
	var channelID string
	if len(args) > 0 {
		channelID = args[0]
	} else {
		// Find Slack channel by type
		channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
		if err != nil {
			return fmt.Errorf("failed to check Slack integration: %w", err)
		}

		if channel == nil {
			return fmt.Errorf("no Slack integration found in profile %q", profile.Name)
		}

		channelID = channel.ID
	}

	// Confirm removal
	if !force {
		if !promptConfirm(fmt.Sprintf("Remove Slack integration %q from profile %q? [y/N] ", channelID, profile.Name)) {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := pm.RemoveNotifyChannel(profile.Name, channelID); err != nil {
		return fmt.Errorf("failed to remove Slack integration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack integration removed."))

	return nil
}

func runSlackStatus(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	profileName, _ := cmd.Flags().GetString("profile")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get target profile
	var profile *model.Profile
	if profileName != "" {
		profile, err = pm.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("profile %q not found: %w", profileName, err)
		}
	} else {
		profile, err = pm.GetActiveProfile()
		if err != nil {
			return fmt.Errorf("no active profile")
		}
	}

	// Get channel by name or by type
	var channel *model.NotifyChannel
	if len(args) > 0 {
		channel, err = pm.GetNotifyChannel(profile.Name, args[0])
		if err != nil {
			return fmt.Errorf("failed to get Slack account %q: %w", args[0], err)
		}
	} else {
		channel, err = pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
		if err != nil {
			return fmt.Errorf("failed to check Slack integration: %w", err)
		}
	}

	if channel == nil {
		if jsonOutput {
			_, _ = fmt.Fprintln(os.Stdout, `{"configured": false}`)
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "No Slack integration configured for profile %q\n", profile.Name)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Add Slack with:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr slack add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr slack add --token xoxb-xxxx")

		return nil
	}

	// Decrypt config for display
	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return fmt.Errorf("failed to decrypt Slack config: %w", err)
	}

	if jsonOutput {
		type slackStatus struct {
			Configured     bool   `json:"configured"`
			Profile        string `json:"profile"`
			AccountName    string `json:"account_name"`
			WorkspaceID    string `json:"workspace_id,omitempty"`
			WorkspaceName  string `json:"workspace_name,omitempty"`
			BotUserID      string `json:"bot_user_id,omitempty"`
			DefaultChannel string `json:"default_channel,omitempty"`
			Enabled        bool   `json:"enabled"`
			CreatedAt      string `json:"created_at"`
		}

		status := slackStatus{
			Configured:     true,
			Profile:        profile.Name,
			AccountName:    channel.ID,
			WorkspaceID:    config["workspace_id"],
			WorkspaceName:  config["workspace_name"],
			BotUserID:      config["bot_user_id"],
			DefaultChannel: config["default_channel"],
			Enabled:        channel.Enabled,
			CreatedAt:      channel.CreatedAt.Format(time.RFC3339),
		}

		return outputJSON(status)
	}

	// Display status
	printBoxHeader("SLACK INTEGRATION")
	printBoxLine("Profile", profile.Name)
	printBoxLine("Account", channel.ID)
	printBoxLine("Workspace", fmt.Sprintf("%s (%s)", config["workspace_name"], config["workspace_id"]))

	if config["bot_user_id"] != "" {
		printBoxLine("Bot User", config["bot_user_id"])
	}

	if config["app_id"] != "" {
		printBoxLine("App ID", config["app_id"])
	}

	printBoxLine("Default Channel", config["default_channel"])
	printBoxLine("Enabled", fmt.Sprintf("%t", channel.Enabled))
	printBoxLine("Added", channel.CreatedAt.Format("2006-01-02 15:04:05"))
	printBoxFooter()

	// Test connection
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Testing connection..."))

	token := config["bot_token"]
	if token != "" {
		client := slack.NewClient(token, slack.ClientOptions{})

		if _, err := client.AuthTest(context.Background()); err != nil {
			_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Connection failed: %v", err)))
		} else {
			_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Connection OK"))
		}
	}

	return nil
}

func runSlackAccounts(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get all profiles
	profiles, err := pm.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to get profiles: %w", err)
	}

	type slackAccount struct {
		Profile       string `json:"profile"`
		AccountName   string `json:"account_name"`
		WorkspaceName string `json:"workspace_name"`
		Enabled       bool   `json:"enabled"`
	}

	var accounts []slackAccount

	for _, profile := range profiles {
		// Get Slack channels from profile's NotifyChannels
		for _, ch := range profile.NotifyChannels {
			if ch.Type != model.ChannelSlack {
				continue
			}

			config, _ := pm.DecryptChannelConfig(profile.Name, &ch)

			workspaceName := ""
			if config != nil {
				workspaceName = config["workspace_name"]
			}

			accounts = append(accounts, slackAccount{
				Profile:       profile.Name,
				AccountName:   ch.ID,
				WorkspaceName: workspaceName,
				Enabled:       ch.Enabled,
			})
		}
	}

	if jsonOutput {
		return outputJSON(accounts)
	}

	if len(accounts) == 0 {
		printEmptyResult("Slack accounts", "clonr slack add")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSlack Accounts (%d)\n\n", len(accounts))
	_, _ = fmt.Fprintf(os.Stdout, "  %-15s │ %-15s │ %-25s │ %s\n", "Profile", "Account", "Workspace", "Status")
	_, _ = fmt.Fprintln(os.Stdout, "  ────────────────┼─────────────────┼───────────────────────────┼─────────")

	for _, acc := range accounts {
		status := "enabled"
		if !acc.Enabled {
			status = "disabled"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %-15s │ %-15s │ %-25s │ %s\n",
			truncateString(acc.Profile, 15),
			truncateString(acc.AccountName, 15),
			truncateString(acc.WorkspaceName, 25),
			status)
	}

	return nil
}

// =============================================================================
// Operation Commands
// =============================================================================

// slackGetClient gets a Slack client from token flag or stored credentials
func slackGetClient(cmd *cobra.Command) (*slack.Client, error) {
	tokenFlag, _ := cmd.Flags().GetString("token")
	accountName, _ := cmd.Flags().GetString("account")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Try token flag first
	if tokenFlag != "" {
		return slack.NewClient(tokenFlag, slack.ClientOptions{}), nil
	}

	// Try stored token
	token, _, err := slack.ResolveSlackToken("")
	if err == nil && token != "" {
		var logger *slog.Logger
		if jsonOutput {
			logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		} else {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		}

		return slack.NewClient(token, slack.ClientOptions{Logger: logger}), nil
	}

	// Try profile-based token
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("no Slack token found and could not connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile; add Slack with: clonr slack add")
	}

	var channel *model.NotifyChannel
	if accountName != "" {
		channel, err = pm.GetNotifyChannel(profile.Name, accountName)
	} else {
		channel, err = pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
	}

	if err != nil || channel == nil {
		return nil, fmt.Errorf("no Slack integration found; add with: clonr slack add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Slack config: %w", err)
	}

	token = config["bot_token"]
	if token == "" {
		return nil, fmt.Errorf("no bot token found in Slack config")
	}

	var logger *slog.Logger
	if jsonOutput {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	return slack.NewClient(token, slack.ClientOptions{Logger: logger}), nil
}

func runSlackChannels(cmd *cobra.Command, _ []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	includePrivate, _ := cmd.Flags().GetBool("private")
	includeArchived, _ := cmd.Flags().GetBool("archived")
	limit, _ := cmd.Flags().GetInt("limit")

	client, err := slackGetClient(cmd)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching channels..."))
	}

	// Build channel types
	types := "public_channel"
	if includePrivate {
		types = "public_channel,private_channel"
	}

	// List channels
	result, err := client.ListChannels(cmd.Context(), slack.ListChannelsOptions{
		Types:           types,
		ExcludeArchived: !includeArchived,
		Limit:           limit,
	})
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Channels)
	}

	if len(result.Channels) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No channels found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSlack Channels (%d)\n\n", len(result.Channels))
	_, _ = fmt.Fprintf(os.Stdout, "  %-12s │ %-25s │ %-8s │ %s\n", "ID", "Name", "Members", "Topic")
	_, _ = fmt.Fprintln(os.Stdout, "  ─────────────┼───────────────────────────┼──────────┼─────────────────────")

	for _, ch := range result.Channels {
		name := ch.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		topic := ch.Topic.Value
		if len(topic) > 40 {
			topic = topic[:37] + "..."
		}

		prefix := "#"
		if ch.IsPrivate {
			prefix = "🔒"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %-12s │ %s%-24s │ %-8d │ %s\n",
			ch.ID, prefix, name, ch.NumMembers, topic)
	}

	return nil
}

func runSlackMessages(cmd *cobra.Command, _ []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	channel, _ := cmd.Flags().GetString("channel")
	limit, _ := cmd.Flags().GetInt("limit")
	since, _ := cmd.Flags().GetString("since")
	before, _ := cmd.Flags().GetString("before")

	client, err := slackGetClient(cmd)
	if err != nil {
		return err
	}

	// Resolve channel ID if name given
	channelID, err := slackResolveChannelID(cmd.Context(), client, channel, outputJSON)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Fetching messages from %s...\n"), channel)
	}

	// Build options
	opts := slack.GetChannelHistoryOptions{
		Channel: channelID,
		Limit:   limit,
	}

	if before != "" {
		opts.Latest = before
	}

	if since != "" {
		duration, parseErr := slackParseDuration(since)
		if parseErr != nil {
			return fmt.Errorf("invalid duration: %w", parseErr)
		}

		oldest := time.Now().Add(-duration)
		opts.Oldest = slack.FormatTimestamp(oldest)
	}

	// Get messages
	result, err := client.GetChannelHistory(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Fetch user info for display names
	userCache := make(map[string]string)

	if !outputJSON {
		for _, msg := range result.Messages {
			if msg.User != "" {
				if _, ok := userCache[msg.User]; !ok {
					if user, userErr := client.GetUser(cmd.Context(), msg.User); userErr == nil {
						userCache[msg.User] = user.Profile.DisplayName
						if userCache[msg.User] == "" {
							userCache[msg.User] = user.Name
						}
					} else {
						userCache[msg.User] = msg.User
					}
				}
			}
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Messages)
	}

	if len(result.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nMessages from #%s (%d)\n\n", channel, len(result.Messages))

	// Messages are returned newest first, reverse for chronological display
	for i := len(result.Messages) - 1; i >= 0; i-- {
		msg := result.Messages[i]

		ts, _ := slack.ParseTimestamp(msg.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		userName := userCache[msg.User]
		if userName == "" {
			userName = msg.User
			if msg.BotProfile != nil {
				userName = msg.BotProfile.Name + " (bot)"
			}
		}

		// Print message header
		_, _ = fmt.Fprintf(os.Stdout, "┌─ %s @ %s\n", okStyle.Render(userName), dimStyle.Render(timeStr))

		// Print message text (with indentation)
		text := msg.Text
		if len(text) > 500 {
			text = text[:497] + "..."
		}

		for line := range strings.SplitSeq(text, "\n") {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
		}

		// Print reactions
		if len(msg.Reactions) > 0 {
			var reactions []string
			for _, r := range msg.Reactions {
				reactions = append(reactions, fmt.Sprintf(":%s: %d", r.Name, r.Count))
			}

			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", dimStyle.Render(strings.Join(reactions, "  ")))
		}

		// Print thread info
		if msg.ReplyCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", dimStyle.Render(fmt.Sprintf("💬 %d replies", msg.ReplyCount)))
		}

		// Print files
		if len(msg.Files) > 0 {
			for _, f := range msg.Files {
				_, _ = fmt.Fprintf(os.Stdout, "│  📎 %s (%s)\n", f.Name, f.PrettyType)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "└─ ts: %s\n\n", dimStyle.Render(msg.Timestamp))
	}

	if result.HasMore {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("More messages available. Use --before flag with the oldest timestamp to paginate."))
	}

	return nil
}

func runSlackSearch(cmd *cobra.Command, args []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	sort, _ := cmd.Flags().GetString("sort")
	dir, _ := cmd.Flags().GetString("dir")

	query := strings.Join(args, " ")

	client, err := slackGetClient(cmd)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Searching for: %s\n"), query)
	}

	// Search
	result, err := client.SearchMessages(cmd.Context(), slack.SearchMessagesOptions{
		Query: query,
		Sort:  sort,
		Dir:   dir,
		Count: limit,
		Page:  page,
	})
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	if result.Total == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No results found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSearch Results for \"%s\" (%d of %d)\n\n", query, len(result.Matches), result.Total)

	for _, match := range result.Matches {
		ts, _ := slack.ParseTimestamp(match.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		text := match.Text
		if len(text) > 200 {
			text = text[:197] + "..."
		}

		_, _ = fmt.Fprintf(os.Stdout, "┌─ #%s @ %s\n", okStyle.Render(channelName), dimStyle.Render(timeStr))
		_, _ = fmt.Fprintf(os.Stdout, "│  by: %s\n", match.Username)

		for line := range strings.SplitSeq(text, "\n") {
			if line != "" {
				_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "└─ %s\n\n", dimStyle.Render(match.Permalink))
	}

	if result.Paging.Pages > page {
		_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Page %d of %d. Use --page to see more.\n"), page, result.Paging.Pages)
	}

	return nil
}

func runSlackThread(cmd *cobra.Command, _ []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	channel, _ := cmd.Flags().GetString("channel")
	threadTS, _ := cmd.Flags().GetString("ts")
	limit, _ := cmd.Flags().GetInt("limit")

	client, err := slackGetClient(cmd)
	if err != nil {
		return err
	}

	// Resolve channel ID if name given
	channelID, err := slackResolveChannelID(cmd.Context(), client, channel, outputJSON)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching thread replies..."))
	}

	// Get thread
	result, err := client.GetThreadReplies(cmd.Context(), slack.GetThreadRepliesOptions{
		Channel:  channelID,
		ThreadTS: threadTS,
		Limit:    limit,
	})
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	// Fetch user info for display names
	userCache := make(map[string]string)

	if !outputJSON {
		for _, msg := range result.Messages {
			if msg.User != "" {
				if _, ok := userCache[msg.User]; !ok {
					if user, userErr := client.GetUser(cmd.Context(), msg.User); userErr == nil {
						userCache[msg.User] = user.Profile.DisplayName
						if userCache[msg.User] == "" {
							userCache[msg.User] = user.Name
						}
					} else {
						userCache[msg.User] = msg.User
					}
				}
			}
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Messages)
	}

	if len(result.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No replies found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nThread Replies (%d messages)\n\n", len(result.Messages))

	for i, msg := range result.Messages {
		ts, _ := slack.ParseTimestamp(msg.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		userName := userCache[msg.User]
		if userName == "" {
			userName = msg.User
		}

		prefix := "├─"
		if i == 0 {
			prefix = "┌─ (original)"
		} else if i == len(result.Messages)-1 {
			prefix = "└─"
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s %s @ %s\n", prefix, okStyle.Render(userName), dimStyle.Render(timeStr))

		text := msg.Text
		if len(text) > 500 {
			text = text[:497] + "..."
		}

		for line := range strings.SplitSeq(text, "\n") {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runSlackUsers(cmd *cobra.Command, _ []string) error {
	outputJSON, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")

	client, err := slackGetClient(cmd)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching users..."))
	}

	// List users
	result, err := client.ListUsers(cmd.Context(), slack.ListUsersOptions{
		Limit: limit,
	})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Filter out bots and deleted users for display
	var activeUsers []slack.User

	for _, u := range result.Users {
		if !u.Deleted && !u.IsBot {
			activeUsers = append(activeUsers, u)
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Users)
	}

	if len(activeUsers) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No users found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSlack Users (%d active)\n\n", len(activeUsers))
	_, _ = fmt.Fprintf(os.Stdout, "  %-15s │ %-20s │ %-25s │ %s\n", "Username", "Display Name", "Real Name", "Status")
	_, _ = fmt.Fprintln(os.Stdout, "  ────────────────┼──────────────────────┼───────────────────────────┼─────────────")

	for _, u := range activeUsers {
		displayName := u.Profile.DisplayName
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}

		realName := u.Profile.RealName
		if len(realName) > 25 {
			realName = realName[:22] + "..."
		}

		status := u.Profile.StatusEmoji + " " + u.Profile.StatusText
		if len(status) > 30 {
			status = status[:27] + "..."
		}

		adminBadge := ""
		if u.IsAdmin {
			adminBadge = " (admin)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  @%-14s │ %-20s │ %-25s │ %s%s\n",
			u.Name, displayName, realName, status, adminBadge)
	}

	return nil
}

// =============================================================================
// Notification Commands (Webhooks)
// =============================================================================

func runSlackNotifyAdd(cmd *cobra.Command, _ []string) error {
	webhook, _ := cmd.Flags().GetString("webhook")
	channel, _ := cmd.Flags().GetString("channel")

	if webhook == "" {
		return fmt.Errorf("--webhook is required")
	}

	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	if err := manager.AddWebhook(webhook, channel); err != nil {
		return fmt.Errorf("failed to add webhook: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack webhook added successfully!"))
	if channel != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Default channel: %s\n", channel)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nTest with: clonr slack notify test")

	return nil
}

func runSlackNotifyRemove(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")

	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		_, _ = fmt.Fprintln(os.Stdout, "Slack notifications not configured.")
		return nil
	}

	if !force {
		if !promptConfirm("Remove Slack webhook? [y/N]: ") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := manager.Remove(); err != nil {
		return fmt.Errorf("failed to remove webhook: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack webhook removed."))

	return nil
}

func runSlackNotifyList(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		if jsonOutput {
			_, _ = fmt.Fprintln(os.Stdout, "null")
			return nil
		}

		printEmptyResult("Slack notifications", "clonr slack notify add --webhook <url>")

		return nil
	}

	// Determine integration type
	integrationType := "webhook"
	if len(config.EncryptedBotToken) > 0 && config.BotEnabled {
		integrationType = "bot"
	}

	// JSON output
	if jsonOutput {
		type notifyOutput struct {
			Enabled        bool   `json:"enabled"`
			Type           string `json:"type"`
			DefaultChannel string `json:"default_channel,omitempty"`
		}

		output := notifyOutput{
			Enabled:        config.Enabled,
			Type:           integrationType,
			DefaultChannel: config.DefaultChannel,
		}

		return outputJSON(output)
	}

	// Text output
	printBoxHeader("SLACK NOTIFICATIONS")
	printBoxLine("Status", slackFormatEnabled(config.Enabled))
	printBoxLine("Type", integrationType)

	if config.DefaultChannel != "" {
		printBoxLine("Channel", config.DefaultChannel)
	}

	printBoxFooter()

	return nil
}

func runSlackNotifyTest(cmd *cobra.Command, _ []string) error {
	channel, _ := cmd.Flags().GetString("channel")

	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Sending test notification..."))

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	if err := manager.Test(ctx, channel); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Test notification sent successfully!"))
	if channel != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Check channel: %s\n", channel)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Check your configured default channel.")
	}

	return nil
}

func runSlackNotifyEnable(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("Slack notifications not configured\nSet up with: clonr slack notify add --webhook <url>")
	}

	if config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Slack notifications are already enabled.")
		return nil
	}

	if err := manager.Enable(); err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack notifications enabled."))

	return nil
}

func runSlackNotifyDisable(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("Slack notifications not configured")
	}

	if !config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Slack notifications are already disabled.")
		return nil
	}

	if err := manager.Disable(); err != nil {
		return fmt.Errorf("failed to disable notifications: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack notifications disabled."))
	_, _ = fmt.Fprintln(os.Stdout, "Re-enable with: clonr slack notify enable")

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// slackResolveChannelID resolves a channel name to its ID.
func slackResolveChannelID(ctx context.Context, client *slack.Client, channel string, quiet bool) (string, error) {
	// If it looks like an ID (starts with C, G, or D), use it directly
	if len(channel) > 0 && (channel[0] == 'C' || channel[0] == 'G' || channel[0] == 'D') {
		return channel, nil
	}

	// Strip # prefix if present
	channel = strings.TrimPrefix(channel, "#")

	if !quiet {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Looking up channel #%s...\n"), channel)
	}

	// List channels to find the ID
	result, err := client.ListChannels(ctx, slack.ListChannelsOptions{
		Types: "public_channel,private_channel",
		Limit: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range result.Channels {
		if ch.Name == channel {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel #%s not found", channel)
}

// slackParseDuration parses duration strings like "24h", "7d", "2w".
func slackParseDuration(s string) (time.Duration, error) {
	// Check for days/weeks suffix
	if num, found := strings.CutSuffix(s, "d"); found {
		days, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}

		return time.Duration(days) * 24 * time.Hour, nil
	}

	if num, found := strings.CutSuffix(s, "w"); found {
		weeks, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}

		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// Standard Go duration
	return time.ParseDuration(s)
}

func slackFormatEnabled(enabled bool) string {
	if enabled {
		return "enabled"
	}

	return "disabled"
}
