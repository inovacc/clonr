package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
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

Note: TPM 2.0 is currently only supported on Linux.`,
}

var tpmInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TPM-sealed master key",
	Long: `Initialize a new TPM-sealed master key for encryption.

This command creates a new cryptographic key, seals it to the TPM, and stores
the sealed blob on disk. The key can only be unsealed on this machine's TPM.

Prerequisites:
  - TPM 2.0 device available (/dev/tpmrm0 on Linux)
  - No existing TPM-sealed key (use 'clonr tpm reset' first)

Security benefits:
  - Key material is bound to hardware (cannot be extracted)
  - No password to remember or type
  - Resistant to offline attacks

Note: The sealed key CANNOT be backed up. If you lose access to this machine's
TPM (hardware failure, BIOS update, etc.), encrypted data will be inaccessible.`,
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
  - Current encryption mode (TPM or file-based)`,
	Run: func(cmd *cobra.Command, args []string) {
		runTPMStatus()
	},
}

var tpmResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Remove TPM-sealed key",
	Long: `Remove the TPM-sealed key from disk.

This command deletes the sealed key file, effectively disabling TPM-based
encryption. After running this command:

  - Encryption will fall back to file-based master key
  - You can run 'clonr tpm init' to create a new TPM key

WARNING: If you have data encrypted with the TPM-sealed key and reset it,
that data will become inaccessible.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTPMReset(); err != nil {
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

	// Check if file-based key exists (indicates existing encrypted data)
	if core.HasFileBasedKey() {
		_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  WARNING: Existing file-based encryption key detected!")
		_, _ = fmt.Fprintln(os.Stdout, "If you have existing profiles with encrypted tokens, they will")
		_, _ = fmt.Fprintln(os.Stdout, "become INACCESSIBLE after switching to TPM encryption.")
		_, _ = fmt.Fprintln(os.Stdout, "\nBefore proceeding, you should:")
		_, _ = fmt.Fprintln(os.Stdout, "  1. Remove all existing profiles: clonr profile list")
		_, _ = fmt.Fprintln(os.Stdout, "  2. Re-add them after TPM initialization")
		_, _ = fmt.Fprint(os.Stdout, "\nContinue anyway? (yes/no): ")

		var response string

		_, _ = fmt.Scanln(&response)

		if response != "yes" && response != "y" {
			_, _ = fmt.Fprintln(os.Stdout, "TPM initialization cancelled.")

			return nil
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nGenerating and sealing key to TPM...")

	// Initialize TPM key
	if err := core.InitializeTPMKey(); err != nil {
		return fmt.Errorf("failed to initialize TPM key: %w", err)
	}

	keyPath, _ := core.GetTPMKeyStorePath()
	_, _ = fmt.Fprintf(os.Stdout, "  ✓ Sealed key stored at: %s\n", keyPath)

	_, _ = fmt.Fprintln(os.Stdout, "\n========================================")
	_, _ = fmt.Fprintln(os.Stdout, "TPM initialization complete!")
	_, _ = fmt.Fprintln(os.Stdout, "\nYour encryption keys are now protected by the TPM.")
	_, _ = fmt.Fprintln(os.Stdout, "Token encryption will automatically use the TPM-sealed key.")
	_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  WARNING: The sealed key cannot be backed up.")
	_, _ = fmt.Fprintln(os.Stdout, "If you lose access to this TPM, encrypted data will be lost.")

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

	// Determine encryption mode
	_, _ = fmt.Fprintln(os.Stdout, "----------------------------------------")

	switch {
	case hasKey && tpmAvailable:
		_, _ = fmt.Fprintln(os.Stdout, "Encryption:     TPM (hardware-backed)")
		_, _ = fmt.Fprintln(os.Stdout, "\nYour tokens are protected by the TPM.")
		_, _ = fmt.Fprintln(os.Stdout, "Encryption happens automatically without passwords.")
	case hasKey && !tpmAvailable:
		_, _ = fmt.Fprintln(os.Stdout, "Encryption:     ERROR - Key exists but TPM unavailable")
		_, _ = fmt.Fprintln(os.Stdout, "\n⚠️  Warning: You have a sealed key but TPM is not accessible.")
		_, _ = fmt.Fprintln(os.Stdout, "Please check TPM device permissions or BIOS settings.")
	default:
		_, _ = fmt.Fprintln(os.Stdout, "Encryption:     File-based (software)")
		_, _ = fmt.Fprintln(os.Stdout, "\nYour tokens use file-based encryption.")

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
	_, _ = fmt.Fprintln(os.Stdout, "Data encrypted with this key will become inaccessible!")
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
	_, _ = fmt.Fprintln(os.Stdout, "\nEncryption will now use file-based keys.")
	_, _ = fmt.Fprintln(os.Stdout, "To create a new TPM key, run: clonr tpm init")

	return nil
}
