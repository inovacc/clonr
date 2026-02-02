package cmd

import (
	"github.com/spf13/cobra"
)

var standaloneCmd = &cobra.Command{
	Use:   "standalone",
	Short: "Instance synchronization commands",
	Long: `Manage standalone mode for synchronizing data between clonr instances.

Standalone mode allows you to securely share profiles, workspaces, and
repository metadata between multiple machines running clonr.

Source Instance (Server):
  clonr standalone init     - Initialize standalone mode and generate sync key
  clonr standalone status   - Show current standalone status
  clonr standalone rotate   - Generate new sync key (invalidates old connections)
  clonr standalone clients  - List connected clients
  clonr standalone revoke   - Revoke a client's access
  clonr standalone disable  - Disable standalone mode

Destination Instance (Client):
  clonr standalone connect     - Connect to a standalone instance
  clonr standalone list        - List all connections
  clonr standalone sync        - Sync data from a connection
  clonr standalone disconnect  - Remove a connection`,
}

func init() {
	rootCmd.AddCommand(standaloneCmd)
}
