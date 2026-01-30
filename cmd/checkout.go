package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var (
	checkoutCreate bool
	checkoutForce  bool
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch|commit>",
	Short: "Switch branches or restore files",
	Long: `Switch branches or restore working tree files.

Examples:
  clonr checkout main              # Switch to main branch
  clonr checkout -b feature        # Create and switch to new branch
  clonr checkout -f main           # Force checkout (discard changes)`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckout,
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
	checkoutCmd.Flags().BoolVarP(&checkoutCreate, "branch", "b", false, "Create and switch to new branch")
	checkoutCmd.Flags().BoolVarP(&checkoutForce, "force", "f", false, "Force checkout (discard local changes)")
}

func runCheckout(_ *cobra.Command, args []string) error {
	target := args[0]

	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	opts := git.CheckoutOptions{
		Create: checkoutCreate,
		Force:  checkoutForce,
	}

	if err := client.Checkout(ctx, target, opts); err != nil {
		return err
	}

	if checkoutCreate {
		_, _ = fmt.Fprintf(os.Stdout, "Switched to new branch '%s'\n", target)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Switched to '%s'\n", target)
	}

	return nil
}
