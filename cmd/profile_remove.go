package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

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

var profileRemoveForce bool

func init() {
	profileCmd.AddCommand(profileRemoveCmd)

	profileRemoveCmd.Flags().BoolVarP(&profileRemoveForce, "force", "f", false, "Skip confirmation")
}

func runProfileRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	pm, err := core.NewProfileManager()
	if err != nil {
		return err
	}

	// Check if profile exists
	profile, err := pm.GetProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// Warn if deleting active profile
	if profile.Active && !profileRemoveForce {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: '%s' is the active profile.\n", name)
		_, _ = fmt.Fprint(os.Stdout, "Are you sure you want to delete it? (y/N): ")

		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil {
			return fmt.Errorf("cancelled")
		}

		if confirm != "y" && confirm != "Y" {
			_, _ = fmt.Fprintln(os.Stdout, "Cancelled.")

			return nil
		}
	}

	if err := pm.DeleteProfile(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Profile '%s' deleted.\n", name)

	// If we deleted the active profile, suggest setting a new one
	if profile.Active {
		profiles, listErr := pm.ListProfiles()
		if listErr == nil && len(profiles) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "\nTo set a new active profile: clonr profile use %s\n", profiles[0].Name)
		}
	}

	return nil
}
