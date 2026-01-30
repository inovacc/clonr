package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var (
	stashMessage         string
	stashIncludeUntracked bool
	stashKeepIndex       bool
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Stash changes in working directory",
	Long: `Stash changes in the working directory.

Examples:
  clonr stash                     # Stash changes
  clonr stash -m "WIP feature"    # Stash with message
  clonr stash -u                  # Include untracked files
  clonr stash list                # List stashes
  clonr stash pop                 # Apply and remove latest stash
  clonr stash drop                # Remove latest stash`,
	RunE: runStashPush,
}

var stashListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stashes",
	RunE:  runStashList,
}

var stashPopCmd = &cobra.Command{
	Use:   "pop",
	Short: "Apply and remove latest stash",
	RunE:  runStashPop,
}

var stashDropCmd = &cobra.Command{
	Use:   "drop [stash]",
	Short: "Remove a stash",
	RunE:  runStashDrop,
}

func init() {
	rootCmd.AddCommand(stashCmd)

	stashCmd.Flags().StringVarP(&stashMessage, "message", "m", "", "Stash message")
	stashCmd.Flags().BoolVarP(&stashIncludeUntracked, "include-untracked", "u", false, "Include untracked files")
	stashCmd.Flags().BoolVarP(&stashKeepIndex, "keep-index", "k", false, "Keep staged changes")

	stashCmd.AddCommand(stashListCmd)
	stashCmd.AddCommand(stashPopCmd)
	stashCmd.AddCommand(stashDropCmd)
}

func runStashPush(_ *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	opts := git.StashOptions{
		Message:          stashMessage,
		IncludeUntracked: stashIncludeUntracked,
		KeepIndex:        stashKeepIndex,
	}

	if err := client.Stash(ctx, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Changes stashed successfully!")
	return nil
}

func runStashList(_ *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	list, err := client.StashList(ctx)
	if err != nil {
		return err
	}

	if list == "" {
		_, _ = fmt.Fprintln(os.Stdout, "No stashes found")
		return nil
	}

	_, _ = fmt.Fprint(os.Stdout, list)
	return nil
}

func runStashPop(_ *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	if err := client.StashPop(ctx); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Stash applied and removed!")
	return nil
}

func runStashDrop(_ *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	stash := ""
	if len(args) > 0 {
		stash = args[0]
	}

	if err := client.StashDrop(ctx, stash); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Stash dropped!")
	return nil
}
