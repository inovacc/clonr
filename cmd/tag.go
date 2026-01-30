package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var tagMessage string

var tagCmd = &cobra.Command{
	Use:   "tag <name>",
	Short: "Create a git tag",
	Long: `Create a git tag in the current repository.

Examples:
  clonr tag v1.0.0
  clonr tag v1.0.0 -m "Release version 1.0.0"`,
	Args: cobra.ExactArgs(1),
	RunE: runTag,
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.Flags().StringVarP(&tagMessage, "message", "m", "", "Tag message (creates annotated tag)")
}

func runTag(_ *cobra.Command, args []string) error {
	tagName := args[0]

	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	if err := client.Tag(ctx, tagName, tagMessage); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Tag '%s' created successfully!\n", tagName)
	return nil
}
