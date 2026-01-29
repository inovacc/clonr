package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var profileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new profile with GitHub OAuth",
	Long: `Create a new GitHub authentication profile.

This will start an OAuth device flow where you:
1. Copy the displayed code
2. Open the GitHub URL in your browser
3. Paste the code and authorize clonr

The token will be stored securely in your system keyring if available,
or encrypted in the database as a fallback.

Examples:
  clonr profile add work
  clonr profile add personal --host github.com
  clonr profile add enterprise --host github.mycompany.com`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileAdd,
}

var (
	profileAddHost   string
	profileAddScopes []string
)

func init() {
	profileCmd.AddCommand(profileAddCmd)

	profileAddCmd.Flags().StringVar(&profileAddHost, "host", "github.com", "GitHub host (for enterprise)")
	profileAddCmd.Flags().StringSliceVar(&profileAddScopes, "scopes", nil, "OAuth scopes (default: repo,read:org,gist,read:user,user:email)")
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

	// Use default scopes if not specified
	scopes := profileAddScopes
	if len(scopes) == 0 {
		scopes = model.DefaultScopes()
	}

	_, _ = fmt.Fprintf(os.Stdout, "Creating profile: %s\n", name)
	_, _ = fmt.Fprintf(os.Stdout, "Host: %s\n", profileAddHost)
	_, _ = fmt.Fprintf(os.Stdout, "Scopes: %s\n\n", strings.Join(scopes, ", "))

	// Run OAuth flow
	flow := core.NewOAuthFlow(profileAddHost, scopes)

	var deviceCode, verificationURL string

	flow.OnDeviceCode(func(code, url string) {
		deviceCode = code
		verificationURL = url

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

	// Store token
	var tokenStorage model.TokenStorage

	var encryptedToken []byte

	if err := core.SetToken(name, profileAddHost, result.Token); err != nil {
		// Keyring not available, use encrypted storage
		encryptedToken, encErr := core.EncryptToken(result.Token, name, profileAddHost)
		if encErr != nil {
			return fmt.Errorf("failed to encrypt token: %w", encErr)
		}

		tokenStorage = model.TokenStorageInsecure
		_ = encryptedToken
	} else {
		tokenStorage = model.TokenStorageKeyring
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
		User:           result.Username,
		TokenStorage:   tokenStorage,
		Scopes:         result.Scopes,
		Active:         isFirstProfile,
		EncryptedToken: encryptedToken,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
	}

	// Save profile
	if err := client.SaveProfile(profile); err != nil {
		// Clean up token on failure
		if tokenStorage == model.TokenStorageKeyring {
			_ = core.DeleteToken(name, profileAddHost)
		}

		return fmt.Errorf("failed to save profile: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nSuccess!")
	_, _ = fmt.Fprintf(os.Stdout, "Profile: %s\n", profile.Name)
	_, _ = fmt.Fprintf(os.Stdout, "User: %s\n", profile.User)
	_, _ = fmt.Fprintf(os.Stdout, "Storage: %s\n", tokenStorage)

	if isFirstProfile {
		_, _ = fmt.Fprintln(os.Stdout, "\nThis profile is now active.")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "\nTo use this profile: clonr profile use %s\n", name)
	}

	// Return device code info for potential further use
	_ = deviceCode
	_ = verificationURL

	return nil
}
