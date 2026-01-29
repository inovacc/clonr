package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long: `List all GitHub authentication profiles.

The active profile is marked with an asterisk (*).

Examples:
  clonr profile list`,
	Aliases: []string{"ls"},
	RunE:    runProfileList,
}

func init() {
	profileCmd.AddCommand(profileListCmd)
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
		_, _ = fmt.Fprintln(os.Stdout, "No profiles configured.")
		_, _ = fmt.Fprintln(os.Stdout, "\nCreate a profile with: clonr profile add <name>")

		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "NAME\tHOST\tUSER\tSTORAGE\tACTIVE")
	_, _ = fmt.Fprintln(w, "----\t----\t----\t-------\t------")

	for _, p := range profiles {
		active := ""
		if p.Active {
			active = "*"
		}

		storage := string(p.TokenStorage)
		switch storage {
		case "keyring":
			storage = "secure"
		case "insecure_storage":
			storage = "encrypted"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.Name,
			p.Host,
			p.User,
			storage,
			active,
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	// Show scopes info for active profile
	for _, p := range profiles {
		if p.Active {
			_, _ = fmt.Fprintf(os.Stdout, "\nActive profile: %s\n", p.Name)
			_, _ = fmt.Fprintf(os.Stdout, "Scopes: %s\n", strings.Join(p.Scopes, ", "))

			break
		}
	}

	return nil
}
