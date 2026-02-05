package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var (
	standaloneInitPort   int
	standaloneInitHost   string
	standaloneInitOutput string
)

var standaloneInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize standalone mode",
	Long: `Initialize standalone mode on this instance to enable data synchronization.

This command generates a standalone key that can be shared with other clonr
instances to establish a secure connection for syncing profiles, workspaces,
and repository metadata.

The generated key contains:
  - Instance ID (unique identifier for this instance)
  - Connection info (host, port)
  - API key (for authentication)
  - Refresh token (for session renewal)
  - Encryption key hint (for verification)

Examples:
  # Initialize with auto-detected IP
  clonr standalone init

  # Initialize with specific host and port
  clonr standalone init --host 192.168.1.100 --port 50052

  # Output key to a file
  clonr standalone init --output sync-key.json

  # Copy key directly (Linux/macOS)
  clonr standalone init | xclip -selection clipboard`,
	RunE: runStandaloneInit,
}

func init() {
	standaloneCmd.AddCommand(standaloneInitCmd)

	standaloneInitCmd.Flags().IntVarP(&standaloneInitPort, "port", "p", standalone.DefaultPort, "Port for the sync service")
	standaloneInitCmd.Flags().StringVar(&standaloneInitHost, "host", "", "Host address (auto-detected if not specified)")
	standaloneInitCmd.Flags().StringVarP(&standaloneInitOutput, "output", "o", "", "Output file for the key (default: stdout)")
}

func runStandaloneInit(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	// Check if standalone mode is already initialized
	existingConfig, err := db.GetStandaloneConfig()
	if err == nil && existingConfig != nil && existingConfig.Enabled {
		return fmt.Errorf("standalone mode is already initialized (use 'clonr standalone rotate' to generate a new key)")
	}

	// Auto-detect host if not specified
	host := standaloneInitHost
	if host == "" {
		detectedHost, err := standalone.GetLocalIP()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: could not detect local IP, using localhost: %v\n", err)
			host = "127.0.0.1"
		} else {
			host = detectedHost
		}
	}

	// Generate the standalone key and config
	key, config, err := standalone.GenerateStandaloneKey(host, standaloneInitPort)
	if err != nil {
		return fmt.Errorf("failed to generate standalone key: %w", err)
	}

	// Save the config to the database
	if err := db.SaveStandaloneConfig(config); err != nil {
		return fmt.Errorf("failed to save standalone configuration: %w", err)
	}

	// Serialize the key for output
	var output []byte
	if standaloneInitOutput != "" || isOutputToFile() {
		// Pretty print for file output
		output, err = json.MarshalIndent(key, "", "  ")
	} else {
		// Compact encoding for terminal (easy to copy)
		encoded, err := standalone.EncodeKeyForSharing(key)
		if err != nil {
			return fmt.Errorf("failed to encode key: %w", err)
		}

		output = []byte(encoded)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize key: %w", err)
	}

	// Output the key
	if standaloneInitOutput != "" {
		if err := os.WriteFile(standaloneInitOutput, output, 0600); err != nil {
			return fmt.Errorf("failed to write key to file: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stderr, "Standalone key written to: %s\n", standaloneInitOutput)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, string(output))
	}

	// Print info to stderr so it doesn't interfere with piping
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Standalone mode initialized successfully!")
	_, _ = fmt.Fprintf(os.Stderr, "  Instance ID: %s\n", key.InstanceID)
	_, _ = fmt.Fprintf(os.Stderr, "  Host: %s\n", key.Host)
	_, _ = fmt.Fprintf(os.Stderr, "  Port: %d\n", key.Port)
	_, _ = fmt.Fprintf(os.Stderr, "  Expires: %s\n", key.ExpiresAt.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Share this key with other instances to establish a connection.")
	_, _ = fmt.Fprintln(os.Stderr, "IMPORTANT: Keep this key secure - it grants access to sync your data.")
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Next steps:")
	_, _ = fmt.Fprintln(os.Stderr, "  1. Start the sync server: clonr server start")
	_, _ = fmt.Fprintln(os.Stderr, "  2. On the destination: clonr standalone connect '<key>'")

	return nil
}

// isOutputToFile checks if stdout is being redirected to a file
func isOutputToFile() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}
