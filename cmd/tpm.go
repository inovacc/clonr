package cmd

import (
	"github.com/spf13/cobra"
)

var tpmCmd = &cobra.Command{
	Use:   "tpm",
	Short: "TPM 2.0 + KeePass key management",
	Long: `Manage TPM 2.0 sealed keys for hardware-backed encryption.

TPM (Trusted Platform Module) provides hardware-based key protection:
  - Keys are sealed to the TPM and bound to this machine
  - No password required - authentication happens automatically
  - Keys cannot be extracted or copied to other machines

Commands:
  init              Initialize a new TPM-sealed master key
  status            Show current TPM configuration status
  reset             Remove the TPM-sealed key
  migrate           Migrate existing database to TPM protection
  migrate-profiles  Migrate profiles from keyring to KeePass

Note: TPM 2.0 is currently only supported on Linux.`,
}

func init() {
	rootCmd.AddCommand(tpmCmd)
}
