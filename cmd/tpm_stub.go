//go:build !linux

package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var tpmInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize TPM key + KeePass database",
	Long:  `Initialize a new TPM-sealed master key for KeePass encryption.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTPMNotSupported()
	},
}

var tpmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show TPM + KeePass status",
	Long:  `Display the current TPM configuration status.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTPMNotSupported()
	},
}

var tpmResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Remove TPM-sealed key",
	Long:  `Remove the TPM-sealed key from disk.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTPMNotSupported()
	},
}

var tpmMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate existing KeePass DB to TPM",
	Long:  `Migrate an existing password-protected KeePass database to TPM protection.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTPMNotSupported()
	},
}

var tpmMigrateProfilesCmd = &cobra.Command{
	Use:   "migrate-profiles",
	Short: "Migrate profiles from keyring to KeePass",
	Long:  `Migrate existing profile tokens from keyring/encrypted storage to KeePass.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTPMNotSupported()
	},
}

func init() {
	tpmCmd.AddCommand(tpmInitCmd)
	tpmCmd.AddCommand(tpmStatusCmd)
	tpmCmd.AddCommand(tpmResetCmd)
	tpmCmd.AddCommand(tpmMigrateCmd)
	tpmCmd.AddCommand(tpmMigrateProfilesCmd)
}

func printTPMNotSupported() {
	_, _ = fmt.Fprintf(os.Stderr, "TPM 2.0 is not supported on %s.\n", runtime.GOOS)
	_, _ = fmt.Fprintln(os.Stderr, "")
	_, _ = fmt.Fprintln(os.Stderr, "TPM hardware-backed encryption is currently only available on Linux.")
	_, _ = fmt.Fprintln(os.Stderr, "")
	_, _ = fmt.Fprintln(os.Stderr, "On this platform, you can use password-based KeePass protection instead:")
	_, _ = fmt.Fprintln(os.Stderr, "  - Profile tokens are stored securely in the system keyring")
	_, _ = fmt.Fprintln(os.Stderr, "  - Fallback to encrypted file storage if keyring is unavailable")

	os.Exit(1)
}
