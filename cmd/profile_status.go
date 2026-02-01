package cmd

import (
	"context"
	"fmt"
	"os"
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

If no name is provided, shows the default profile.

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

	storage := string(profile.TokenStorage)
	switch profile.TokenStorage {
	case model.TokenStorageEncrypted:
		storage = "encrypted (TPM)"
	case model.TokenStorageOpen:
		storage = "plain text (no TPM)"
	}

	_, _ = fmt.Fprintf(os.Stdout, "Storage: %s\n", storage)
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
