package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var (
	profileStatusValidate bool
	profileRemoveForce    bool
	profileListJSON       bool
)

func init() {
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileStatusCmd)
	profileCmd.AddCommand(profileRemoveCmd)
	profileCmd.AddCommand(profileListCmd)

	profileStatusCmd.Flags().BoolVar(&profileStatusValidate, "validate", false, "Validate token with GitHub API")
	profileRemoveCmd.Flags().BoolVarP(&profileRemoveForce, "force", "f", false, "Skip confirmation")
	profileListCmd.Flags().BoolVar(&profileListJSON, "json", false, "Output as JSON")
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage GitHub authentication profiles",
	Long: `Manage GitHub authentication profiles for clonr.

Each profile stores GitHub credentials using OAuth device flow authentication.
Tokens are stored securely in the system keyring when available.

Available Commands:
  add          Create a new profile with GitHub OAuth
  list         List all profiles
  use          Set the active profile
  remove       Delete a profile
  status       Show current profile information

Examples:
  clonr profile add work
  clonr profile list
  clonr profile use work
  clonr profile status
  clonr profile remove old-profile`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand provided, show help
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(profileCmd)
}

var profileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new profile with GitHub OAuth or PAT",
	Long: `Create a new GitHub authentication profile.

By default, this will start an OAuth device flow where you:
1. Copy the displayed code
2. Open the GitHub URL in your browser
3. Paste the code and authorize clonr

Alternatively, you can provide a Personal Access Token (PAT) directly
using the --token flag to skip the OAuth flow.

The token will be stored securely in your system keyring if available,
or encrypted in the database as a fallback.

A workspace must be specified using the --workspace flag. The profile
will be associated with this workspace.

Examples:
  clonr profile add work --workspace work
  clonr profile add personal --workspace personal --host github.com
  clonr profile add enterprise --workspace corp --host github.mycompany.com
  clonr profile add myprofile --workspace dev --token ghp_xxxxxxxxxxxx`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileAdd,
}

var (
	profileAddHost      string
	profileAddScopes    []string
	profileAddToken     string
	profileAddWorkspace string
)

func init() {
	profileCmd.AddCommand(profileAddCmd)

	profileAddCmd.Flags().StringVar(&profileAddHost, "host", "github.com", "GitHub host (for enterprise)")
	profileAddCmd.Flags().StringSliceVar(&profileAddScopes, "scopes", nil, "OAuth scopes (default: repo,read:org,gist,read:user,user:email)")
	profileAddCmd.Flags().StringVar(&profileAddToken, "token", "", "Personal Access Token (skip OAuth flow)")
	profileAddCmd.Flags().StringVar(&profileAddWorkspace, "workspace", "", "Associated workspace (required)")

	_ = profileAddCmd.MarkFlagRequired("workspace")
}

func runProfileAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	// Get the client to check if the profile exists
	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	// Check if a profile already exists
	exists, err := client.ProfileExists(name)
	if err != nil {
		return fmt.Errorf("failed to check profile existence: %w", err)
	}

	if exists {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	// Validate workspace exists
	wsExists, err := client.WorkspaceExists(profileAddWorkspace)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !wsExists {
		return fmt.Errorf("workspace '%s' does not exist\nCreate it with: clonr workspace add %s --path <directory>", profileAddWorkspace, profileAddWorkspace)
	}

	// Use default scopes if not specified
	scopes := profileAddScopes
	if len(scopes) == 0 {
		scopes = model.DefaultScopes()
	}

	_, _ = fmt.Fprintf(os.Stdout, "Creating profile: %s\n", name)
	_, _ = fmt.Fprintf(os.Stdout, "Host: %s\n", profileAddHost)
	_, _ = fmt.Fprintf(os.Stdout, "Workspace: %s\n", profileAddWorkspace)

	var token, username string

	// Check if PAT was provided
	if profileAddToken != "" {
		_, _ = fmt.Fprintln(os.Stdout, "Using provided Personal Access Token...")

		// Validate token and get username
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		valid, user, err := core.ValidateToken(ctx, profileAddToken, profileAddHost)
		if err != nil {
			return fmt.Errorf("failed to validate token: %w", err)
		}

		if !valid {
			return fmt.Errorf("invalid or expired token")
		}

		token = profileAddToken
		username = user
		_, _ = fmt.Fprintf(os.Stdout, "Token validated for user: %s\n", username)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Scopes: %s\n\n", strings.Join(scopes, ", "))

		// Run OAuth flow
		flow := core.NewOAuthFlow(profileAddHost, scopes)

		flow.OnDeviceCode(func(code, url string) {
			_, _ = fmt.Fprintln(os.Stdout, "GitHub OAuth Authentication")
			_, _ = fmt.Fprintln(os.Stdout, strings.Repeat("-", 40))
			_, _ = fmt.Fprintf(os.Stdout, "\n1. Copy this code: %s\n\n", code)
			_, _ = fmt.Fprintf(os.Stdout, "2. Open: %s\n\n", url)
			_, _ = fmt.Fprintln(os.Stdout, "3. Paste the code and authorize clonr")
			_, _ = fmt.Fprintln(os.Stdout, "\nWaiting for authorization...")
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := flow.Run(ctx)
		if err != nil {
			return fmt.Errorf("OAuth authentication failed: %w", err)
		}

		token = result.Token
		username = result.Username
		scopes = result.Scopes
	}

	// Encrypt and store the token
	encryptedToken, err := tpm.EncryptToken(token, name, profileAddHost)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Determine a storage type based on whether data is encrypted or open
	tokenStorage := model.TokenStorageEncrypted
	if tpm.IsDataOpen(encryptedToken) {
		tokenStorage = model.TokenStorageOpen
	}

	// Check if this is the first profile (make it active)
	profiles, err := client.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	isFirstProfile := len(profiles) == 0

	// Create profile
	profile := &model.Profile{
		Name:           name,
		Host:           profileAddHost,
		User:           username,
		TokenStorage:   tokenStorage,
		Scopes:         scopes,
		Default:        isFirstProfile,
		EncryptedToken: encryptedToken,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
		Workspace:      profileAddWorkspace,
	}

	// Save profile to BoltDB
	if err := client.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nSuccess!")
	_, _ = fmt.Fprintf(os.Stdout, "Profile: %s\n", profile.Name)
	_, _ = fmt.Fprintf(os.Stdout, "User: %s\n", profile.User)
	_, _ = fmt.Fprintf(os.Stdout, "Workspace: %s\n", profile.Workspace)
	_, _ = fmt.Fprintf(os.Stdout, "Storage: %s\n", tokenStorage)

	if isFirstProfile {
		_, _ = fmt.Fprintln(os.Stdout, "\nThis profile is now the default.")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "\nTo set as default: clonr profile use %s\n", name)
	}

	return nil
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long: `List all GitHub authentication profiles.

The default profile is marked with an asterisk (*).

Examples:
  clonr profile list
  clonr profile list --json`,
	Aliases: []string{"ls"},
	RunE:    runProfileList,
}

// ProfileListItem represents a profile in JSON output
type ProfileListItem struct {
	Name       string    `json:"name"`
	Host       string    `json:"host"`
	User       string    `json:"user"`
	Storage    string    `json:"storage"`
	Scopes     []string  `json:"scopes"`
	Workspace  string    `json:"workspace,omitempty"`
	Default    bool      `json:"default"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitzero"`
}

func runProfileList(_ *cobra.Command, _ []string) error {
	pm, err := core.NewProfileManager()
	if err != nil {
		return err
	}

	profiles, err := pm.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		if profileListJSON {
			_, _ = fmt.Fprintln(os.Stdout, "[]")
			return nil
		}

		printEmptyResult("profiles", "clonr profile add <name>")

		return nil
	}

	// JSON output
	if profileListJSON {
		items := make([]ProfileListItem, 0, len(profiles))

		for _, p := range profiles {
			items = append(items, ProfileListItem{
				Name:       p.Name,
				Host:       p.Host,
				User:       p.User,
				Storage:    formatTokenStorage(p.TokenStorage),
				Scopes:     p.Scopes,
				Workspace:  p.Workspace,
				Default:    p.Default,
				CreatedAt:  p.CreatedAt,
				LastUsedAt: p.LastUsedAt,
			})
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(items)
	}

	// Text output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "NAME\tHOST\tUSER\tSTORAGE\tDEFAULT")
	_, _ = fmt.Fprintln(w, "----\t----\t----\t-------\t-------")

	for _, p := range profiles {
		defaultMarker := ""
		if p.Default {
			defaultMarker = "*"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.Name,
			p.Host,
			p.User,
			formatTokenStorage(p.TokenStorage),
			defaultMarker,
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	// Show scopes info for default profile
	for _, p := range profiles {
		if p.Default {
			_, _ = fmt.Fprintf(os.Stdout, "\nDefault profile: %s\n", p.Name)
			_, _ = fmt.Fprintf(os.Stdout, "Scopes: %s\n", strings.Join(p.Scopes, ", "))

			break
		}
	}

	return nil
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile",
	Long: `Delete a GitHub authentication profile.

This removes the profile and its stored token.

Examples:
  clonr profile remove old-profile`,
	Aliases: []string{"rm", "delete"},
	Args:    cobra.ExactArgs(1),
	RunE:    runProfileRemove,
}

func runProfileRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	pm, err := core.NewProfileManager()
	if err != nil {
		return err
	}

	// Check if a profile exists
	profile, err := pm.GetProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// Warn if deleting default profile
	if profile.Default && !profileRemoveForce {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: '%s' is the default profile.\n", name)
		if !promptConfirm("Are you sure you want to delete it? [y/N]: ") {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")
			return nil
		}
	}

	if err := pm.DeleteProfile(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Profile '%s' deleted.\n", name)

	// If we deleted the default profile, suggest setting a new one
	if profile.Default {
		profiles, listErr := pm.ListProfiles()
		if listErr == nil && len(profiles) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "\nTo set a new default profile: clonr profile use %s\n", profiles[0].Name)
		}
	}

	return nil
}

var profileStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show profile information",
	Long: `Show detailed information about a profile.

If no name is provided, shows the default profile.

Examples:
  clonr profile status
  clonr profile status work`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileStatus,
}

func runProfileStatus(_ *cobra.Command, args []string) error {
	pm, err := core.NewProfileManager()
	if err != nil {
		return err
	}

	var profile *model.Profile

	if len(args) > 0 {
		profile, err = pm.GetProfile(args[0])
		if err != nil {
			if errors.Is(err, core.ErrProfileNotFound) {
				return fmt.Errorf("profile '%s' not found", args[0])
			}

			return fmt.Errorf("failed to get profile: %w", err)
		}
	} else {
		profile, err = pm.GetActiveProfile()
		if err != nil {
			if errors.Is(err, core.ErrNoActiveProfile) {
				_, _ = fmt.Fprintln(os.Stdout, "No default profile.")
				_, _ = fmt.Fprintln(os.Stdout, "\nCreate a profile with: clonr profile add <name>")

				return nil
			}

			return fmt.Errorf("failed to get profile: %w", err)
		}
	}

	if profile == nil {
		_, _ = fmt.Fprintln(os.Stdout, "No profile found.")

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Profile: %s\n", profile.Name)
	_, _ = fmt.Fprintf(os.Stdout, "Host: %s\n", profile.Host)
	_, _ = fmt.Fprintf(os.Stdout, "User: %s\n", profile.User)
	_, _ = fmt.Fprintf(os.Stdout, "Storage: %s\n", formatTokenStorage(profile.TokenStorage))
	_, _ = fmt.Fprintf(os.Stdout, "Scopes: %s\n", strings.Join(profile.Scopes, ", "))
	_, _ = fmt.Fprintf(os.Stdout, "Default: %t\n", profile.Default)
	_, _ = fmt.Fprintf(os.Stdout, "Created: %s\n", profile.CreatedAt.Format(time.RFC3339))

	if !profile.LastUsedAt.IsZero() {
		_, _ = fmt.Fprintf(os.Stdout, "Last used: %s\n", profile.LastUsedAt.Format(time.RFC3339))
	}

	// Validate token if requested
	if profileStatusValidate {
		_, _ = fmt.Fprintln(os.Stdout, "\nValidating token...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		valid, err := pm.ValidateProfileToken(ctx, profile.Name)

		switch {
		case err != nil:
			_, _ = fmt.Fprintf(os.Stdout, "Validation error: %v\n", err)
		case valid:
			_, _ = fmt.Fprintln(os.Stdout, "Token is valid.")
		default:
			_, _ = fmt.Fprintln(os.Stdout, "Token is invalid or expired.")
			_, _ = fmt.Fprintf(os.Stdout, "Re-authenticate with: clonr profile remove %s && clonr profile add %s\n", profile.Name, profile.Name)
		}
	}

	return nil
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active profile",
	Long: `Set a profile as the active profile.

The active profile's token will be used by default for GitHub operations.

Examples:
  clonr profile use work
  clonr profile use personal`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileUse,
}

func runProfileUse(_ *cobra.Command, args []string) error {
	name := args[0]

	pm, err := core.NewProfileManager()
	if err != nil {
		return err
	}

	// Get current active profile for comparison
	currentActive, _ := pm.GetActiveProfile()

	if err := pm.SetActiveProfile(name); err != nil {
		if errors.Is(err, core.ErrProfileNotFound) {
			return fmt.Errorf("profile '%s' not found", name)
		}

		return fmt.Errorf("failed to set active profile: %w", err)
	}

	if currentActive != nil && currentActive.Name == name {
		_, _ = fmt.Fprintf(os.Stdout, "Profile '%s' is already active.\n", name)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Switched to profile: %s\n", name)

	// Show profile info
	profile, err := pm.GetProfile(name)
	if err == nil && profile != nil {
		_, _ = fmt.Fprintf(os.Stdout, "User: %s\n", profile.User)
		_, _ = fmt.Fprintf(os.Stdout, "Host: %s\n", profile.Host)
	}

	return nil
}
