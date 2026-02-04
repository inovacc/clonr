package cmd

import "github.com/spf13/cobra"

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git operations with profile authentication",
	Long: `Git operations that use clonr profile authentication.

These commands mirror standard git commands but automatically use
the active clonr profile for authentication with GitHub.

Available Commands:
  clone     Clone a repository
  status    Show working tree status
  commit    Create a commit
  push      Push changes (with pre-push security scan)
  pull      Pull changes from remote
  log       Show commit log
  diff      Show changes
  branch    Manage branches

Examples:
  clonr git status
  clonr git log --limit 5
  clonr git push -u origin main`,
}

func init() {
	rootCmd.AddCommand(gitCmd)
}
