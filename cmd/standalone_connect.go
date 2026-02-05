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
	connectName string
	connectFile string
)

var standaloneConnectCmd = &cobra.Command{
	Use:   "connect <standalone-key>",
	Short: "Connect to a standalone instance",
	Long: `Connect to a standalone instance to sync data.

This command initiates a handshake with a standalone server:

1. You provide the standalone key (from 'clonr standalone init' on server)
2. Your client sends identification info to the server
3. Server issues a challenge
4. Your client generates a secure encryption key
5. The key is displayed - you must enter it on the server
6. Once verified, the connection is established

The encryption key ensures that sensitive data (like tokens) synced from
the server can only be decrypted by you. Repositories are synced normally.

Examples:
  # Connect using a key string
  clonr standalone connect "CLONR-SYNC:..."

  # Connect using a key file
  clonr standalone connect --file standalone-key.json

  # Connect with a custom name
  clonr standalone connect --name "work-server" "CLONR-SYNC:..."`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStandaloneConnect,
}

func init() {
	standaloneCmd.AddCommand(standaloneConnectCmd)

	standaloneConnectCmd.Flags().StringVarP(&connectName, "name", "n", "", "Connection name (default: auto-generated)")
	standaloneConnectCmd.Flags().StringVarP(&connectFile, "file", "f", "", "Read key from file instead of argument")
}

func runStandaloneConnect(_ *cobra.Command, args []string) error {
	db := store.GetDB()

	// Get the standalone key
	var keyData string

	if connectFile != "" {
		data, err := os.ReadFile(connectFile)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		keyData = strings.TrimSpace(string(data))
	} else if len(args) > 0 {
		keyData = args[0]
	} else {
		return fmt.Errorf("provide a standalone key as argument or use --file")
	}

	// Parse the key
	key, err := standalone.DecodeSharedKey(keyData)
	if err != nil {
		return fmt.Errorf("invalid standalone key: %w", err)
	}

	// Validate the key
	if err := standalone.ValidateKey(key); err != nil {
		return fmt.Errorf("key validation failed: %w", err)
	}

	// Check if already connected to this instance
	existing, _ := db.ListStandaloneConnections()
	for _, conn := range existing {
		if conn.InstanceID == key.InstanceID {
			return fmt.Errorf("already connected to instance %s (connection: %s)", key.InstanceID[:8], conn.Name)
		}
	}

	// Generate connection name if not provided
	name := connectName
	if name == "" {
		name = fmt.Sprintf("server-%s", key.InstanceID[:8])
	}

	_, _ = fmt.Fprintf(os.Stderr, "Connecting to standalone instance...\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Host: %s:%d\n", key.Host, key.Port)
	_, _ = fmt.Fprintf(os.Stderr, "  Instance: %s\n", key.InstanceID[:8])
	_, _ = fmt.Fprintln(os.Stderr)

	// Start handshake
	machineInfo := standalone.GenerateMachineInfo("dev") // Version will be set at build time
	handshake := standalone.NewHandshake(name, machineInfo)

	_, _ = fmt.Fprintf(os.Stderr, "Starting handshake...\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Client ID: %s\n", handshake.GetRegistration().ClientID[:8])
	_, _ = fmt.Fprintln(os.Stderr)

	// Generate encryption key
	displayKey, err := handshake.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Display the key prominently
	_, _ = fmt.Fprintln(os.Stdout)

	printBoxHeader("ENCRYPTION KEY")

	_, _ = fmt.Fprintf(os.Stdout, "║%s║\n", centerString("", boxWidth-2))
	_, _ = fmt.Fprintf(os.Stdout, "║%s║\n", centerString(displayKey, boxWidth-2))
	_, _ = fmt.Fprintf(os.Stdout, "║%s║\n", centerString("", boxWidth-2))
	_, _ = fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════╣")
	_, _ = fmt.Fprintln(os.Stdout, "║  Enter this key on the server to complete registration.     ║")
	_, _ = fmt.Fprintln(os.Stdout, "║  Run on server: clonr standalone accept                     ║")

	printBoxFooter()

	_, _ = fmt.Fprintln(os.Stdout)

	// Wait for user confirmation
	_, _ = fmt.Fprint(os.Stderr, "Press Enter after entering the key on the server...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	// Get local password to encrypt stored credentials
	_, _ = fmt.Fprintln(os.Stderr)

	localPassword, err := readArchivePassword("Enter a local password to secure this connection: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(localPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	confirm, err := readArchivePassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if localPassword != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Create connection with the encryption key embedded
	conn, err := standalone.CreateConnection(name, key, localPassword)
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}

	// Store the client's encryption key (encrypted with local password)
	// This is the key that the server will use to encrypt sensitive data for us
	clientKey := handshake.GetFullKey()
	localSalt, _ := standalone.GenerateSalt()
	localDerivedKey := standalone.DeriveKeyArgon2(localPassword, localSalt)

	encryptedClientKey, err := standalone.EncryptWithKey(clientKey, localDerivedKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt client key: %w", err)
	}

	// Add client key to connection (we'll need to extend the type)
	// For now, store in the existing encrypted fields
	conn.APIKeyEncrypted = encryptedClientKey
	conn.LocalSalt = localSalt
	conn.SyncStatus = standalone.StatusConnected

	// Save connection
	if err := db.SaveStandaloneConnection(conn); err != nil {
		return fmt.Errorf("failed to save connection: %w", err)
	}

	handshake.Complete()

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Connection established successfully!")
	_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", conn.Name)
	_, _ = fmt.Fprintf(os.Stdout, "  Instance: %s\n", conn.InstanceID[:8])
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Next steps:")
	_, _ = fmt.Fprintln(os.Stdout, "  - Sync data: clonr standalone sync", conn.Name)
	_, _ = fmt.Fprintln(os.Stdout, "  - List synced data: clonr standalone decrypt --list")

	return nil
}
