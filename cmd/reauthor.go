package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var reauthorCmd = &cobra.Command{
	Use:   "reauthor",
	Short: "Rewrite git history to change author/committer identity",
	Long: `Reauthor rewrites git history to change the author and committer email/name.

This is useful for consolidating contributor identities, fixing incorrect emails,
or replacing corporate emails with personal ones.

WARNING: This rewrites git history and requires a force push to update remote.
All collaborators will need to re-clone or rebase their work.

Examples:
  # Replace corporate email with personal email
  clonr reauthor --old-email="john@corp.com" --new-email="john@personal.com"

  # Also change the name
  clonr reauthor --old-email="john@corp.com" --new-email="john@personal.com" --new-name="John Doe"

  # List all authors in the repository
  clonr reauthor --list

  # Run in a specific repository
  clonr reauthor --old-email="old@email.com" --new-email="new@email.com" --repo /path/to/repo`,
	RunE: runReauthor,
}

func init() {
	rootCmd.AddCommand(reauthorCmd)

	reauthorCmd.Flags().String("old-email", "", "Email address to replace (required)")
	reauthorCmd.Flags().String("new-email", "", "New email address (required)")
	reauthorCmd.Flags().String("new-name", "", "New author/committer name (optional)")
	reauthorCmd.Flags().String("repo", "", "Path to repository (default: current directory)")
	reauthorCmd.Flags().Bool("list", false, "List all unique author emails in the repository")
	reauthorCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runReauthor(cmd *cobra.Command, _ []string) error {
	listAuthors, _ := cmd.Flags().GetBool("list")
	repoPath, _ := cmd.Flags().GetString("repo")

	if listAuthors {
		return listAuthorEmails(repoPath)
	}

	oldEmail, _ := cmd.Flags().GetString("old-email")
	newEmail, _ := cmd.Flags().GetString("new-email")
	newName, _ := cmd.Flags().GetString("new-name")
	force, _ := cmd.Flags().GetBool("force")

	if oldEmail == "" || newEmail == "" {
		return fmt.Errorf("both --old-email and --new-email are required")
	}

	// Count commits that will be affected
	count, err := core.CountCommitsByEmail(repoPath, oldEmail)
	if err != nil {
		return fmt.Errorf("failed to count commits: %w", err)
	}

	if count == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No commits found with email: %s\n", oldEmail)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Found %d commit(s) with email: %s\n", count, oldEmail)
	_, _ = fmt.Fprintf(os.Stdout, "\nThis will rewrite history:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Old email: %s\n", oldEmail)
	_, _ = fmt.Fprintf(os.Stdout, "  New email: %s\n", newEmail)

	if newName != "" {
		_, _ = fmt.Fprintf(os.Stdout, "  New name:  %s\n", newName)
	}

	if !force {
		_, _ = fmt.Fprintf(os.Stdout, "\nWARNING: This operation rewrites git history and cannot be undone.\n")
		_, _ = fmt.Fprintf(os.Stdout, "You will need to force push after this operation.\n")
		_, _ = fmt.Fprintf(os.Stdout, "\nProceed? [y/N]: ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("operation cancelled")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			_, _ = fmt.Fprintln(os.Stdout, "Operation cancelled.")
			return nil
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nRewriting history...")

	opts := core.ReauthorOptions{
		OldEmail: oldEmail,
		NewEmail: newEmail,
		NewName:  newName,
		RepoPath: repoPath,
		AllRefs:  true,
	}

	result, err := core.Reauthor(opts)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nHistory rewritten successfully!")
	_, _ = fmt.Fprintf(os.Stdout, "  Commits processed: %d\n", result.CommitsRewritten)

	if len(result.BranchesRewritten) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  Branches rewritten: %s\n", strings.Join(result.BranchesRewritten, ", "))
	}

	if len(result.TagsRewritten) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  Tags rewritten: %s\n", strings.Join(result.TagsRewritten, ", "))
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nNext steps:")
	_, _ = fmt.Fprintln(os.Stdout, "  1. Review the changes: git log --oneline")
	_, _ = fmt.Fprintln(os.Stdout, "  2. Force push to remote: git push --force --all && git push --force --tags")

	return nil
}

func listAuthorEmails(repoPath string) error {
	emails, err := core.ListAuthors(repoPath)
	if err != nil {
		return err
	}

	if len(emails) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No authors found in repository.")
		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, "Authors in repository:")
	_, _ = fmt.Fprintln(os.Stdout)

	for _, email := range emails {
		count, err := core.CountCommitsByEmail(repoPath, email)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "  %s (error counting commits)\n", email)
			continue
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s (%d commits)\n", email, count)
	}

	return nil
}
