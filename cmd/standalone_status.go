package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var standaloneStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show standalone mode status",
	Long: `Show the current status of standalone mode on this instance.

Displays information about:
  - Whether standalone mode is enabled
  - Instance ID
  - Configured port
  - Key expiration date
  - Connected clients (if any)

Examples:
  clonr standalone status`,
	RunE: runStandaloneStatus,
}

func init() {
	standaloneCmd.AddCommand(standaloneStatusCmd)
}

func runStandaloneStatus(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	// Get standalone config
	config, err := db.GetStandaloneConfig()
	if err != nil {
		return fmt.Errorf("failed to get standalone configuration: %w", err)
	}

	if config == nil || !config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Standalone mode: disabled")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "To enable standalone mode, run: clonr standalone init")

		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, "Standalone mode: enabled")
	_, _ = fmt.Fprintf(os.Stdout, "  Instance ID: %s\n", config.InstanceID)
	_, _ = fmt.Fprintf(os.Stdout, "  Port: %d\n", config.Port)
	_, _ = fmt.Fprintf(os.Stdout, "  Created: %s\n", config.CreatedAt.Format("2006-01-02 15:04:05"))

	// Check expiration
	if !config.ExpiresAt.IsZero() {
		expiresIn := time.Until(config.ExpiresAt)
		if expiresIn > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Expires: %s (%s remaining)\n",
				config.ExpiresAt.Format("2006-01-02 15:04:05"),
				formatDuration(expiresIn))
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "  Expires: %s (EXPIRED)\n",
				config.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "  Capabilities: %v\n", config.Capabilities)

	// Get connected clients
	clients, err := db.GetStandaloneClients()
	if err == nil && len(clients) > 0 {
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Connected clients:")

		for _, client := range clients {
			lastSeen := "never"
			if !client.LastSeen.IsZero() {
				lastSeen = time.Since(client.LastSeen).Round(time.Second).String() + " ago"
			}

			_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s) - last seen: %s, syncs: %d\n",
				client.Name, client.IPAddress, lastSeen, client.SyncCount)
		}
	} else {
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "No connected clients")
	}

	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}
