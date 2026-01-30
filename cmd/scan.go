package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/security"
	"github.com/spf13/cobra"
)

var scanGitHistory bool

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan for secrets in repository",
	Long: `Scan for hardcoded secrets and credentials.

Uses gitleaks rules to detect:
- API keys, tokens, passwords
- Private keys, certificates
- Cloud provider credentials
- Database connection strings

Examples:
  clonr scan                      # Scan current directory
  clonr scan /path/to/repo        # Scan specific path
  clonr scan --git                # Scan git history`,
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().BoolVar(&scanGitHistory, "git", false, "Scan git history instead of files")
}

func runScan(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine path to scan
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	_, _ = fmt.Fprintf(os.Stdout, "🔍 Scanning %s for secrets...\n\n", path)

	scanner, err := security.NewLeakScanner()
	if err != nil {
		return fmt.Errorf("failed to initialize scanner: %w", err)
	}

	// Load .gitleaksignore if exists
	_ = scanner.LoadGitleaksIgnore(path)

	var result *security.ScanResult

	if scanGitHistory {
		result, err = scanner.ScanGitRepo(ctx, path)
	} else {
		result, err = scanner.ScanDirectory(ctx, path)
	}

	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if result.HasLeaks {
		_, _ = fmt.Fprint(os.Stdout, security.FormatFindings(result.Findings))
		_, _ = fmt.Fprintf(os.Stdout, "❌ Found %d secret(s)\n", len(result.Findings))

		return fmt.Errorf("secrets detected")
	}

	_, _ = fmt.Fprintln(os.Stdout, "✅ No secrets detected!")

	return nil
}
