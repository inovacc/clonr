package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var slackTestChannel string

var slackTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test notification",
	Long: `Send a test notification to verify Slack configuration.

This sends a simple test message to confirm that your Slack integration
is working correctly.

Examples:
  clonr slack test
  clonr slack test --channel "#testing"`,
	RunE: runSlackTest,
}

func init() {
	slackCmd.AddCommand(slackTestCmd)
	slackTestCmd.Flags().StringVar(&slackTestChannel, "channel", "", "Override default channel for test")
}

func runSlackTest(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Sending test notification...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.Test(ctx, slackTestChannel); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Test notification sent successfully!")
	if slackTestChannel != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Check channel: %s\n", slackTestChannel)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Check your configured default channel.")
	}

	return nil
}
