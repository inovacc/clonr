package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var (
	decryptConnection string
	decryptAll        bool
	decryptList       bool
)

var standaloneDecryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt synced data from standalone instances",
	Long: `Decrypt data that was synced from standalone instances.

When you sync data from a standalone instance, it arrives encrypted and
is stored encrypted until you provide the decryption key. This keeps your
synced data secure even if someone gains access to your local storage.

To decrypt the data, you need the encryption password that was set up on
the source instance (using 'clonr standalone encrypt setup').

Examples:
  # List all encrypted (pending) data
  clonr standalone decrypt --list

  # Decrypt data from a specific connection
  clonr standalone decrypt --connection home-server

  # Decrypt all encrypted data
  clonr standalone decrypt --all`,
	RunE: runStandaloneDecrypt,
}

func init() {
	standaloneCmd.AddCommand(standaloneDecryptCmd)

	standaloneDecryptCmd.Flags().StringVarP(&decryptConnection, "connection", "c", "", "Connection name to decrypt")
	standaloneDecryptCmd.Flags().BoolVar(&decryptAll, "all", false, "Decrypt all encrypted data")
	standaloneDecryptCmd.Flags().BoolVar(&decryptList, "list", false, "List encrypted data without decrypting")
}

func runStandaloneDecrypt(_ *cobra.Command, _ []string) error {
	db := store.GetDB()

	// List mode
	if decryptList {
		encrypted, err := db.ListSyncedDataByState(standalone.SyncStateEncrypted)
		if err != nil {
			return fmt.Errorf("failed to list encrypted data: %w", err)
		}

		if len(encrypted) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "No encrypted data pending decryption")
			return nil
		}

		_, _ = fmt.Fprintf(os.Stdout, "Encrypted data awaiting decryption (%d items):\n\n", len(encrypted))

		// Group by connection
		byConnection := make(map[string][]standalone.SyncedData)
		for _, item := range encrypted {
			byConnection[item.ConnectionName] = append(byConnection[item.ConnectionName], item)
		}

		for connName, items := range byConnection {
			_, _ = fmt.Fprintf(os.Stdout, "Connection: %s\n", connName)
			for _, item := range items {
				_, _ = fmt.Fprintf(os.Stdout, "  - [%s] %s (synced: %s)\n",
					item.DataType, item.Name, item.SyncedAt.Format("2006-01-02 15:04"))
			}
			_, _ = fmt.Fprintln(os.Stdout)
		}

		_, _ = fmt.Fprintln(os.Stdout, "To decrypt, run: clonr standalone decrypt --connection <name>")
		return nil
	}

	// Decrypt mode
	if !decryptAll && decryptConnection == "" {
		return fmt.Errorf("specify --connection or --all (use --list to see encrypted data)")
	}

	// Get items to decrypt
	var toDecrypt []standalone.SyncedData
	var err error

	if decryptAll {
		toDecrypt, err = db.ListSyncedDataByState(standalone.SyncStateEncrypted)
	} else {
		toDecrypt, err = db.ListSyncedData(decryptConnection)
		// Filter to only encrypted items
		var filtered []standalone.SyncedData
		for _, item := range toDecrypt {
			if item.State == standalone.SyncStateEncrypted {
				filtered = append(filtered, item)
			}
		}
		toDecrypt = filtered
	}

	if err != nil {
		return fmt.Errorf("failed to list data: %w", err)
	}

	if len(toDecrypt) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No encrypted data to decrypt")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stderr, "Found %d items to decrypt\n", len(toDecrypt))

	// Get decryption password
	password, err := readArchivePassword("Enter decryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Derive key from password (using Argon2)
	// We need the salt from the connection or encrypted data
	// For now, use PBKDF2 which doesn't require stored salt
	salt, err := standalone.GenerateSalt() // This is a placeholder - in real impl, salt should come from connection
	if err != nil {
		return fmt.Errorf("failed to process key: %w", err)
	}
	key := standalone.DeriveKeyArgon2(password, salt)

	// Try to decrypt each item
	var (
		decrypted int
		failed    int
	)

	for _, item := range toDecrypt {
		_, decryptErr := standalone.DecryptSyncedData(&item, key)
		if decryptErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "  Failed: [%s] %s - %v\n", item.DataType, item.Name, decryptErr)
			failed++
			continue
		}

		// Update state to decrypted
		item.State = standalone.SyncStateDecrypted
		if saveErr := db.SaveSyncedData(&item); saveErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "  Warning: decrypted but failed to save [%s] %s: %v\n",
				item.DataType, item.Name, saveErr)
			continue
		}

		_, _ = fmt.Fprintf(os.Stdout, "  Decrypted: [%s] %s\n", item.DataType, item.Name)
		decrypted++
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "Results: %d decrypted, %d failed\n", decrypted, failed)

	if failed > 0 && decrypted == 0 {
		return fmt.Errorf("decryption failed - check your password")
	}

	return nil
}
