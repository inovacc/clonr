package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/microsoft"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(outlookCmd)

	// Authentication subcommands
	outlookCmd.AddCommand(outlookAddCmd)
	outlookCmd.AddCommand(outlookRemoveCmd)
	outlookCmd.AddCommand(outlookStatusCmd)

	// Operation subcommands
	outlookCmd.AddCommand(outlookFoldersCmd)
	outlookCmd.AddCommand(outlookMessagesCmd)
	outlookCmd.AddCommand(outlookReadCmd)
	outlookCmd.AddCommand(outlookSearchCmd)

	// Add command flags
	outlookAddCmd.Flags().String("client-id", "", "Azure AD App Client ID")
	outlookAddCmd.Flags().String("client-secret", "", "Azure AD App Client Secret")
	outlookAddCmd.Flags().String("tenant-id", "common", "Azure AD Tenant ID (common, organizations, or specific tenant)")
	outlookAddCmd.Flags().StringP("token", "t", "", "Access token (skip OAuth flow)")
	outlookAddCmd.Flags().StringP("refresh-token", "r", "", "Refresh token for token rotation")
	outlookAddCmd.Flags().Int("port", 8341, "Local callback server port for OAuth")
	outlookAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	outlookAddCmd.Flags().String("name", "outlook", "Name for the Outlook channel configuration")

	// Remove command flags
	outlookRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	// Status command flags
	outlookStatusCmd.Flags().Bool("json", false, "Output as JSON")

	// Folders command flags
	outlookFoldersCmd.Flags().Bool("json", false, "Output as JSON")

	// Messages command flags
	outlookMessagesCmd.Flags().IntP("limit", "n", 10, "Maximum number of messages")
	outlookMessagesCmd.Flags().StringP("folder", "f", "inbox", "Folder to list messages from")
	outlookMessagesCmd.Flags().Bool("json", false, "Output as JSON")

	// Read command flags
	outlookReadCmd.Flags().Bool("json", false, "Output as JSON")

	// Search command flags
	outlookSearchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	outlookSearchCmd.Flags().Bool("json", false, "Output as JSON")
}

var outlookCmd = &cobra.Command{
	Use:   "outlook",
	Short: "Microsoft Outlook integration and operations",
	Long: `Microsoft Outlook integration and operations for the active profile.

Authentication Commands:
  add          Add Outlook integration via OAuth or access token
  remove       Remove Outlook integration from profile
  status       Show Outlook integration status

Operation Commands:
  folders    List mail folders
  messages   List messages in a folder
  read       Read a specific message
  search     Search messages

Examples:
  # Setup
  clonr outlook add --client-id <id> --client-secret <secret>
  clonr outlook add --token <access_token>
  clonr outlook status
  clonr outlook remove

  # Operations
  clonr outlook folders
  clonr outlook messages
  clonr outlook messages --folder sentitems
  clonr outlook read <message-id>
  clonr outlook search "project update"`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// ============================================================================
// Authentication Commands
// ============================================================================

var outlookAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add Microsoft Outlook integration to the active profile",
	Long: `Add Microsoft Outlook integration to the active profile.

By default, this starts an OAuth flow where you:
1. A browser window opens to Microsoft authorization
2. Authorize clonr for your Outlook account
3. The access token is automatically saved to your profile

Alternatively, provide an access token directly with --token to skip OAuth.

Prerequisites for OAuth:
  1. Go to https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
  2. Create a new App Registration
  3. Add redirect URI: http://localhost:8341/microsoft/callback (Web platform)
  4. Create a client secret in "Certificates & secrets"
  5. Add API permissions for Microsoft Graph:
     - Mail.Read
     - Mail.ReadBasic
     - User.Read

Examples:
  clonr outlook add --client-id <id> --client-secret <secret>
  clonr outlook add --token <access_token>
  AZURE_CLIENT_ID=xxx AZURE_CLIENT_SECRET=yyy clonr outlook add`,
	RunE: runOutlookAdd,
}

var outlookRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Microsoft Outlook integration from the active profile",
	Long: `Remove Microsoft Outlook integration from the active profile.

This removes the stored Outlook credentials from the profile.
Use --force to skip confirmation.

Examples:
  clonr outlook remove
  clonr outlook remove --force`,
	RunE: runOutlookRemove,
}

var outlookStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Microsoft Outlook integration status for the active profile",
	Long: `Show Microsoft Outlook integration status for the active profile.

Displays account information, connection status, and inbox statistics.

Examples:
  clonr outlook status
  clonr outlook status --json`,
	RunE: runOutlookStatus,
}

// ============================================================================
// Operation Commands
// ============================================================================

var outlookFoldersCmd = &cobra.Command{
	Use:   "folders",
	Short: "List mail folders",
	RunE:  runOutlookFolders,
}

var outlookMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List messages in a folder",
	RunE:  runOutlookMessages,
}

var outlookReadCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific message",
	Args:  cobra.ExactArgs(1),
	RunE:  runOutlookRead,
}

var outlookSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages",
	Long: `Search messages in Outlook.

The search query uses Microsoft's search syntax.

Examples:
  clonr outlook search "project update"
  clonr outlook search "from:boss@company.com"
  clonr outlook search "hasattachment:true"`,
	Args: cobra.ExactArgs(1),
	RunE: runOutlookSearch,
}

// ============================================================================
// Helper Functions
// ============================================================================

func outlookGetClient() (*microsoft.OutlookClient, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelOutlook)
	if err != nil {
		return nil, fmt.Errorf("failed to get Outlook config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Outlook integration configured; add with: clonr outlook add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Outlook config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Outlook config")
	}

	return microsoft.NewOutlookClient(accessToken, microsoft.OutlookClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
		TenantID:     config["tenant_id"],
	}), nil
}

// ============================================================================
// Authentication Command Implementations
// ============================================================================

func runOutlookAdd(cmd *cobra.Command, _ []string) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	tenantID, _ := cmd.Flags().GetString("tenant-id")
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

	_, _ = fmt.Fprintf(os.Stdout, "Adding Microsoft Outlook to profile %q\n\n", profile.Name)

	// If token provided directly, skip OAuth
	if token != "" {
		return outlookAddWithToken(pm, profile, token, refreshToken, tenantID, channelName)
	}

	// Try environment variables if flags not provided
	if clientID == "" {
		clientID = os.Getenv("AZURE_CLIENT_ID")
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`azure AD Client ID and Client Secret are required for OAuth

Provide them via flags:
  clonr outlook add --client-id <id> --client-secret <secret>

Or via environment variables:
  export AZURE_CLIENT_ID=<your-client-id>
  export AZURE_CLIENT_SECRET=<your-client-secret>

Or provide an access token directly:
  clonr outlook add --token <access_token>

To get OAuth credentials:
  1. Go to https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
  2. Create a new App Registration
  3. Add redirect URI: http://localhost:%d/microsoft/callback
  4. Create a client secret in "Certificates & secrets"
  5. Add API permissions for Microsoft Graph (Mail.Read)`, port)
	}

	// Parse scopes if provided
	var scopeList []string

	if scopes != "" {
		for s := range strings.SplitSeq(scopes, ",") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				scopeList = append(scopeList, trimmed)
			}
		}
	} else {
		scopeList = microsoft.DefaultOutlookScopes
	}

	// Run OAuth flow
	config := microsoft.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Port:         port,
		Scopes:       scopeList,
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Starting Microsoft Outlook OAuth flow..."))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "A browser window will open for authorization.")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Waiting for authorization (timeout: 5 minutes)..."))
	_, _ = fmt.Fprintln(os.Stdout, "")

	result, err := microsoft.RunOAuthFlow(cmd.Context(), config, core.OpenBrowser)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Authorization successful!"))
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Get user profile
	userProfile, err := microsoft.GetUserProfile(cmd.Context(), result.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	// Display connection info
	_, _ = fmt.Fprintf(os.Stdout, "  User:   %s\n", userProfile.DisplayName)
	_, _ = fmt.Fprintf(os.Stdout, "  Email:  %s\n", userProfile.Mail)
	_, _ = fmt.Fprintf(os.Stdout, "  Scopes: %s\n", result.Scope)
	_, _ = fmt.Fprintln(os.Stdout, "")

	email := userProfile.Mail
	if email == "" {
		email = userProfile.UserPrincipalName
	}

	// Create NotifyChannel for Outlook
	notifyChannel := &model.NotifyChannel{
		ID:   channelName,
		Type: model.ChannelOutlook,
		Name: fmt.Sprintf("Outlook - %s", userProfile.DisplayName),
		Config: map[string]string{
			"access_token":  result.AccessToken,
			"refresh_token": result.RefreshToken,
			"client_id":     clientID,
			"client_secret": clientSecret,
			"tenant_id":     tenantID,
			"user_email":    email,
			"user_name":     userProfile.DisplayName,
		},
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile (AddNotifyChannel encrypts sensitive fields)
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Outlook credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Microsoft Outlook added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Outlook commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook folders")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook read <message-id>")

	return nil
}

func outlookAddWithToken(pm *core.ProfileManager, profile *model.Profile, token, refreshToken, tenantID, channelName string) error {
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Validating access token..."))

	// Get user profile to validate token
	userProfile, err := microsoft.GetUserProfile(context.Background(), token)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Token validated!"))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "  User:  %s\n", userProfile.DisplayName)
	_, _ = fmt.Fprintf(os.Stdout, "  Email: %s\n", userProfile.Mail)
	_, _ = fmt.Fprintln(os.Stdout, "")

	email := userProfile.Mail
	if email == "" {
		email = userProfile.UserPrincipalName
	}

	// Create NotifyChannel for Outlook
	config := map[string]string{
		"access_token": token,
		"tenant_id":    tenantID,
		"user_email":   email,
		"user_name":    userProfile.DisplayName,
	}

	// Add refresh token if provided
	if refreshToken != "" {
		config["refresh_token"] = refreshToken
	}

	notifyChannel := &model.NotifyChannel{
		ID:        channelName,
		Type:      model.ChannelOutlook,
		Name:      fmt.Sprintf("Outlook - %s", userProfile.DisplayName),
		Config:    config,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to profile
	if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
		return fmt.Errorf("failed to save Outlook credentials: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Microsoft Outlook added to profile %q!", profile.Name)))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "You can now use Outlook commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook folders")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook read <message-id>")

	return nil
}

func runOutlookRemove(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Check if Outlook is configured
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelOutlook)
	if err != nil {
		return fmt.Errorf("failed to check Outlook integration: %w", err)
	}

	if channel == nil {
		return fmt.Errorf("no Outlook integration found in profile %q", profile.Name)
	}

	// Confirm removal
	if !force {
		_, _ = fmt.Fprintf(os.Stdout, "Remove Outlook integration from profile %q? [y/N] ", profile.Name)

		if !promptConfirm("") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := pm.RemoveNotifyChannel(profile.Name, channel.ID); err != nil {
		return fmt.Errorf("failed to remove Outlook integration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Outlook integration removed."))

	return nil
}

func runOutlookStatus(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	pm, err := core.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("no active profile")
	}

	// Get Outlook channel
	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelOutlook)
	if err != nil {
		return fmt.Errorf("failed to check Outlook integration: %w", err)
	}

	if channel == nil {
		if jsonOutput {
			_, _ = fmt.Fprintln(os.Stdout, `{"configured": false}`)
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "No Outlook integration configured for profile %q\n", profile.Name)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Add Outlook with:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr outlook add --token <access_token>")

		return nil
	}

	// Decrypt config for display
	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return fmt.Errorf("failed to decrypt Outlook config: %w", err)
	}

	if jsonOutput {
		type outlookStatus struct {
			Configured bool   `json:"configured"`
			Profile    string `json:"profile"`
			UserName   string `json:"user_name,omitempty"`
			UserEmail  string `json:"user_email,omitempty"`
			TenantID   string `json:"tenant_id,omitempty"`
			Enabled    bool   `json:"enabled"`
			CreatedAt  string `json:"created_at"`
		}

		status := outlookStatus{
			Configured: true,
			Profile:    profile.Name,
			UserName:   config["user_name"],
			UserEmail:  config["user_email"],
			TenantID:   config["tenant_id"],
			Enabled:    channel.Enabled,
			CreatedAt:  channel.CreatedAt.Format(time.RFC3339),
		}

		return outputJSON(status)
	}

	// Display status
	printBoxHeader("OUTLOOK INTEGRATION")
	printBoxLine("Profile", profile.Name)
	printBoxLine("User", config["user_name"])
	printBoxLine("Email", config["user_email"])

	if config["tenant_id"] != "" && config["tenant_id"] != "common" {
		printBoxLine("Tenant", config["tenant_id"])
	}

	printBoxLine("Enabled", fmt.Sprintf("%t", channel.Enabled))
	printBoxLine("Added", channel.CreatedAt.Format("2006-01-02 15:04:05"))
	printBoxFooter()

	// Test connection
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Testing connection..."))

	accessToken := config["access_token"]
	if accessToken != "" {
		client := microsoft.NewOutlookClient(accessToken, microsoft.OutlookClientOptions{})

		inbox, err := client.GetInboxStats(context.Background())
		if err != nil {
			_, _ = fmt.Fprintln(os.Stdout, warnStyle.Render(fmt.Sprintf("Connection failed: %v", err)))
		} else {
			_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Connection OK"))
			_, _ = fmt.Fprintf(os.Stdout, "  Inbox:  %d messages (%d unread)\n", inbox.TotalItemCount, inbox.UnreadItemCount)
		}
	}

	return nil
}

// ============================================================================
// Operation Command Implementations
// ============================================================================

func runOutlookFolders(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := outlookGetClient()
	if err != nil {
		return err
	}

	folders, err := client.GetMailFolders(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	if len(folders) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No folders found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(folders)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Mail Folders:")
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, folder := range folders {
		unread := ""
		if folder.UnreadItemCount > 0 {
			unread = fmt.Sprintf(" (%d unread)", folder.UnreadItemCount)
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s - %d messages%s\n", folder.DisplayName, folder.TotalItemCount, unread)
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("ID: "+folder.ID))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runOutlookMessages(cmd *cobra.Command, _ []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	folder, _ := cmd.Flags().GetString("folder")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := outlookGetClient()
	if err != nil {
		return err
	}

	messages, err := client.ListMessages(context.Background(), microsoft.ListMailMessagesOptions{
		Top:      limit,
		FolderID: folder,
	})
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found.")
		return nil
	}

	if jsonOutput {
		type msgInfo struct {
			ID          string `json:"id"`
			Subject     string `json:"subject"`
			From        string `json:"from"`
			Received    string `json:"received"`
			IsRead      bool   `json:"is_read"`
			BodyPreview string `json:"body_preview"`
		}

		var infos []msgInfo

		for _, msg := range messages {
			from := ""
			if msg.From != nil && msg.From.EmailAddress != nil {
				from = msg.From.EmailAddress.Address
			}

			infos = append(infos, msgInfo{
				ID:          msg.ID,
				Subject:     msg.Subject,
				From:        from,
				Received:    msg.ReceivedDateTime.Format(time.RFC3339),
				IsRead:      msg.IsRead,
				BodyPreview: msg.BodyPreview,
			})
		}

		return outputJSON(infos)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Messages in %s (%d):\n", folder, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, msg := range messages {
		from := "Unknown"
		if msg.From != nil && msg.From.EmailAddress != nil {
			from = fmt.Sprintf("%s <%s>", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
		}

		readStatus := ""
		if !msg.IsRead {
			readStatus = " [UNREAD]"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "  Subject: %s%s\n", msg.Subject, readStatus)
		_, _ = fmt.Fprintf(os.Stdout, "  From:    %s\n", from)
		_, _ = fmt.Fprintf(os.Stdout, "  Date:    %s\n", msg.ReceivedDateTime.Format("2006-01-02 15:04"))

		if msg.BodyPreview != "" {
			preview := msg.BodyPreview
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  Preview: %s\n", dimStyle.Render(preview))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runOutlookRead(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := outlookGetClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if jsonOutput {
		return outputJSON(msg)
	}

	from := "Unknown"
	if msg.From != nil && msg.From.EmailAddress != nil {
		from = fmt.Sprintf("%s <%s>", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
	}

	var toAddrs []string

	for _, r := range msg.ToRecipients {
		if r.EmailAddress != nil {
			toAddrs = append(toAddrs, r.EmailAddress.Address)
		}
	}

	to := "Unknown"
	if len(toAddrs) > 0 {
		to = toAddrs[0]
		if len(toAddrs) > 1 {
			to += fmt.Sprintf(" (+%d more)", len(toAddrs)-1)
		}
	}

	printBoxHeader("MESSAGE")
	printBoxLine("ID", msg.ID)
	printBoxLine("Subject", msg.Subject)
	printBoxLine("From", from)
	printBoxLine("To", to)
	printBoxLine("Date", msg.ReceivedDateTime.Format("2006-01-02 15:04:05"))

	if msg.HasAttachments {
		printBoxLine("Attachments", "Yes")
	}

	printBoxFooter()

	_, _ = fmt.Fprintln(os.Stdout, "")

	if msg.Body != nil && msg.Body.Content != "" {
		_, _ = fmt.Fprintln(os.Stdout, msg.Body.Content)
	} else if msg.BodyPreview != "" {
		_, _ = fmt.Fprintln(os.Stdout, msg.BodyPreview)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("(no body content)"))
	}

	return nil
}

func runOutlookSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := outlookGetClient()
	if err != nil {
		return err
	}

	messages, err := client.SearchMessages(context.Background(), query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found matching query.")
		return nil
	}

	if jsonOutput {
		return outputJSON(messages)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Search results for %q (%d):\n", query, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, msg := range messages {
		from := "Unknown"
		if msg.From != nil && msg.From.EmailAddress != nil {
			from = msg.From.EmailAddress.Address
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     Subject: %s\n", msg.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "     From:    %s\n", from)
		_, _ = fmt.Fprintf(os.Stdout, "     Date:    %s\n", msg.ReceivedDateTime.Format("2006-01-02 15:04"))

		if msg.BodyPreview != "" {
			preview := msg.BodyPreview
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Preview: %s\n", dimStyle.Render(preview))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}
