package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var (
	acceptClientID string
	acceptListOnly bool
)

var standaloneAcceptCmd = &cobra.Command{
	Use:   "accept [display-key]",
	Short: "Accept a pending client connection",
	Long: `Accept a pending client connection by entering their encryption key.

When a client connects using 'clonr standalone connect', they generate an
encryption key and display it on their screen. You must enter that key here
to complete the registration.

The encryption key is used to:
- Encrypt sensitive data (tokens, credentials) specifically for that client
- Ensure only that client can decrypt their own sensitive data
- Allow normal storage of non-sensitive data (repositories, workspaces)

Examples:
  # List pending client connections
  clonr standalone accept --list

  # Accept a specific client
  clonr standalone accept --client abc12345

  # Accept the most recent pending client
  clonr standalone accept`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStandaloneAccept,
}

func init() {
	standaloneCmd.AddCommand(standaloneAcceptCmd)

	standaloneAcceptCmd.Flags().StringVarP(&acceptClientID, "client", "c", "", "Client ID to accept (first 8 chars)")
	standaloneAcceptCmd.Flags().BoolVar(&acceptListOnly, "list", false, "List pending clients without accepting")
}

func runStandaloneAccept(_ *cobra.Command, args []string) error {
	db := store.GetDB()

	// Get standalone config
	config, err := db.GetStandaloneConfig()
	if err != nil {
		return fmt.Errorf("not in standalone mode: %w", err)
	}

	if !config.IsServer {
		return fmt.Errorf("this command is only available on standalone server instances")
	}

	// Get pending registrations
	pending, err := db.ListPendingRegistrations()
	if err != nil {
		return fmt.Errorf("failed to list pending registrations: %w", err)
	}

	// List mode
	if acceptListOnly {
		if len(pending) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "No pending client registrations")
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "Pending client registrations (%d):\n\n", len(pending))
		for _, reg := range pending {
			_, _ = fmt.Fprintf(os.Stdout, "  Client ID: %s\n", reg.ClientID[:8])
			_, _ = fmt.Fprintf(os.Stdout, "    Name: %s\n", reg.ClientName)
			_, _ = fmt.Fprintf(os.Stdout, "    Machine: %s (%s/%s)\n",
				reg.MachineInfo.Hostname, reg.MachineInfo.OS, reg.MachineInfo.Arch)
			_, _ = fmt.Fprintf(os.Stdout, "    Clonr Version: %s\n", reg.MachineInfo.ClonrVersion)
			_, _ = fmt.Fprintf(os.Stdout, "    Initiated: %s\n", reg.InitiatedAt.Format("2006-01-02 15:04:05"))
			_, _ = fmt.Fprintf(os.Stdout, "    State: %s\n", reg.State)
			_, _ = fmt.Fprintln(os.Stdout)
		}

		_, _ = fmt.Fprintln(os.Stdout, "To accept a client: clonr standalone accept --client <id>")
		return nil
	}

	// Find the client to accept
	var target *standalone.ClientRegistration

	if acceptClientID != "" {
		// Find by ID prefix
		for i := range pending {
			if strings.HasPrefix(pending[i].ClientID, acceptClientID) {
				target = pending[i]
				break
			}
		}
		if target == nil {
			return fmt.Errorf("no pending client found with ID prefix: %s", acceptClientID)
		}
	} else if len(pending) == 1 {
		// Auto-select if only one pending
		target = pending[0]
	} else if len(pending) == 0 {
		return fmt.Errorf("no pending client registrations")
	} else {
		// Multiple pending, must specify
		_, _ = fmt.Fprintf(os.Stderr, "Multiple pending registrations (%d). Specify --client <id>:\n\n", len(pending))
		for _, reg := range pending {
			_, _ = fmt.Fprintf(os.Stderr, "  %s - %s (%s)\n",
				reg.ClientID[:8], reg.ClientName, reg.MachineInfo.Hostname)
		}
		return fmt.Errorf("specify --client to choose which client to accept")
	}

	// Display client info
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════╗")
	_, _ = fmt.Fprintln(os.Stdout, "║                    CLIENT REGISTRATION                       ║")
	_, _ = fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════╣")
	_, _ = fmt.Fprintf(os.Stdout, "║  Client ID: %-48s ║\n", target.ClientID[:8])
	_, _ = fmt.Fprintf(os.Stdout, "║  Name:      %-48s ║\n", truncateString(target.ClientName, 48))
	_, _ = fmt.Fprintf(os.Stdout, "║  Hostname:  %-48s ║\n", truncateString(target.MachineInfo.Hostname, 48))
	_, _ = fmt.Fprintf(os.Stdout, "║  Platform:  %-48s ║\n",
		fmt.Sprintf("%s/%s", target.MachineInfo.OS, target.MachineInfo.Arch))
	_, _ = fmt.Fprintf(os.Stdout, "║  Version:   %-48s ║\n", target.MachineInfo.ClonrVersion)
	_, _ = fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════╝")
	_, _ = fmt.Fprintln(os.Stdout)

	// Get the encryption key from user
	var displayKey string
	if len(args) > 0 {
		displayKey = args[0]
	} else {
		_, _ = fmt.Fprint(os.Stderr, "Enter the encryption key displayed on the client: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}
		displayKey = strings.TrimSpace(input)
	}

	if displayKey == "" {
		return fmt.Errorf("encryption key is required")
	}

	// Create server handshake manager and register the client
	serverHandshake := standalone.NewServerHandshake()

	// Re-add the pending client to the handshake manager
	// (in a real implementation, this would be persisted)
	_, _ = serverHandshake.InitiateHandshake(target)

	// Register the client with the provided key
	registeredClient, err := serverHandshake.RegisterClient(target.ClientID, displayKey)
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}

	// Save the registered client
	if err := db.SaveRegisteredClient(registeredClient); err != nil {
		return fmt.Errorf("failed to save client registration: %w", err)
	}

	// Remove from pending
	if err := db.RemovePendingRegistration(target.ClientID); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to remove pending registration: %v\n", err)
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════╗")
	_, _ = fmt.Fprintln(os.Stdout, "║                  CLIENT REGISTERED                           ║")
	_, _ = fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════╣")
	_, _ = fmt.Fprintf(os.Stdout, "║  Client: %-51s ║\n", registeredClient.ClientName)
	_, _ = fmt.Fprintf(os.Stdout, "║  Key Hint: %-49s ║\n", registeredClient.KeyHint)
	_, _ = fmt.Fprintf(os.Stdout, "║  Status: %-51s ║\n", registeredClient.Status)
	_, _ = fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════╝")
	_, _ = fmt.Fprintln(os.Stdout)

	_, _ = fmt.Fprintln(os.Stdout, "The client can now sync data with this instance.")
	_, _ = fmt.Fprintln(os.Stdout, "Sensitive data (tokens, credentials) will be encrypted with the client's key.")
	_, _ = fmt.Fprintln(os.Stdout, "Repository data will be stored normally for easy access.")

	return nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
