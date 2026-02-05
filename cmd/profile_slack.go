package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
	"github.com/spf13/cobra"
)

func init() {
	profileCmd.AddCommand(profileSlackCmd)
	profileSlackCmd.AddCommand(profileSlackAddCmd)
	profileSlackCmd.AddCommand(profileSlackRemoveCmd)
	profileSlackCmd.AddCommand(profileSlackStatusCmd)

	// Add flags
	profileSlackAddCmd.Flags().String("client-id", "", "Slack App Client ID")
	profileSlackAddCmd.Flags().String("client-secret", "", "Slack App Client Secret")
	profileSlackAddCmd.Flags().StringP("token", "t", "", "Bot token (skip OAuth flow)")
	profileSlackAddCmd.Flags().StringP("channel", "c", "#general", "Default channel for notifications")
	profileSlackAddCmd.Flags().Int("port", 8338, "Local callback server port for OAuth")
	profileSlackAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	profileSlackAddCmd.Flags().String("name", "slack", "Name for the Slack channel configuration")

	profileSlackRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	profileSlackStatusCmd.Flags().Bool("json", false, "Output as JSON")
}

var profileSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Manage Slack integration for the active profile",
	Long: `Manage Slack integration for the active profile.

Slack credentials are stored securely using the profile's encryption
(TPM-backed when available).

Available Commands:
  add          Add Slack integration via OAuth or bot token
  remove       Remove Slack integration from profile
  status       Show Slack integration status

Examples:
  clonr profile slack add --client-id <id> --client-secret <secret>
  clonr profile slack add --token xoxb-xxxx
  clonr profile slack status
  clonr profile slack remove`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var profileSlackAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Slack integration to the active profile",
	Long: `Add Slack integration to the active profile.

By default, this starts an OAuth flow where you:
1. A browser window opens to Slack authorization
2. Authorize clonr for your workspace
3. The bot token is automatically saved to your profile

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
  clonr profile slack add --client-id <id> --client-secret <secret>
  clonr profile slack add --token xoxb-xxxxxxxxxxxx
  SLACK_CLIENT_ID=xxx SLACK_CLIENT_SECRET=yyy clonr profile slack add`,
	RunE: runProfileSlackAdd,
}

var profileSlackRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Slack integration from the active profile",
	Long: `Remove Slack integration from the active profile.

This removes the stored Slack credentials from the profile.
Use --force to skip confirmation.

Examples:
  clonr profile slack remove
  clonr profile slack remove --force`,
	RunE: runProfileSlackRemove,
}

var profileSlackStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Slack integration status for the active profile",
	Long: `Show Slack integration status for the active profile.

Displays workspace information, connection status, and configured channels.

Examples:
  clonr profile slack status
  clonr profile slack status --json`,
	RunE: runProfileSlackStatus,
}

func runProfileSlackAdd(cmd *cobra.Command, _ []string) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	token, _ := cmd.Flags().GetString("token")
	channel, _ := cmd.Flags().GetString("channel")
	port, _ := cmd.Flags().GetInt("port")
	scopes, _ := cmd.Flags().GetString("scopes")
	channelName, _ := cmd.Flags().GetString("name")

	// Get profile manager and active profile
	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile; create one first with: clonr profile add <name>")
	}

	_, _ = fmt.Fprintf(os.Stdout, "Adding Slack to profile %q\n\n", profile.Name)

	// If token provided directly, skip OAuth
	if token != "" {
		return addSlackWithToken(pm, profile, token, channel, channelName)
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
  clonr profile slack add --client-id <id> --client-secret <secret>

Or via environment variables:
  export SLACK_CLIENT_ID=<your-client-id>
  export SLACK_CLIENT_SECRET=<your-client-secret>

Or provide a bot token directly:
  clonr profile slack add --token xoxb-xxxxxxxxxxxx

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
		ID:   channelName,
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

	// Save to profile (AddNotifyChannel encrypts sensitive fields)
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Slack credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Slack added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Slack commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack channels")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack messages --channel general")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack search \"keyword\"")

	return nil
}

func addSlackWithToken(pm *core.ProfileManager, profile *model.Profile, token, channel, channelName string) error {
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
	notifyChannel := &model.NotifyChannel{
		ID:   channelName,
		Type: model.ChannelSlack,
		Name: fmt.Sprintf("Slack - %s", authResult.Team),
		Config: map[string]string{
			"bot_token":       token,
			"default_channel": channel,
			"workspace_id":    authResult.TeamID,
			"workspace_name":  authResult.Team,
			"bot_user_id":     authResult.UserID,
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
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack channels")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack messages --channel general")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack search \"keyword\"")

	return nil
}

func runProfileSlackRemove(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Check if Slack is configured
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
	if err != nil {
		return fmt.Errorf("failed to check Slack integration: %w", err)
	}

	if channel == nil {
		return fmt.Errorf("no Slack integration found in profile %q", profile.Name)
	}

	// Confirm removal
	if !force {
		_, _ = fmt.Fprintf(os.Stdout, "Remove Slack integration from profile %q? [y/N] ", profile.Name)

		if !promptConfirm("") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := pm.RemoveNotifyChannel(profile.Name, channel.ID); err != nil {
		return fmt.Errorf("failed to remove Slack integration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Slack integration removed."))

	return nil
}

func runProfileSlackStatus(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Get Slack channel
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
	if err != nil {
		return fmt.Errorf("failed to check Slack integration: %w", err)
	}

	if channel == nil {
		if jsonOutput {
			_, _ = fmt.Fprintln(os.Stdout, `{"configured": false}`)
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "No Slack integration configured for profile %q\n", profile.Name)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Add Slack with:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile slack add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile slack add --token xoxb-xxxx")

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
