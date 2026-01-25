package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Export database to JSON snapshot",
	Long: `Create a JSON snapshot of the clonr database.

The snapshot includes:
  - All tracked repositories with metadata
  - Current git branch for each repository
  - Configuration settings

Examples:
  clonr snapshot                     # Output to stdout
  clonr snapshot -o backup.json      # Write to file
  clonr snapshot --no-branch         # Skip branch detection (faster)
  clonr snapshot --no-config         # Exclude configuration`,
	RunE: runSnapshot,
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
	snapshotCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	snapshotCmd.Flags().Bool("no-branch", false, "Skip branch detection")
	snapshotCmd.Flags().Bool("no-config", false, "Exclude configuration")
	snapshotCmd.Flags().Bool("compact", false, "Compact JSON output (no indentation)")
}

func runSnapshot(cmd *cobra.Command, _ []string) error {
	outputPath, _ := cmd.Flags().GetString("output")
	noBranch, _ := cmd.Flags().GetBool("no-branch")
	noConfig, _ := cmd.Flags().GetBool("no-config")
	compact, _ := cmd.Flags().GetBool("compact")

	opts := core.CreateSnapshotOptions{
		IncludeBranches: !noBranch,
		IncludeConfig:   !noConfig,
	}

	// Show progress on stderr if outputting to stdout
	if outputPath == "" && !noBranch {
		_, _ = fmt.Fprintln(os.Stderr, "Creating snapshot (fetching branches)...")
	}

	snapshot, err := core.CreateSnapshot(opts)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	pretty := !compact

	// Write to file or stdout
	if outputPath != "" {
		if err := core.WriteSnapshotToFile(outputPath, snapshot, pretty); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(os.Stderr, "Snapshot written to %s (%d repositories)\n",
			outputPath, len(snapshot.Repositories))

		return nil
	}

	return core.WriteSnapshot(os.Stdout, snapshot, pretty)
}
