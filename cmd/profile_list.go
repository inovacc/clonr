package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

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

var profileListJSON bool

func init() {
	profileCmd.AddCommand(profileListCmd)

	profileListCmd.Flags().BoolVar(&profileListJSON, "json", false, "Output as JSON")
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

		_, _ = fmt.Fprintln(os.Stdout, "No profiles configured.")
		_, _ = fmt.Fprintln(os.Stdout, "\nCreate a profile with: clonr profile add <name>")

		return nil
	}

	// JSON output
	if profileListJSON {
		items := make([]ProfileListItem, 0, len(profiles))

		for _, p := range profiles {
			storage := string(p.TokenStorage)
			switch p.TokenStorage {
			case model.TokenStorageEncrypted:
				storage = "encrypted (TPM)"
			case model.TokenStorageOpen:
				storage = "plain text"
			}

			items = append(items, ProfileListItem{
				Name:       p.Name,
				Host:       p.Host,
				User:       p.User,
				Storage:    storage,
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

		storage := string(p.TokenStorage)
		switch p.TokenStorage {
		case model.TokenStorageEncrypted:
			storage = "encrypted (TPM)"
		case model.TokenStorageOpen:
			storage = "plain text"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.Name,
			p.Host,
			p.User,
			storage,
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
