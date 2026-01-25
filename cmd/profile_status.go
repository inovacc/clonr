package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var profileStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show profile information",
	Long: `Show detailed information about a profile.

If no name is provided, shows the active profile.

Examples:
  clonr profile status
  clonr profile status work`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileStatus,
}

var profileStatusValidate bool

func init() {
	profileCmd.AddCommand(profileStatusCmd)

	profileStatusCmd.Flags().BoolVar(&profileStatusValidate, "validate", false, "Validate token with GitHub API")
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
			if err == core.ErrProfileNotFound {
				return fmt.Errorf("profile '%s' not found", args[0])
			}

			return fmt.Errorf("failed to get profile: %w", err)
		}
	} else {
		profile, err = pm.GetActiveProfile()
		if err != nil {
			if err == core.ErrNoActiveProfile {
				fmt.Println("No active profile.")
				fmt.Println("\nCreate a profile with: clonr profile add <name>")

				return nil
			}

			return fmt.Errorf("failed to get profile: %w", err)
		}
	}

	if profile == nil {
		fmt.Println("No profile found.")

		return nil
	}

	fmt.Printf("Profile: %s\n", profile.Name)
	fmt.Printf("Host: %s\n", profile.Host)
	fmt.Printf("User: %s\n", profile.User)

	storage := string(profile.TokenStorage)
	if profile.TokenStorage == model.TokenStorageKeyring {
		storage = "secure (system keyring)"
	} else if profile.TokenStorage == model.TokenStorageInsecure {
		storage = "encrypted (database)"
	}

	fmt.Printf("Storage: %s\n", storage)
	fmt.Printf("Scopes: %s\n", strings.Join(profile.Scopes, ", "))
	fmt.Printf("Active: %t\n", profile.Active)
	fmt.Printf("Created: %s\n", profile.CreatedAt.Format(time.RFC3339))

	if !profile.LastUsedAt.IsZero() {
		fmt.Printf("Last used: %s\n", profile.LastUsedAt.Format(time.RFC3339))
	}

	// Validate token if requested
	if profileStatusValidate {
		fmt.Println("\nValidating token...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		valid, err := pm.ValidateProfileToken(ctx, profile.Name)
		if err != nil {
			fmt.Printf("Validation error: %v\n", err)
		} else if valid {
			fmt.Println("Token is valid.")
		} else {
			fmt.Println("Token is invalid or expired.")
			fmt.Printf("Re-authenticate with: clonr profile remove %s && clonr profile add %s\n", profile.Name, profile.Name)
		}
	}

	return nil
}
