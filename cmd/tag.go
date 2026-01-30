package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	tagMessage   string
	tagAnnotated bool
)

var tagCmd = &cobra.Command{
	Use:   "tag <name>",
	Short: "Create a git tag",
	Long: `Create a git tag in the current repository.

Examples:
  clonr tag v1.0.0
  clonr tag v1.0.0 -m "Release version 1.0.0"
  clonr tag v1.0.0 -a -m "Annotated tag"`,
	Args: cobra.ExactArgs(1),
	RunE: runTag,
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.Flags().StringVarP(&tagMessage, "message", "m", "", "Tag message (creates annotated tag)")
	tagCmd.Flags().BoolVarP(&tagAnnotated, "annotate", "a", false, "Create an annotated tag")
}

func runTag(_ *cobra.Command, args []string) error {
	tagName := args[0]

	// Check if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		return fmt.Errorf("not a git repository")
	}

	var cmd *exec.Cmd

	if tagMessage != "" {
		cmd = exec.Command("git", "tag", "-a", tagName, "-m", tagMessage)
	} else if tagAnnotated {
		cmd = exec.Command("git", "tag", "-a", tagName)
	} else {
		cmd = exec.Command("git", "tag", tagName)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Tag '%s' created successfully!\n", tagName)
	return nil
}
