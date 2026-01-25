package cmd

import (
	"fmt"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

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

func init() {
	profileCmd.AddCommand(profileUseCmd)
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
		if err == core.ErrProfileNotFound {
			return fmt.Errorf("profile '%s' not found", name)
		}

		return fmt.Errorf("failed to set active profile: %w", err)
	}

	if currentActive != nil && currentActive.Name == name {
		fmt.Printf("Profile '%s' is already active.\n", name)

		return nil
	}

	fmt.Printf("Switched to profile: %s\n", name)

	// Show profile info
	profile, err := pm.GetProfile(name)
	if err == nil && profile != nil {
		fmt.Printf("User: %s\n", profile.User)
		fmt.Printf("Host: %s\n", profile.Host)
	}

	return nil
}
