package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/gdrive"
	"github.com/inovacc/clonr/internal/gmail"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(gmailCmd)

	// Authentication subcommands
	gmailCmd.AddCommand(gmailAddCmd)
	gmailCmd.AddCommand(gmailRemoveCmd)
	gmailCmd.AddCommand(gmailStatusCmd)

	// Operation subcommands
	gmailCmd.AddCommand(gmailProfileCmd)
	gmailCmd.AddCommand(gmailLabelsCmd)
	gmailCmd.AddCommand(gmailMessagesCmd)
	gmailCmd.AddCommand(gmailReadCmd)
	gmailCmd.AddCommand(gmailSearchCmd)
	gmailCmd.AddCommand(gmailAttachmentsCmd)
	gmailCmd.AddCommand(gmailDownloadCmd)
	gmailCmd.AddCommand(gmailCalendarCmd)
	gmailCmd.AddCommand(gmailDriveCmd)
	gmailCmd.AddCommand(gmailDriveDownloadCmd)

	// Add command flags
	gmailAddCmd.Flags().String("client-id", "", "Google OAuth Client ID")
	gmailAddCmd.Flags().String("client-secret", "", "Google OAuth Client Secret")
	gmailAddCmd.Flags().StringP("token", "t", "", "Access token (skip OAuth flow)")
	gmailAddCmd.Flags().StringP("refresh-token", "r", "", "Refresh token for token rotation")
	gmailAddCmd.Flags().Int("port", 8339, "Local callback server port for OAuth")
	gmailAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	gmailAddCmd.Flags().String("name", "gmail", "Name for the Gmail channel configuration")

	// Remove command flags
	gmailRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Status command flags
	gmailStatusCmd.Flags().Bool("json", false, "Output as JSON")

	// Messages flags
	gmailMessagesCmd.Flags().IntP("limit", "n", 10, "Maximum number of messages to list")
	gmailMessagesCmd.Flags().StringP("label", "l", "INBOX", "Label to filter messages (INBOX, SENT, etc.)")
	gmailMessagesCmd.Flags().Bool("json", false, "Output as JSON")

	// Search flags
	gmailSearchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	gmailSearchCmd.Flags().Bool("json", false, "Output as JSON")

	// Read flags
	gmailReadCmd.Flags().Bool("html", false, "Show HTML body instead of plain text")
	gmailReadCmd.Flags().Bool("json", false, "Output as JSON")

	// Attachments flags
	gmailAttachmentsCmd.Flags().Bool("json", false, "Output as JSON")

	// Download flags
	gmailDownloadCmd.Flags().StringP("output", "o", "", "Output directory (default: current directory)")

	// Calendar flags
	gmailCalendarCmd.Flags().Bool("json", false, "Output as JSON")

	// Drive flags
	gmailDriveCmd.Flags().Bool("json", false, "Output as JSON")
	gmailDriveDownloadCmd.Flags().StringP("output", "o", "", "Output directory (default: current directory)")
}

var gmailCmd = &cobra.Command{
	Use:   "gmail",
	Short: "Gmail integration and operations",
	Long: `Gmail integration and operations for the active profile.

Authentication Commands:
  add          Add Gmail integration via OAuth or access token
  remove       Remove Gmail integration from profile
  status       Show Gmail integration status

Operation Commands:
  profile      Show Gmail account profile
  labels       List Gmail labels
  messages     List recent messages
  read         Read a specific message
  search       Search messages
  attachments  List attachments in a message
  download     Download an attachment
  calendar     Show calendar events in a message
  drive        List Google Drive links in a message
  drive-download  Download a file from Google Drive

Examples:
  # Setup
  clonr gmail add --client-id <id> --client-secret <secret>
  clonr gmail add --token <access_token>
  clonr gmail status
  clonr gmail remove

  # Operations
  clonr gmail messages
  clonr gmail messages --limit 20 --label INBOX
  clonr gmail read <message-id>
  clonr gmail search "from:someone@example.com"
  clonr gmail calendar <message-id>
  clonr gmail drive <message-id>
  clonr gmail drive-download <file-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// ============================================================================
// Authentication Commands
// ============================================================================

var gmailAddCmd = &cobra.Command{
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
  clonr gmail add --client-id <id> --client-secret <secret>
  clonr gmail add --token <access_token>
  GOOGLE_CLIENT_ID=xxx GOOGLE_CLIENT_SECRET=yyy clonr gmail add`,
	RunE: runGmailAdd,
}

var gmailRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Gmail integration from the active profile",
	Long: `Remove Gmail integration from the active profile.

This removes the stored Gmail credentials from the profile.
Use --force to skip confirmation.

Examples:
  clonr gmail remove
  clonr gmail remove --force`,
	RunE: runGmailRemove,
}

var gmailStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Gmail integration status for the active profile",
	Long: `Show Gmail integration status for the active profile.

Displays account information, connection status, and token validity.

Examples:
  clonr gmail status
  clonr gmail status --json`,
	RunE: runGmailStatus,
}

// ============================================================================
// Operation Commands
// ============================================================================

var gmailProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show Gmail account profile",
	RunE:  runGmailProfile,
}

var gmailLabelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List Gmail labels",
	RunE:  runGmailLabels,
}

var gmailMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List recent messages",
	RunE:  runGmailMessages,
}

var gmailReadCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific message",
	Args:  cobra.ExactArgs(1),
	RunE:  runGmailRead,
}

var gmailSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages using Gmail query syntax",
	Long: `Search messages using Gmail query syntax.

Query examples:
  from:someone@example.com     Messages from a specific sender
  to:me                        Messages sent to you
  subject:meeting              Messages with "meeting" in subject
  is:unread                    Unread messages
  has:attachment               Messages with attachments
  after:2024/01/01             Messages after a date
  before:2024/12/31            Messages before a date
  label:important              Messages with a specific label

Examples:
  clonr gmail search "from:boss@company.com"
  clonr gmail search "is:unread has:attachment"
  clonr gmail search "subject:invoice after:2024/01/01"`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailSearch,
}

var gmailAttachmentsCmd = &cobra.Command{
	Use:   "attachments <message-id>",
	Short: "List attachments in a message",
	Long: `List all attachments in a Gmail message.

Examples:
  clonr gmail attachments 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailAttachments,
}

var gmailDownloadCmd = &cobra.Command{
	Use:   "download <message-id> <attachment-id>",
	Short: "Download an attachment from a message",
	Long: `Download an attachment from a Gmail message.

Use 'clonr gmail attachments <message-id>' to get attachment IDs.

Examples:
  clonr gmail download 19c2d20451b4bb54 ANGjdJ8abc123
  clonr gmail download 19c2d20451b4bb54 ANGjdJ8abc123 -o ~/Downloads`,
	Args: cobra.ExactArgs(2),
	RunE: runGmailDownload,
}

var gmailCalendarCmd = &cobra.Command{
	Use:   "calendar <message-id>",
	Short: "Show calendar events in a message",
	Long: `Extract and display calendar events from a Gmail message.

Detects ICS calendar attachments and displays event details including:
- Event title, date/time, location
- Organizer and attendees
- Event status (confirmed, tentative, cancelled)

Examples:
  clonr gmail calendar 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailCalendar,
}

var gmailDriveCmd = &cobra.Command{
	Use:   "drive <message-id>",
	Short: "List Google Drive links in a message",
	Long: `Extract and display Google Drive links from a Gmail message.

Detects links to:
- Google Drive files and folders
- Google Docs, Sheets, and Slides

Examples:
  clonr gmail drive 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailDrive,
}

var gmailDriveDownloadCmd = &cobra.Command{
	Use:   "drive-download <file-id>",
	Short: "Download a file from Google Drive",
	Long: `Download a file from Google Drive using its file ID.

Get file IDs using 'clonr gmail drive <message-id>'.

For Google Docs/Sheets/Slides, the file is exported to PDF/XLSX format.

Examples:
  clonr gmail drive-download 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs
  clonr gmail drive-download 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs -o ~/Downloads`,
	Args: cobra.ExactArgs(1),
	RunE: runGmailDriveDownload,
}

// ============================================================================
// Helper Functions
// ============================================================================

func gmailGetClient() (*gmail.Client, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Gmail integration configured; add with: clonr gmail add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Gmail config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Gmail config")
	}

	return gmail.NewClient(accessToken, gmail.ClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
	}), nil
}

func gmailGetDriveClient() (*gdrive.Client, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Gmail integration configured; add with: clonr gmail add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Gmail config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Gmail config")
	}

	return gdrive.NewClient(accessToken, gdrive.ClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
	}), nil
}

// ============================================================================
// Authentication Command Implementations
// ============================================================================

func runGmailAdd(cmd *cobra.Command, _ []string) error {
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
		return gmailAddWithToken(pm, profile, token, refreshToken, channelName)
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
  clonr gmail add --client-id <id> --client-secret <secret>

Or via environment variables:
  export GOOGLE_CLIENT_ID=<your-client-id>
  export GOOGLE_CLIENT_SECRET=<your-client-secret>

Or provide an access token directly:
  clonr gmail add --token <access_token>

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
		scopeList = gmailParseScopes(scopes)
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
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail profile")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail search \"keyword\"")

	return nil
}

func gmailAddWithToken(pm *core.ProfileManager, profile *model.Profile, token, refreshToken, channelName string) error {
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
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail profile")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail search \"keyword\"")

	return nil
}

func runGmailRemove(cmd *cobra.Command, _ []string) error {
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

func runGmailStatus(cmd *cobra.Command, _ []string) error {
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
		_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr gmail add --token <access_token>")

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

// gmailParseScopes parses comma-separated scopes
func gmailParseScopes(scopes string) []string {
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

// ============================================================================
// Operation Command Implementations
// ============================================================================

func runGmailProfile(_ *cobra.Command, _ []string) error {
	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	profile, err := client.GetProfile(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	printBoxHeader("GMAIL PROFILE")
	printBoxLine("Email", profile.EmailAddress)
	printBoxLine("Messages", fmt.Sprintf("%d", profile.MessagesTotal))
	printBoxLine("Threads", fmt.Sprintf("%d", profile.ThreadsTotal))
	printBoxFooter()

	return nil
}

func runGmailLabels(_ *cobra.Command, _ []string) error {
	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	labels, err := client.ListLabels(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Gmail Labels:")
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, label := range labels {
		if label.Type == "system" {
			_, _ = fmt.Fprintf(os.Stdout, "  %s (system)\n", label.Name)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", label.Name)
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	return nil
}

func runGmailMessages(cmd *cobra.Command, _ []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	label, _ := cmd.Flags().GetString("label")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	opts := gmail.ListMessagesOptions{
		MaxResults: limit,
		LabelIDs:   []string{label},
	}

	resp, err := client.ListMessages(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(resp.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found.")
		return nil
	}

	type messageInfo struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}

	var messages []messageInfo

	for _, ref := range resp.Messages {
		msg, msgErr := client.GetMessage(context.Background(), ref.ID, "metadata")
		if msgErr != nil {
			continue
		}

		info := messageInfo{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Snippet: msg.Snippet,
		}
		messages = append(messages, info)
	}

	if jsonOutput {
		return outputJSON(messages)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Messages in %s (%d):\n", label, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, msg := range messages {
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "  From:    %s\n", msg.From)
		_, _ = fmt.Fprintf(os.Stdout, "  Subject: %s\n", msg.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "  Date:    %s\n", msg.Date)

		if msg.Snippet != "" {
			snippet := msg.Snippet
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  Preview: %s\n", dimStyle.Render(snippet))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runGmailRead(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	showHTML, _ := cmd.Flags().GetBool("html")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if jsonOutput {
		type messageDetail struct {
			ID      string `json:"id"`
			From    string `json:"from"`
			To      string `json:"to"`
			Subject string `json:"subject"`
			Date    string `json:"date"`
			Body    string `json:"body"`
		}

		var body string
		if showHTML {
			body = client.GetMessageHTMLBody(msg)
		} else {
			body = client.GetMessageBody(msg)
		}

		detail := messageDetail{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			To:      msg.Headers["to"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Body:    body,
		}

		return outputJSON(detail)
	}

	attachments := client.GetMessageAttachments(msg)
	hasCalendar := client.HasCalendarEvent(msg)
	driveLinks := client.GetDriveLinks(msg)

	printBoxHeader("MESSAGE")
	printBoxLine("ID", msg.ID)
	printBoxLine("From", msg.Headers["from"])
	printBoxLine("To", msg.Headers["to"])
	printBoxLine("Subject", msg.Headers["subject"])
	printBoxLine("Date", msg.Headers["date"])

	if len(attachments) > 0 {
		printBoxLine("Attachments", fmt.Sprintf("%d file(s)", len(attachments)))
	}

	if hasCalendar {
		printBoxLine("Calendar", "Event detected")
	}

	if len(driveLinks) > 0 {
		printBoxLine("Drive Links", fmt.Sprintf("%d link(s)", len(driveLinks)))
	}

	printBoxFooter()

	// Show attachments if any
	if len(attachments) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Attachments:")

		for _, att := range attachments {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s)\n", att.Filename, gmailFormatAttachmentSize(att.Size))
		}
	}

	// Show calendar events if detected
	if hasCalendar {
		events := client.GetCalendarEventsWithAttachments(context.Background(), msg)
		if len(events) > 0 {
			// Deduplicate events by UID
			seen := make(map[string]bool)

			var uniqueEvents []gmail.CalendarEvent

			for _, event := range events {
				if !seen[event.UID] {
					seen[event.UID] = true
					uniqueEvents = append(uniqueEvents, event)
				}
			}

			_, _ = fmt.Fprintln(os.Stdout, "")
			_, _ = fmt.Fprintln(os.Stdout, "Calendar Events:")

			for _, event := range uniqueEvents {
				if event.IsAllDay {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s, all day)\n", event.Summary, event.Start.Format("2006-01-02"))
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s)\n", event.Summary, event.Start.Format("2006-01-02 15:04"))
				}
			}

			_, _ = fmt.Fprintln(os.Stdout, "")
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("Use 'clonr gmail calendar "+msg.ID+"' for details"))
		}
	}

	// Show Drive links if detected
	if len(driveLinks) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Google Drive Links:")

		for _, link := range driveLinks {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", dimStyle.Render(link.FileID))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("Use 'clonr gmail drive "+msg.ID+"' for details"))
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	var body string
	if showHTML {
		body = client.GetMessageHTMLBody(msg)
	} else {
		body = client.GetMessageBody(msg)
	}

	if body != "" {
		_, _ = fmt.Fprintln(os.Stdout, body)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("(no body content)"))
	}

	return nil
}

func runGmailSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	resp, err := client.SearchMessages(context.Background(), query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(resp.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found matching query.")
		return nil
	}

	type searchResult struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}

	var results []searchResult

	for _, ref := range resp.Messages {
		msg, msgErr := client.GetMessage(context.Background(), ref.ID, "metadata")
		if msgErr != nil {
			continue
		}

		result := searchResult{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Snippet: msg.Snippet,
		}
		results = append(results, result)
	}

	if jsonOutput {
		return outputJSON(results)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Search results for %q (%d):\n", query, len(results))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, result := range results {
		_, _ = fmt.Fprintf(os.Stdout, "  %s %s\n", dimStyle.Render(strconv.Itoa(i+1)+"."), dimStyle.Render(result.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     From:    %s\n", result.From)
		_, _ = fmt.Fprintf(os.Stdout, "     Subject: %s\n", result.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "     Date:    %s\n", result.Date)

		if result.Snippet != "" {
			snippet := result.Snippet
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Preview: %s\n", dimStyle.Render(snippet))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runGmailAttachments(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	attachments := client.GetMessageAttachments(msg)

	if len(attachments) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No attachments found in this message.")
		return nil
	}

	if jsonOutput {
		return outputJSON(attachments)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Attachments (%d):\n", len(attachments))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, att := range attachments {
		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, att.Filename)
		_, _ = fmt.Fprintf(os.Stdout, "     ID:   %s\n", dimStyle.Render(att.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     Type: %s\n", att.MimeType)
		_, _ = fmt.Fprintf(os.Stdout, "     Size: %s\n", gmailFormatAttachmentSize(att.Size))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	_, _ = fmt.Fprintln(os.Stdout, "To download: clonr gmail download", messageID, "<attachment-id>")

	return nil
}

func runGmailDownload(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	attachmentID := args[1]
	outputDir, _ := cmd.Flags().GetString("output")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	// First get the message to find the attachment filename
	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	attachments := client.GetMessageAttachments(msg)

	var filename string

	for _, att := range attachments {
		if att.ID == attachmentID {
			filename = att.Filename

			break
		}
	}

	if filename == "" {
		return fmt.Errorf("attachment not found: %s", attachmentID)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Downloading %s...\n", filename)

	// Download the attachment
	data, err := client.GetAttachment(context.Background(), messageID, attachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}

	// Determine output path
	outputPath := filename
	if outputDir != "" {
		outputPath = outputDir + "/" + filename
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Saved: %s (%s)", outputPath, gmailFormatAttachmentSize(len(data)))))

	return nil
}

func runGmailCalendar(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := gmailGetClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if !client.HasCalendarEvent(msg) {
		_, _ = fmt.Fprintln(os.Stdout, "No calendar events found in this message.")

		return nil
	}

	// Use method that also parses ICS attachments
	allEvents := client.GetCalendarEventsWithAttachments(context.Background(), msg)

	if len(allEvents) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No calendar events could be parsed from this message.")

		return nil
	}

	// Deduplicate events by UID
	seen := make(map[string]bool)

	var events []gmail.CalendarEvent

	for _, event := range allEvents {
		if !seen[event.UID] {
			seen[event.UID] = true
			events = append(events, event)
		}
	}

	if jsonOutput {
		return outputJSON(events)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Calendar Events (%d):\n", len(events))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, event := range events {
		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, event.Summary)

		if event.Method != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Type:      %s\n", event.Method)
		}

		if event.IsAllDay {
			_, _ = fmt.Fprintf(os.Stdout, "     Date:      %s (all day)\n", event.Start.Format("2006-01-02"))
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "     Start:     %s\n", event.Start.Format("2006-01-02 15:04"))
			_, _ = fmt.Fprintf(os.Stdout, "     End:       %s\n", event.End.Format("2006-01-02 15:04"))
		}

		if event.Location != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Location:  %s\n", event.Location)
		}

		if event.Organizer != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Organizer: %s\n", event.Organizer)
		}

		if len(event.Attendees) > 0 {
			if len(event.Attendees) <= 3 {
				_, _ = fmt.Fprintf(os.Stdout, "     Attendees: %s\n", strings.Join(event.Attendees, ", "))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "     Attendees: %s (+%d more)\n",
					strings.Join(event.Attendees[:3], ", "), len(event.Attendees)-3)
			}
		}

		if event.Status != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Status:    %s\n", event.Status)
		}

		if event.Description != "" {
			desc := event.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Details:   %s\n", dimStyle.Render(desc))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runGmailDrive(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	gmailClient, err := gmailGetClient()
	if err != nil {
		return err
	}

	msg, err := gmailClient.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	links := gmailClient.GetDriveLinks(msg)

	if len(links) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No Google Drive links found in this message.")

		return nil
	}

	// Try to get metadata for each link
	driveClient, driveErr := gmailGetDriveClient()

	type linkInfo struct {
		URL      string `json:"url"`
		FileID   string `json:"file_id"`
		Name     string `json:"name,omitempty"`
		MimeType string `json:"mime_type,omitempty"`
		Size     string `json:"size,omitempty"`
		Owner    string `json:"owner,omitempty"`
	}

	var infos []linkInfo

	for _, link := range links {
		info := linkInfo{
			URL:    link.URL,
			FileID: link.FileID,
		}

		// Try to get file metadata if Drive client is available
		if driveErr == nil && driveClient != nil {
			if file, fileErr := driveClient.GetFile(context.Background(), link.FileID); fileErr == nil {
				info.Name = file.Name
				info.MimeType = file.MimeType
				info.Size = gmailFormatFileSizeInt64(file.Size)

				if len(file.Owners) > 0 {
					info.Owner = file.Owners[0].EmailAddress
				}
			}
		}

		infos = append(infos, info)
	}

	if jsonOutput {
		return outputJSON(infos)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Google Drive Links (%d):\n", len(infos))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, info := range infos {
		name := info.Name
		if name == "" {
			name = "(unable to retrieve metadata)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, name)
		_, _ = fmt.Fprintf(os.Stdout, "     ID:   %s\n", dimStyle.Render(info.FileID))

		if info.MimeType != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Type: %s\n", info.MimeType)
		}

		if info.Size != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Size: %s\n", info.Size)
		}

		if info.Owner != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Owner: %s\n", info.Owner)
		}

		_, _ = fmt.Fprintf(os.Stdout, "     URL:  %s\n", dimStyle.Render(info.URL))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	_, _ = fmt.Fprintln(os.Stdout, "To download: clonr gmail drive-download <file-id>")

	return nil
}

func runGmailDriveDownload(cmd *cobra.Command, args []string) error {
	fileID := args[0]
	outputDir, _ := cmd.Flags().GetString("output")

	driveClient, err := gmailGetDriveClient()
	if err != nil {
		return err
	}

	// Get file metadata first
	file, err := driveClient.GetFile(context.Background(), fileID)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Downloading %s...\n", file.Name)

	// Download the file
	data, err := driveClient.DownloadFile(context.Background(), fileID)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Determine output filename
	filename := file.Name

	// Add extension for Google Docs exports
	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps.") {
		ext := gdrive.GetExportExtension(file.MimeType)
		if ext != "" && !strings.HasSuffix(filename, ext) {
			filename += ext
		}
	}

	// Determine output path
	outputPath := filename
	if outputDir != "" {
		outputPath = filepath.Join(outputDir, filename)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Saved: %s (%s)", outputPath, gmailFormatFileSizeInt64(int64(len(data))))))

	return nil
}

// ============================================================================
// Utility Functions
// ============================================================================

// gmailFormatAttachmentSize formats attachment size for display
func gmailFormatAttachmentSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}

	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

// gmailFormatFileSizeInt64 formats file size for display (int64 version)
func gmailFormatFileSizeInt64(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}

	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}

	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
