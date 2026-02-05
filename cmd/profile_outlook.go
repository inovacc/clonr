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
	profileCmd.AddCommand(profileOutlookCmd)
	profileOutlookCmd.AddCommand(profileOutlookAddCmd)
	profileOutlookCmd.AddCommand(profileOutlookRemoveCmd)
	profileOutlookCmd.AddCommand(profileOutlookStatusCmd)

	// Add flags
	profileOutlookAddCmd.Flags().String("client-id", "", "Azure AD App Client ID")
	profileOutlookAddCmd.Flags().String("client-secret", "", "Azure AD App Client Secret")
	profileOutlookAddCmd.Flags().String("tenant-id", "common", "Azure AD Tenant ID (common, organizations, or specific tenant)")
	profileOutlookAddCmd.Flags().StringP("token", "t", "", "Access token (skip OAuth flow)")
	profileOutlookAddCmd.Flags().StringP("refresh-token", "r", "", "Refresh token for token rotation")
	profileOutlookAddCmd.Flags().Int("port", 8341, "Local callback server port for OAuth")
	profileOutlookAddCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	profileOutlookAddCmd.Flags().String("name", "outlook", "Name for the Outlook channel configuration")

	profileOutlookRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	profileOutlookStatusCmd.Flags().Bool("json", false, "Output as JSON")
}

var profileOutlookCmd = &cobra.Command{
	Use:   "outlook",
	Short: "Manage Microsoft Outlook integration for the active profile",
	Long: `Manage Microsoft Outlook integration for the active profile.

Outlook credentials are stored securely using the profile's encryption
(TPM-backed when available).

Available Commands:
  add          Add Outlook integration via OAuth or access token
  remove       Remove Outlook integration from profile
  status       Show Outlook integration status

Examples:
  clonr profile outlook add --client-id <id> --client-secret <secret>
  clonr profile outlook add --token <access_token>
  clonr profile outlook status
  clonr profile outlook remove`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var profileOutlookAddCmd = &cobra.Command{
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
  clonr profile outlook add --client-id <id> --client-secret <secret>
  clonr profile outlook add --token <access_token>
  AZURE_CLIENT_ID=xxx AZURE_CLIENT_SECRET=yyy clonr profile outlook add`,
	RunE: runProfileOutlookAdd,
}

var profileOutlookRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove Microsoft Outlook integration from the active profile",
	Long: `Remove Microsoft Outlook integration from the active profile.

This removes the stored Outlook credentials from the profile.
Use --force to skip confirmation.

Examples:
  clonr profile outlook remove
  clonr profile outlook remove --force`,
	RunE: runProfileOutlookRemove,
}

var profileOutlookStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Microsoft Outlook integration status for the active profile",
	Long: `Show Microsoft Outlook integration status for the active profile.

Displays account information, connection status, and inbox statistics.

Examples:
  clonr profile outlook status
  clonr profile outlook status --json`,
	RunE: runProfileOutlookStatus,
}

func runProfileOutlookAdd(cmd *cobra.Command, _ []string) error {
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
		return addOutlookWithToken(pm, profile, token, refreshToken, tenantID, channelName)
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
  clonr profile outlook add --client-id <id> --client-secret <secret>

Or via environment variables:
  export AZURE_CLIENT_ID=<your-client-id>
  export AZURE_CLIENT_SECRET=<your-client-secret>

Or provide an access token directly:
  clonr profile outlook add --token <access_token>

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
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook folders")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook read <message-id>")

	return nil
}

func addOutlookWithToken(pm *core.ProfileManager, profile *model.Profile, token, refreshToken, tenantID, channelName string) error {
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
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook folders")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook messages")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm outlook read <message-id>")

	return nil
}

func runProfileOutlookRemove(cmd *cobra.Command, _ []string) error {
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

func runProfileOutlookStatus(cmd *cobra.Command, _ []string) error {
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
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile outlook add --client-id <id> --client-secret <secret>")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr profile outlook add --token <access_token>")

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
