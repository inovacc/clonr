package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var (
	clientsShowAll bool
)

var standaloneClientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "List registered clients",
	Long: `List all clients that have registered with this standalone instance.

Each client has its own encryption key for sensitive data. The key hint
shows the first few characters to help identify which key belongs to which client.

Examples:
  # List all registered clients
  clonr standalone clients

  # Show all details including suspended clients
  clonr standalone clients --all`,
	RunE: runStandaloneClients,
}

func init() {
	standaloneCmd.AddCommand(standaloneClientsCmd)

	standaloneClientsCmd.Flags().BoolVar(&clientsShowAll, "all", false, "Show all clients including suspended")
}

func runStandaloneClients(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	// Get standalone config
	config, err := db.GetStandaloneConfig()
	if err != nil {
		return fmt.Errorf("not in standalone mode: %w", err)
	}

	if !config.IsServer {
		return fmt.Errorf("this command is only available on standalone server instances")
	}

	// Get registered clients
	clients, err := db.ListRegisteredClients()
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	if len(clients) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No registered clients")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "Clients can connect using: clonr standalone connect <key>")

		return nil
	}

	// Filter if not showing all
	var displayClients = clients
	if !clientsShowAll {
		displayClients = nil

		for _, c := range clients {
			if c.Status != "suspended" && c.Status != "revoked" {
				displayClients = append(displayClients, c)
			}
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Registered clients (%d):\n\n", len(displayClients))

	for _, client := range displayClients {
		statusIcon := "●"

		switch client.Status {
		case "active":
			statusIcon = "●" // Green in terminal with color support
		case "suspended":
			statusIcon = "○"
		case "revoked":
			statusIcon = "✕"
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", statusIcon, client.ClientName)
		_, _ = fmt.Fprintf(os.Stdout, "    ID: %s\n", client.ClientID[:8])
		_, _ = fmt.Fprintf(os.Stdout, "    Key Hint: %s\n", client.KeyHint)
		_, _ = fmt.Fprintf(os.Stdout, "    Status: %s\n", client.Status)
		_, _ = fmt.Fprintf(os.Stdout, "    Machine: %s (%s/%s)\n",
			client.MachineInfo.Hostname,
			client.MachineInfo.OS,
			client.MachineInfo.Arch)
		_, _ = fmt.Fprintf(os.Stdout, "    Registered: %s\n", client.RegisteredAt.Format("2006-01-02 15:04:05"))
		_, _ = fmt.Fprintf(os.Stdout, "    Last Seen: %s\n", client.LastSeenAt.Format("2006-01-02 15:04:05"))

		_, _ = fmt.Fprintf(os.Stdout, "    Sync Count: %d\n", client.SyncCount)
		if client.LastIP != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Last IP: %s\n", client.LastIP)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	// Also show pending if any
	pending, _ := db.ListPendingRegistrations()
	if len(pending) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Pending registrations (%d):\n", len(pending))
		for _, p := range pending {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s) - awaiting key entry\n",
				p.ClientName, p.ClientID[:8])
		}

		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "To accept a pending client: clonr standalone accept")
	}

	return nil
}
