package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/gmail"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	profileCmd.AddCommand(profileGmailCmd)
	profileGmailCmd.AddCommand(profileGmailAddCmd)
	profileGmailCmd.AddCommand(profileGmailRemoveCmd)
	profileGmailCmd.AddCommand(profileGmailStatusCmd)

	// Add flags
	profileGmailAddCmd.Flags().String("client-id", "", "Google OAuth Client ID")
	profileGmailAddCmd.Flags().String("client-secret", "", "Google OAuth Client Secret")
	profileGmailAddCmd.Flags().StringP("token", "t", "", "Access token (skip OAuth flow)")
	profileGmailAddCmd.Flags().StringP("refresh-token", "r", "", "Refresh token for token rotation")
	profileGmailAddCmd.Flags().Int("port", 8339, "Local callback server port for OAuth")
	profileGmailAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	profileGmailAddCmd.Flags().String("name", "gmail", "Name for the Gmail channel configuration")

	profileGmailRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	profileGmailStatusCmd.Flags().Bool("json", false, "Output as JSON")
}

var profileGmailCmd = &cobra.Command{
	Use:   "gmail",
	Short: "Manage Gmail integration for the active profile",
	Long: `Manage Gmail integration for the active profile.

Gmail credentials are stored securely using the profile's encryption
(TPM-backed when available).

Available Commands:
  add          Add Gmail integration via OAuth or access token
  remove       Remove Gmail integration from profile
  status       Show Gmail integration status

Examples:
  clonr profile gmail add --client-id <id> --client-secret <secret>
  clonr profile gmail add --token <access_token>
  clonr profile gmail status
  clonr profile gmail remove`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var profileGmailAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Gmail integration to the active profile",
	Long: `Add Gmail integration to the active profile.

By default, this starts an OAuth flow where you:
1. A browser window opens to Google authorization
2. Authorize clonr for your Gmail account
3. The access token is automatically saved to your profile

Alternatively, provide an access token directly with --token to skip OAuth.

Prerequisites for OAuth:
  1. Go to https://console.cloud.google.com/apis/credentials
  2. Create an OAuth 2.0 Client ID (Desktop or Web application)
  3. Add redirect URI: http://localhost:8339/gmail/callback
  4. Enable Gmail API in your Google Cloud project
  5. Get Client ID and Client Secret

Required Scopes:
  - https://www.googleapis.com/auth/gmail.readonly
  - https://www.googleapis.com/auth/userinfo.email

Examples:
  clonr profile gmail add --client-id <id> --client-secret <secret>
  clonr profile gmail add --token <access_token>
  GOOGLE_CLIENT_ID=xxx GOOGLE_CLIENT_SECRET=yyy clonr profile gmail add`,
	RunE: runProfileGmailAdd,
}

var profileGmailRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Gmail integration from the active profile",
	Long: `Remove Gmail integration from the active profile.

This removes the stored Gmail credentials from the profile.
Use --force to skip confirmation.

Examples:
  clonr profile gmail remove
  clonr profile gmail remove --force`,
	RunE: runProfileGmailRemove,
}

var profileGmailStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gmail integration status for the active profile",
	Long: `Show Gmail integration status for the active profile.

Displays account information, connection status, and token validity.

Examples:
  clonr profile gmail status
  clonr profile gmail status --json`,
	RunE: runProfileGmailStatus,
}

func runProfileGmailAdd(cmd *cobra.Command, _ []string) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	token, _ := cmd.Flags().GetString("token")
	refreshToken, _ := cmd.Flags().GetString("refresh-token")
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

	_, _ = fmt.Fprintf(os.Stdout, "Adding Gmail to profile %q\n\n", profile.Name)

	// If token provided directly, skip OAuth
	if token != "" {
		return addGmailWithToken(pm, profile, token, refreshToken, channelName)
	}

	// Try environment variables if flags not provided
	if clientID == "" {
		clientID = os.Getenv("GOOGLE_CLIENT_ID")
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`google Client ID and Client Secret are required for OAuth

Provide them via flags:
  clonr profile gmail add --client-id <id> --client-secret <secret>

Or via environment variables:
  export GOOGLE_CLIENT_ID=<your-client-id>
  export GOOGLE_CLIENT_SECRET=<your-client-secret>

Or provide an access token directly:
  clonr profile gmail add --token <access_token>

To get OAuth credentials:
  1. Go to https://console.cloud.google.com/apis/credentials
  2. Create an OAuth 2.0 Client ID
  3. Copy the Client ID and Client Secret

Also configure OAuth consent screen and enable Gmail API:
  1. Go to "OAuth consent screen" and configure
  2. Go to "Enabled APIs & services" and enable Gmail API
  3. Add redirect URI: http://localhost:%d/gmail/callback`, port)
	}

	// Parse scopes if provided
	var scopeList []string
	if scopes != "" {
		scopeList = parseScopes(scopes)
	}

	// Run OAuth flow
	config := gmail.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Port:         port,
		Scopes:       scopeList,
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Starting Gmail OAuth flow..."))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "A browser window will open for authorization.")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Waiting for authorization (timeout: 5 minutes)..."))
	_, _ = fmt.Fprintln(os.Stdout, "")

	result, err := gmail.RunOAuthFlow(cmd.Context(), config, core.OpenBrowser)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Authorization successful!"))
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Validate token and get user info
	tokenInfo, err := gmail.ValidateToken(cmd.Context(), result.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}

	// Display connection info
	_, _ = fmt.Fprintf(os.Stdout, "  Email:    %s\n", tokenInfo.Email)
	_, _ = fmt.Fprintf(os.Stdout, "  Verified: %t\n", tokenInfo.EmailVerified)
	_, _ = fmt.Fprintf(os.Stdout, "  Scopes:   %s\n", result.Scope)
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Create NotifyChannel for Gmail
	notifyChannel := &model.NotifyChannel{
		ID:   channelName,
		Type: model.ChannelGmail,
		Name: fmt.Sprintf("Gmail - %s", tokenInfo.Email),
		Config: map[string]string{
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
			"email":         tokenInfo.Email,
			"client_id":     clientID,
			"client_secret": clientSecret,
		},
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile (AddNotifyChannel encrypts sensitive fields)
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Gmail credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Gmail added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Gmail commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail profile")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail search \"keyword\"")

	return nil
}

func addGmailWithToken(pm *core.ProfileManager, profile *model.Profile, token, refreshToken, channelName string) error {
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Validating access token..."))

	// Validate token and get user info
	tokenInfo, err := gmail.ValidateToken(context.Background(), token)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Token validated!"))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "  Email:    %s\n", tokenInfo.Email)
	_, _ = fmt.Fprintf(os.Stdout, "  Verified: %t\n", tokenInfo.EmailVerified)
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Create NotifyChannel for Gmail
	config := map[string]string{
		"access_token": token,
		"email":        tokenInfo.Email,
	}

	// Add refresh token if provided
	if refreshToken != "" {
		config["refresh_token"] = refreshToken
	}

	notifyChannel := &model.NotifyChannel{
		ID:        channelName,
		Type:      model.ChannelGmail,
		Name:      fmt.Sprintf("Gmail - %s", tokenInfo.Email),
		Config:    config,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Gmail credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Gmail added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Gmail commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail profile")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm gmail search \"keyword\"")

	return nil
}

func runProfileGmailRemove(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Check if Gmail is configured
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return fmt.Errorf("failed to check Gmail integration: %w", err)
	}

	if channel == nil {
		return fmt.Errorf("no Gmail integration found in profile %q", profile.Name)
	}

	// Confirm removal
	if !force {
		_, _ = fmt.Fprintf(os.Stdout, "Remove Gmail integration from profile %q? [y/N] ", profile.Name)

		if !promptConfirm("") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := pm.RemoveNotifyChannel(profile.Name, channel.ID); err != nil {
		return fmt.Errorf("failed to remove Gmail integration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Gmail integration removed."))

	return nil
}

func runProfileGmailStatus(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Get Gmail channel
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return fmt.Errorf("failed to check Gmail integration: %w", err)
	}

	if channel == nil {
		if jsonOutput {
			_, _ = fmt.Fprintln(os.Stdout, `{"configured": false}`)
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "No Gmail integration configured for profile %q\n", profile.Name)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Add Gmail with:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile gmail add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile gmail add --token <access_token>")

		return nil
	}

	// Decrypt config for display
	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return fmt.Errorf("failed to decrypt Gmail config: %w", err)
	}

	if jsonOutput {
		type gmailStatus struct {
			Configured bool   `json:"configured"`
			Profile    string `json:"profile"`
			Email      string `json:"email,omitempty"`
			Enabled    bool   `json:"enabled"`
			CreatedAt  string `json:"created_at"`
		}

		status := gmailStatus{
			Configured: true,
			Profile:    profile.Name,
			Email:      config["email"],
			Enabled:    channel.Enabled,
			CreatedAt:  channel.CreatedAt.Format(time.RFC3339),
		}

		return outputJSON(status)
	}

	// Display status
	printBoxHeader("GMAIL INTEGRATION")
	printBoxLine("Profile", profile.Name)
	printBoxLine("Email", config["email"])
	printBoxLine("Enabled", fmt.Sprintf("%t", channel.Enabled))
	printBoxLine("Added", channel.CreatedAt.Format("2006-01-02 15:04:05"))
	printBoxFooter()

	// Test connection
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Testing connection..."))

	accessToken := config["access_token"]
	if accessToken != "" {
		client := gmail.NewClient(accessToken, gmail.ClientOptions{})

		gmailProfile, err := client.GetProfile(context.Background())
		if err != nil {
			// Try to refresh token if available
			refreshToken := config["refresh_token"]
			clientID := config["client_id"]
			clientSecret := config["client_secret"]

			if refreshToken != "" && clientID != "" && clientSecret != "" {
				_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Token expired, attempting refresh..."))

				newToken, refreshErr := gmail.RefreshAccessToken(context.Background(), clientID, clientSecret, refreshToken)
				if refreshErr != nil {
					_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Connection failed: %v", err)))
					_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Token refresh failed: %v", refreshErr)))
				} else {
					// Update stored token
					channel.Config["access_token"] = newToken.AccessToken
					if newToken.RefreshToken != "" {
						channel.Config["refresh_token"] = newToken.RefreshToken
					}

					if saveErr := pm.AddNotifyChannel(profile.Name, channel); saveErr != nil {
						_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Failed to save refreshed token: %v", saveErr)))
					} else {
						_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Token refreshed and saved"))
					}
				}
			} else {
				_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Connection failed: %v", err)))
			}
		} else {
			_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Connection OK"))
			_, _ = fmt.Fprintf(os.Stdout, "  Messages: %d\n", gmailProfile.MessagesTotal)
			_, _ = fmt.Fprintf(os.Stdout, "  Threads:  %d\n", gmailProfile.ThreadsTotal)
		}
	}

	return nil
}

// parseScopes parses comma-separated scopes
func parseScopes(scopes string) []string {
	if scopes == "" {
		return nil
	}

	var result []string

	for s := range strings.SplitSeq(scopes, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
