package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var slackListJSON bool

var slackListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "status"},
	Short:   "Show current Slack configuration",
	Long: `Display the current Slack integration configuration.

Shows:
  - Integration status (enabled/disabled)
  - Integration type (webhook or bot)
  - Default channel
  - Event configuration

Examples:
  clonr slack list
  clonr slack list --json`,
	RunE: runSlackList,
}

func init() {
	slackCmd.AddCommand(slackListCmd)
	slackListCmd.Flags().BoolVar(&slackListJSON, "json", false, "Output as JSON")
}

// SlackListOutput represents the JSON output for slack list.
type SlackListOutput struct {
	Enabled        bool               `json:"enabled"`
	Type           string             `json:"type"`
	DefaultChannel string             `json:"default_channel,omitempty"`
	WorkspaceID    string             `json:"workspace_id,omitempty"`
	WorkspaceName  string             `json:"workspace_name,omitempty"`
	Events         []SlackEventOutput `json:"events,omitempty"`
}

// SlackEventOutput represents event configuration in JSON.
type SlackEventOutput struct {
	Event    string   `json:"event"`
	Enabled  bool     `json:"enabled"`
	Channel  string   `json:"channel,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Filters  []string `json:"filters,omitempty"`
}

func runSlackList(_ *cobra.Command, _ []string) error {
	manager, err := core.NewSlackManager()
	if err != nil {
		return fmt.Errorf("failed to initialize Slack manager: %w", err)
	}

	config, err := manager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		if slackListJSON {
			_, _ = fmt.Fprintln(os.Stdout, "null")
			return nil
		}

		printEmptyResult("Slack integration", "clonr slack add --webhook <url>")

		return nil
	}

	// Determine integration type
	integrationType := "none"
	if len(config.EncryptedWebhookURL) > 0 {
		integrationType = "webhook"
	}

	if len(config.EncryptedBotToken) > 0 && config.BotEnabled {
		integrationType = "bot"
	}

	// JSON output
	if slackListJSON {
		output := SlackListOutput{
			Enabled:        config.Enabled,
			Type:           integrationType,
			DefaultChannel: config.DefaultChannel,
			WorkspaceID:    config.WorkspaceID,
			WorkspaceName:  config.WorkspaceName,
		}

		for _, e := range config.Events {
			output.Events = append(output.Events, SlackEventOutput{
				Event:    e.Event,
				Enabled:  e.Enabled,
				Channel:  e.Channel,
				Priority: e.Priority,
				Filters:  e.Filters,
			})
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(output)
	}

	// Text output
	printBoxHeader("SLACK INTEGRATION")
	printBoxLine("Status", formatEnabled(config.Enabled))
	printBoxLine("Type", integrationType)

	if config.DefaultChannel != "" {
		printBoxLine("Channel", config.DefaultChannel)
	}

	if config.WorkspaceName != "" {
		printBoxLine("Workspace", config.WorkspaceName)
	}

	printBoxFooter()

	// Show events
	if len(config.Events) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Event Configuration:")

		enabledEvents := make([]string, 0)
		disabledEvents := make([]string, 0)

		for _, e := range config.Events {
			if e.Enabled {
				enabledEvents = append(enabledEvents, e.Event)
			} else {
				disabledEvents = append(disabledEvents, e.Event)
			}
		}

		if len(enabledEvents) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Enabled:  %s\n", strings.Join(enabledEvents, ", "))
		}

		if len(disabledEvents) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "  Disabled: %s\n", strings.Join(disabledEvents, ", "))
		}
	}

	return nil
}

func formatEnabled(enabled bool) string {
	if enabled {
		return "enabled"
	}

	return "disabled"
}
