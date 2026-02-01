package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

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

	// Get client to check if profile exists
	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Check if profile already exists
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

	// Determine storage type based on whether data is encrypted or open
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
		Active:         isFirstProfile,
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
		_, _ = fmt.Fprintln(os.Stdout, "\nThis profile is now active.")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "\nTo use this profile: clonr profile use %s\n", name)
	}

	return nil
}
