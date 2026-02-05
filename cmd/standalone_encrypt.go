package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var standaloneEncryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Configure encryption for standalone mode",
	Long: `Configure the encryption key for standalone sync mode.

When standalone mode is enabled, all synced data is encrypted. The server
requires an encryption key to:
  - Encrypt data before sending to connected clients
  - Store synced data securely

This key is separate from the standalone API key. It's used specifically
for encrypting the actual data content.

Examples:
  # Set up encryption key (prompts for password)
  clonr standalone encrypt setup

  # Check encryption status
  clonr standalone encrypt status

  # Change encryption key (requires current key)
  clonr standalone encrypt rotate`,
}

var standaloneEncryptSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up encryption key for standalone mode",
	Long: `Set up the encryption key that will be used to encrypt all synced data.

You will be prompted to enter a password. This password is used to derive
the encryption key. Remember this password - you'll need it to:
  - Decrypt synced data on destination instances
  - Rotate the encryption key later

The password must be at least 8 characters long.`,
	RunE: runStandaloneEncryptSetup,
}

var standaloneEncryptStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show encryption configuration status",
	RunE:  runStandaloneEncryptStatus,
}

func init() {
	standaloneCmd.AddCommand(standaloneEncryptCmd)
	standaloneEncryptCmd.AddCommand(standaloneEncryptSetupCmd)
	standaloneEncryptCmd.AddCommand(standaloneEncryptStatusCmd)
}

func runStandaloneEncryptSetup(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	// Check if already configured
	existing, err := db.GetServerEncryptionConfig()
	if err != nil {
		return fmt.Errorf("failed to check existing config: %w", err)
	}

	if existing != nil && existing.Enabled {
		return fmt.Errorf("encryption is already configured (use 'clonr standalone encrypt rotate' to change the key)")
	}

	// Get password
	password, err := readArchivePassword("Enter encryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Confirm password
	confirm, err := readArchivePassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Setup encryption
	keyManager := standalone.NewEncryptionKeyManager()

	config, err := keyManager.SetupKey(password)
	if err != nil {
		return fmt.Errorf("failed to setup encryption: %w", err)
	}

	// Save config
	if err := db.SaveServerEncryptionConfig(config); err != nil {
		return fmt.Errorf("failed to save encryption config: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Encryption configured successfully!")
	_, _ = fmt.Fprintf(os.Stdout, "  Key hint: %s\n", config.KeyHint)
	_, _ = fmt.Fprintf(os.Stdout, "  Configured at: %s\n", config.ConfiguredAt.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "IMPORTANT: Remember this password! You'll need it to:")
	_, _ = fmt.Fprintln(os.Stdout, "  - Decrypt synced data on destination instances")
	_, _ = fmt.Fprintln(os.Stdout, "  - Rotate the encryption key in the future")

	return nil
}

func runStandaloneEncryptStatus(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	config, err := db.GetServerEncryptionConfig()
	if err != nil {
		return fmt.Errorf("failed to get encryption config: %w", err)
	}

	if config == nil || !config.Enabled {
		_, _ = fmt.Fprintln(os.Stdout, "Encryption: not configured")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "To configure encryption, run: clonr standalone encrypt setup")

		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, "Encryption: enabled")
	_, _ = fmt.Fprintf(os.Stdout, "  Key hint: %s\n", config.KeyHint)
	_, _ = fmt.Fprintf(os.Stdout, "  Configured at: %s\n", config.ConfiguredAt.Format("2006-01-02 15:04:05"))

	return nil
}
