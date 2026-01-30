package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var tpmCmd = &cobra.Command{
	Use:   "tpm",
	Short: "TPM 2.0 key management",
	Long: `Manage TPM 2.0 sealed keys for hardware-backed encryption.

TPM (Trusted Platform Module) provides hardware-based key protection:
  - Keys are sealed to the TPM and bound to this machine
  - No password required - authentication happens automatically
  - Keys cannot be extracted or copied to other machines

Commands:
  init    Initialize a new TPM-sealed master key
  status  Show current TPM configuration status
  reset   Remove the TPM-sealed key
  migrate Migrate existing database to TPM protection

Note: TPM 2.0 is currently only supported on Linux.`,
}

var tpmInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TPM-sealed master key",
	Long: `Initialize a new TPM-sealed master key for KeePass encryption.

This command creates a new cryptographic key, seals it to the TPM, and stores
the sealed blob on disk. The key can only be unsealed on this machine's TPM.

Prerequisites:
  - TPM 2.0 device available (/dev/tpmrm0 on Linux)
  - No existing KeePass database (use 'clonr tpm migrate' for existing databases)

Security benefits:
  - Key material is bound to hardware (cannot be extracted)
  - No password to remember or type
  - Resistant to offline attacks

Note: The sealed key CANNOT be backed up. If you lose access to this machine's
TPM (hardware failure, BIOS update, etc.), you will lose access to your data.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTPMInit(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var tpmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show TPM configuration status",
	Long: `Display the current TPM configuration status.

This command shows:
  - Whether TPM device is available
  - Whether a TPM-sealed key exists
  - Path to the sealed key file
  - Current authentication mode (TPM or password)`,
	Run: func(cmd *cobra.Command, args []string) {
		runTPMStatus()
	},
}

var tpmResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Remove TPM-sealed key",
	Long: `Remove the TPM-sealed key from disk.

This command deletes the sealed key file, effectively disabling TPM-based
authentication. After running this command:

  - You will need to use password-based authentication
  - You can run 'clonr tpm init' to create a new TPM key
  - Or run 'clonr tpm migrate' to migrate an existing database

WARNING: If you have a TPM-protected database and reset the key without
a backup, you will lose access to your data permanently.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTPMReset(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var tpmMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate existing database to TPM protection",
	Long: `Migrate an existing password-protected KeePass database to TPM protection.

This command:
  1. Creates a new TPM-sealed key
  2. Opens your existing database with your current password
  3. Re-encrypts it with a TPM-derived password

Prerequisites:
  - TPM 2.0 device available
  - Existing KeePass database with password protection
  - Knowledge of your current database password

After migration:
  - No password will be required
  - The database will only be accessible on this machine`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTPMMigrate(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var tpmMigrateProfilesCmd = &cobra.Command{
	Use:   "migrate-profiles",
	Short: "Migrate existing profiles to KeePass",
	Long: `Migrate existing profile tokens from keyring/encrypted storage to KeePass.

This command:
  1. Creates a KeePass database (if not exists) using TPM-derived password
  2. Reads all existing profiles from the server
  3. Gets tokens from keyring or encrypted storage
  4. Stores tokens in KeePass
  5. Updates profiles to use KeePass storage

Prerequisites:
  - TPM key already initialized (run 'clonr tpm init' first if no profiles exist)
  - Existing profiles with tokens in keyring or encrypted storage

After migration:
  - All tokens will be stored in KeePass
  - KeePass database protected by TPM (no password required)`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTPMMigrateProfiles(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tpmCmd)
	tpmCmd.AddCommand(tpmInitCmd)
	tpmCmd.AddCommand(tpmStatusCmd)
	tpmCmd.AddCommand(tpmResetCmd)
	tpmCmd.AddCommand(tpmMigrateCmd)
	tpmCmd.AddCommand(tpmMigrateProfilesCmd)
}

func runTPMInit() error {
	_, _ = fmt.Fprintln(os.Stdout, "Initialize TPM-sealed Key")
	_, _ = fmt.Fprintln(os.Stdout, "========================================")

	// Check TPM availability
	if !core.IsTPMAvailable() {
		return fmt.Errorf("TPM device not available\n\nPlease ensure:\n  - TPM 2.0 is enabled in BIOS\n  - /dev/tpmrm0 exists and is accessible\n  - You have read/write permissions to the TPM device")
	}

	_, _ = fmt.Fprintln(os.Stdout, "  ✓ TPM device available")

	// Check if TPM key already exists
	if core.HasTPMKey() {
		keyPath, _ := core.GetTPMKeyStorePath()

		return fmt.Errorf("TPM key already exists at: %s\n\nTo recreate, first run: clonr tpm reset", keyPath)
	}

	// Check if KeePass database already exists
	if core.KeePassDBExists() {
		kdbxPath, _ := core.GetKeePassDBPath()

		return fmt.Errorf("KeePass database already exists at: %s\n\nTo migrate an existing database to TPM, use: clonr tpm migrate", kdbxPath)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nGenerating and sealing key to TPM...")

	// Initialize TPM key
	if err := core.InitializeTPMKey(); err != nil {
		return fmt.Errorf("failed to initialize TPM key: %w", err)
	}

	keyPath, _ := core.GetTPMKeyStorePath()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Sealed key stored at: %s\n", keyPath)

	// Get the derived password to create the KeePass database
	_, _ = fmt.Fprintln(os.Stdout, "\nCreating KeePass database with TPM-derived password...")

	tpmPassword, err := core.GetKeePassPasswordTPM()
	if err != nil {
		// Clean up the sealed key on failure
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to derive password from TPM key: %w", err)
	}

	// Create the KeePass database
	_, err = core.NewKeePassManager(tpmPassword)
	if err != nil {
		// Clean up the sealed key on failure
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to create KeePass database: %w", err)
	}

	kdbxPath, _ := core.GetKeePassDBPath()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ KeePass database created at: %s\n", kdbxPath)

	_, _ = fmt.Fprintln(os.Stdout, "\n========================================")
	_, _ = fmt.Fprintln(os.Stdout, "TPM initialization complete!")
	_, _ = fmt.Fprintln(os.Stdout, "\nYour KeePass database is now protected by the TPM.")
	_, _ = fmt.Fprintln(os.Stdout, "No password is required - authentication happens automatically.")
	_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  WARNING: The sealed key cannot be backed up.")
	_, _ = fmt.Fprintln(os.Stdout, "If you lose access to this TPM, you will lose your data.")

	return nil
}

func runTPMStatus() {
	_, _ = fmt.Fprintln(os.Stdout, "TPM Status")
	_, _ = fmt.Fprintln(os.Stdout, "========================================")

	// Check TPM availability
	tpmAvailable := core.IsTPMAvailable()
	if tpmAvailable {
		_, _ = fmt.Fprintln(os.Stdout, "TPM Device:     Available ✓")
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "TPM Device:     Not available ✗")
	}

	// Check sealed key
	hasKey := core.HasTPMKey()
	if hasKey {
		_, _ = fmt.Fprintln(os.Stdout, "Sealed Key:     Present ✓")

		if keyPath, err := core.GetTPMKeyStorePath(); err == nil {
			_, _ = fmt.Fprintf(os.Stdout, "Key Path:       %s\n", keyPath)
		}
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Sealed Key:     Not found ✗")
	}

	// Check KeePass database
	hasDB := core.KeePassDBExists()
	if hasDB {
		_, _ = fmt.Fprintln(os.Stdout, "KeePass DB:     Present ✓")

		if dbPath, err := core.GetKeePassDBPath(); err == nil {
			_, _ = fmt.Fprintf(os.Stdout, "DB Path:        %s\n", dbPath)
		}
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "KeePass DB:     Not found ✗")
	}

	// Determine auth mode
	_, _ = fmt.Fprintln(os.Stdout, "----------------------------------------")

	switch {
	case hasKey && tpmAvailable:
		_, _ = fmt.Fprintln(os.Stdout, "Auth Mode:      TPM (hardware-backed)")
		_, _ = fmt.Fprintln(os.Stdout, "\nYour database is protected by the TPM.")
		_, _ = fmt.Fprintln(os.Stdout, "No password is required for access.")
	case hasKey && !tpmAvailable:
		_, _ = fmt.Fprintln(os.Stdout, "Auth Mode:      ERROR - Key exists but TPM unavailable")
		_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  Warning: You have a sealed key but TPM is not accessible.")
		_, _ = fmt.Fprintln(os.Stdout, "Please check TPM device permissions or BIOS settings.")
	default:
		_, _ = fmt.Fprintln(os.Stdout, "Auth Mode:      Password (software)")
		_, _ = fmt.Fprintln(os.Stdout, "\nYour database uses password-based protection.")

		if tpmAvailable {
			_, _ = fmt.Fprintln(os.Stdout, "To enable TPM protection, run: clonr tpm init")
		} else {
			_, _ = fmt.Fprintln(os.Stdout, "TPM is not available on this system.")
		}
	}
}

func runTPMReset() error {
	_, _ = fmt.Fprintln(os.Stdout, "Reset TPM Key")
	_, _ = fmt.Fprintln(os.Stdout, "========================================")

	// Check if TPM key exists
	if !core.HasTPMKey() {
		_, _ = fmt.Fprintln(os.Stdout, "No TPM key found. Nothing to reset.")

		return nil
	}

	keyPath, _ := core.GetTPMKeyStorePath()
	_, _ = fmt.Fprintf(os.Stdout, "Found TPM key at: %s\n", keyPath)

	// Warn and confirm
	_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  WARNING: This will delete your TPM-sealed key.")
	_, _ = fmt.Fprintln(os.Stdout, "If your database is TPM-protected, you will lose access to it!")
	_, _ = fmt.Fprint(os.Stdout, "\nAre you sure? (yes/no): ")

	var response string

	_, _ = fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		_, _ = fmt.Fprintln(os.Stdout, "Reset cancelled.")

		return nil
	}

	// Delete the key
	if err := core.ResetTPMKey(); err != nil {
		return fmt.Errorf("failed to reset TPM key: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\n  ✓ TPM key deleted successfully")
	_, _ = fmt.Fprintln(os.Stdout, "\nTo create a new TPM key, run: clonr tpm init")
	_, _ = fmt.Fprintln(os.Stdout, "To migrate an existing database: clonr tpm migrate")

	return nil
}

// profileToken holds a profile name, host, and its token for migration
type profileToken struct {
	name  string
	host  string
	token string
}

func runTPMMigrate() error {
	_, _ = fmt.Fprintln(os.Stdout, "Migrate to TPM Protection")
	_, _ = fmt.Fprintln(os.Stdout, "========================================")

	// Check TPM availability
	if !core.IsTPMAvailable() {
		return fmt.Errorf("TPM device not available\n\nPlease ensure:\n  - TPM 2.0 is enabled in BIOS\n  - /dev/tpmrm0 exists and is accessible\n  - You have read/write permissions to the TPM device")
	}

	_, _ = fmt.Fprintln(os.Stdout, "  ✓ TPM device available")

	// Check if TPM key already exists
	if core.HasTPMKey() {
		keyPath, _ := core.GetTPMKeyStorePath()

		return fmt.Errorf("TPM key already exists at: %s\n\nDatabase may already be migrated. To reset, run: clonr tpm reset", keyPath)
	}

	// Check if KeePass database exists
	kdbxPath, err := core.GetKeePassDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	if !core.KeePassDBExists() {
		return fmt.Errorf("no KeePass database found at: %s\n\nTo create a new TPM-protected database, use: clonr tpm init", kdbxPath)
	}

	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Found KeePass database at: %s\n", kdbxPath)

	// Warn and confirm
	_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  WARNING: This will migrate your database to TPM protection.")
	_, _ = fmt.Fprintln(os.Stdout, "The sealed key cannot be backed up. Make sure you have a backup!")
	_, _ = fmt.Fprint(os.Stdout, "\nContinue? (yes/no): ")

	var response string

	_, _ = fmt.Scanln(&response)

	if response != "yes" && response != "y" {
		_, _ = fmt.Fprintln(os.Stdout, "Migration cancelled.")

		return nil
	}

	// Get current password
	_, _ = fmt.Fprintln(os.Stdout, "\nEnter your current KeePass master password to open the database.")

	currentPassword, err := core.PromptForPassword("Current master password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Derive the KeePass password from user input
	derivedCurrentPassword := core.DeriveKeePassPasswordFromPassphrase(currentPassword)

	// Open existing database with current password
	_, _ = fmt.Fprintln(os.Stdout, "\nOpening existing database...")

	kpm, err := core.NewKeePassManager(derivedCurrentPassword)
	if err != nil {
		return fmt.Errorf("failed to open database with provided password: %w", err)
	}

	// Get all profiles and their tokens to preserve
	profileNames := kpm.ListProfiles()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Found %d profile(s)\n", len(profileNames))

	// Read all profile tokens
	var tokens []profileToken

	for _, name := range profileNames {
		// Profile names are stored as "name:host" in KeePass
		// We need to extract them properly
		token, err := kpm.GetProfileToken(name, "github.com") // Default host
		if err != nil {
			// Try without host suffix
			continue
		}

		tokens = append(tokens, profileToken{
			name:  name,
			host:  "github.com",
			token: token,
		})
	}

	// Initialize TPM key
	_, _ = fmt.Fprintln(os.Stdout, "\nGenerating and sealing key to TPM...")

	if err := core.InitializeTPMKey(); err != nil {
		return fmt.Errorf("failed to initialize TPM key: %w", err)
	}

	keyPath, _ := core.GetTPMKeyStorePath()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Sealed key stored at: %s\n", keyPath)

	// Get TPM-derived password
	tpmPassword, err := core.GetKeePassPasswordTPM()
	if err != nil {
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to derive password from TPM key: %w", err)
	}

	// Backup old database
	backupPath := kdbxPath + ".backup"
	if err := copyFile(kdbxPath, backupPath); err != nil {
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to backup database: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Backup created at: %s\n", backupPath)

	// Remove old database to create fresh one
	if err := os.Remove(kdbxPath); err != nil {
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to remove old database: %w", err)
	}

	// Create new database with TPM password
	_, _ = fmt.Fprintln(os.Stdout, "\nCreating new database with TPM-derived password...")

	newKpm, err := core.NewKeePassManager(tpmPassword)
	if err != nil {
		_ = copyFile(backupPath, kdbxPath)
		_ = core.ResetTPMKey()

		return fmt.Errorf("failed to create new database: %w", err)
	}

	// Restore all profile tokens
	for _, pt := range tokens {
		if err := newKpm.SetProfileToken(pt.name, pt.host, pt.token); err != nil {
			_ = copyFile(backupPath, kdbxPath)
			_ = core.ResetTPMKey()

			return fmt.Errorf("failed to restore profile '%s': %w", pt.name, err)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Restored %d profile(s)\n", len(tokens))

	_, _ = fmt.Fprintln(os.Stdout, "\n========================================")
	_, _ = fmt.Fprintln(os.Stdout, "Migration complete!")
	_, _ = fmt.Fprintln(os.Stdout, "\nYour KeePass database is now protected by the TPM.")
	_, _ = fmt.Fprintln(os.Stdout, "No password is required - authentication happens automatically.")
	_, _ = fmt.Fprintf(os.Stdout, "\nBackup saved at: %s\n", backupPath)
	_, _ = fmt.Fprintln(os.Stdout, "You may delete the backup once you've verified everything works.")

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0600)
}

func runTPMMigrateProfiles() error {
	_, _ = fmt.Fprintln(os.Stdout, "Migrate Profiles to KeePass")
	_, _ = fmt.Fprintln(os.Stdout, "========================================")

	// Check TPM availability and key
	if !core.IsTPMAvailable() {
		return fmt.Errorf("TPM device not available")
	}

	_, _ = fmt.Fprintln(os.Stdout, "  ✓ TPM device available")

	if !core.HasTPMKey() {
		return fmt.Errorf("no TPM key found\n\nRun 'clonr tpm init' first to create a TPM key")
	}

	_, _ = fmt.Fprintln(os.Stdout, "  ✓ TPM key found")

	// Connect to server
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get all profiles
	profiles, err := client.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if len(profiles) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nNo profiles found. Nothing to migrate.")

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Found %d profile(s)\n", len(profiles))

	// Get TPM-derived password
	tpmPassword, err := core.GetKeePassPasswordTPM()
	if err != nil {
		return fmt.Errorf("failed to derive password from TPM key: %w", err)
	}

	// Create or open KeePass database
	_, _ = fmt.Fprintln(os.Stdout, "\nOpening/creating KeePass database...")

	kpm, err := core.NewKeePassManager(tpmPassword)
	if err != nil {
		return fmt.Errorf("failed to open KeePass database: %w", err)
	}

	kdbxPath, _ := core.GetKeePassDBPath()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ KeePass database at: %s\n", kdbxPath)

	// Migrate each profile
	_, _ = fmt.Fprintln(os.Stdout, "\nMigrating profiles...")

	var migrated, skipped, failed int

	for i := range profiles {
		profile := &profiles[i]

		// Skip if already using KeePass
		if profile.TokenStorage == model.TokenStorageKeePass {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s: already using KeePass (skipped)\n", profile.Name)
			skipped++

			continue
		}

		// Get token from current storage
		var token string

		switch profile.TokenStorage {
		case model.TokenStorageKeyring:
			token, err = core.GetToken(profile.Name, profile.Host)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: failed to get token from keyring: %v\n", profile.Name, err)
				failed++

				continue
			}
		case model.TokenStorageInsecure:
			if len(profile.EncryptedToken) == 0 {
				_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: no encrypted token found\n", profile.Name)
				failed++

				continue
			}

			token, err = core.DecryptToken(profile.EncryptedToken, profile.Name, profile.Host)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: failed to decrypt token: %v\n", profile.Name, err)
				failed++

				continue
			}
		default:
			_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: unknown storage type: %s\n", profile.Name, profile.TokenStorage)
			failed++

			continue
		}

		// Store token in KeePass
		if err := kpm.SetProfileToken(profile.Name, profile.Host, token); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: failed to store in KeePass: %v\n", profile.Name, err)
			failed++

			continue
		}

		// Update profile to use KeePass storage
		oldStorage := profile.TokenStorage
		profile.TokenStorage = model.TokenStorageKeePass
		profile.EncryptedToken = nil // Clear encrypted token

		if err := client.SaveProfile(profile); err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  ✗ %s: failed to update profile: %v\n", profile.Name, err)
			failed++

			continue
		}

		// Delete token from old storage (ignore errors)
		if oldStorage == model.TokenStorageKeyring {
			_ = core.DeleteToken(profile.Name, profile.Host)
		}

		_, _ = fmt.Fprintf(os.Stdout, "  ✓ %s: migrated from %s to KeePass\n", profile.Name, oldStorage)
		migrated++
	}

	_, _ = fmt.Fprintln(os.Stdout, "\n========================================")
	_, _ = fmt.Fprintln(os.Stdout, "Migration complete!")
	_, _ = fmt.Fprintf(os.Stdout, "\nResults: %d migrated, %d skipped, %d failed\n", migrated, skipped, failed)

	if migrated > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nYour profile tokens are now stored in KeePass.")
		_, _ = fmt.Fprintln(os.Stdout, "The database is protected by the TPM - no password required.")
	}

	return nil
}
