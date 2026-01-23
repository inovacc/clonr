package cmd

import (
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map [directory]",
	Short: "Scan directory for existing Git repositories",
	Long: `Recursively scan a directory to find existing Git repositories and register them with Clonr for management.

By default, common directories like node_modules, vendor, and build folders are skipped to improve performance.

Examples:
  clonr map                           # Scan current directory
  clonr map ~/projects                # Scan specific directory
  clonr map --dry-run ~/projects      # Preview without adding
  clonr map --depth 3 ~/projects      # Limit scan depth
  clonr map --json ~/projects         # Output as JSON
  clonr map --no-exclude ~/projects   # Don't skip common directories`,
	RunE: runMap,
}

func init() {
	rootCmd.AddCommand(mapCmd)

	mapCmd.Flags().Bool("dry-run", false, "Show what would be added without actually adding")
	mapCmd.Flags().Int("depth", 0, "Maximum directory depth to scan (0 = unlimited)")
	mapCmd.Flags().Bool("json", false, "Output results as JSON")
	mapCmd.Flags().BoolP("verbose", "v", false, "Show verbose output including skipped directories")
	mapCmd.Flags().Bool("no-exclude", false, "Don't skip common directories (node_modules, vendor, etc.)")
	mapCmd.Flags().StringSlice("exclude", nil, "Additional directories to exclude")
}

func runMap(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	maxDepth, _ := cmd.Flags().GetInt("depth")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	verbose, _ := cmd.Flags().GetBool("verbose")
	noExclude, _ := cmd.Flags().GetBool("no-exclude")
	extraExclude, _ := cmd.Flags().GetStringSlice("exclude")

	// Build exclude list
	var excludeDirs []string

	if !noExclude {
		excludeDirs = append(excludeDirs, core.DefaultExcludeDirs...)
	}

	excludeDirs = append(excludeDirs, extraExclude...)

	opts := core.MapOptions{
		DryRun:   dryRun,
		MaxDepth: maxDepth,
		Exclude:  excludeDirs,
		JSON:     jsonOutput,
		Verbose:  verbose,
	}

	return core.MapReposWithOptions(args, opts)
}
