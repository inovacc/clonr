package cmd

import (
	"github.com/spf13/cobra"
)

// mirrorCmd is a backwards-compatible alias for 'org mirror'
var mirrorCmd = &cobra.Command{
	Use:        "mirror <org_name>",
	Short:      "Mirror all repositories from a GitHub organization (alias for 'org mirror')",
	Long:       `This is an alias for 'clonr org mirror'. Use 'clonr org mirror --help' for full documentation.`,
	Args:       cobra.ExactArgs(1),
	RunE:       runMirror,
	Deprecated: "use 'clonr org mirror' instead",
	Hidden:     false, // Still show in help but mark as deprecated
}

func init() {
	rootCmd.AddCommand(mirrorCmd)
	addMirrorFlags(mirrorCmd)
}
