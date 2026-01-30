package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/inovacc/clonr/internal/security"
	"github.com/spf13/cobra"
)

var (
	pushTags       bool
	pushSetUp      bool
	pushForce      bool
	pushCheckLeaks bool
	pushSkipLeaks  bool
)

var pushCmd = &cobra.Command{
	Use:   "push [remote] [branch]",
	Short: "Push changes to remote repository",
	Long: `Push changes to remote repository using profile authentication.

By default, scans for secrets before pushing. Use --skip-leaks to bypass.

Examples:
  clonr push
  clonr push --tags
  clonr push -u origin main
  clonr push --skip-leaks         # Skip secret scanning`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushTags, "tags", false, "Push all tags")
	pushCmd.Flags().BoolVarP(&pushSetUp, "set-upstream", "u", false, "Set upstream for the current branch")
	pushCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Force push")
	pushCmd.Flags().BoolVar(&pushCheckLeaks, "check-leaks", true, "Check for secrets before pushing (default: true)")
	pushCmd.Flags().BoolVar(&pushSkipLeaks, "skip-leaks", false, "Skip secret scanning")
}

func runPush(_ *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	// Check for leaks before pushing (unless skipped)
	if pushCheckLeaks && !pushSkipLeaks {
		_, _ = fmt.Fprintln(os.Stdout, "🔍 Scanning for secrets...")

		scanner, err := security.NewLeakScanner()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not initialize leak scanner: %v\n", err)
		} else {
			// Get current directory as repo path
			repoPath, _ := os.Getwd()

			// Load .gitleaksignore if exists
			_ = scanner.LoadGitleaksIgnore(repoPath)

			// Scan unpushed commits or staged changes
			result, err := scanner.ScanUnpushedCommits(ctx, repoPath)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "⚠️  Warning: Leak scan failed: %v\n", err)
			} else if result.HasLeaks {
				_, _ = fmt.Fprint(os.Stderr, security.FormatFindings(result.Findings))
				_, _ = fmt.Fprintln(os.Stderr, "❌ Push aborted: secrets detected!")
				_, _ = fmt.Fprintln(os.Stderr, "   Use --skip-leaks to push anyway (not recommended)")
				return fmt.Errorf("secrets detected in commits")
			} else {
				_, _ = fmt.Fprintln(os.Stdout, "✅ No secrets detected")
			}
		}
	}

	var remote, branch string
	if len(args) >= 1 {
		remote = args[0]
	}
	if len(args) >= 2 {
		branch = args[1]
	}

	opts := git.PushOptions{
		SetUpstream: pushSetUp,
		Force:       pushForce,
		Tags:        pushTags,
	}

	// Use authenticated command via credential helper
	if err := client.Push(ctx, remote, branch, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Push completed successfully!")
	return nil
}
