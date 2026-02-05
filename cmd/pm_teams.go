package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/microsoft"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	pmCmd.AddCommand(pmTeamsCmd)
	pmTeamsCmd.AddCommand(pmTeamsListCmd)
	pmTeamsCmd.AddCommand(pmTeamsChannelsCmd)
	pmTeamsCmd.AddCommand(pmTeamsMessagesCmd)
	pmTeamsCmd.AddCommand(pmTeamsChatsCmd)

	// Flags
	pmTeamsListCmd.Flags().Bool("json", false, "Output as JSON")
	pmTeamsChannelsCmd.Flags().Bool("json", false, "Output as JSON")
	pmTeamsMessagesCmd.Flags().IntP("limit", "n", 10, "Maximum number of messages")
	pmTeamsMessagesCmd.Flags().Bool("json", false, "Output as JSON")
	pmTeamsChatsCmd.Flags().IntP("limit", "n", 10, "Maximum number of chats")
	pmTeamsChatsCmd.Flags().Bool("json", false, "Output as JSON")
}

var pmTeamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "Microsoft Teams operations",
	Long: `Microsoft Teams operations for the active profile.

Available Commands:
  list       List your teams
  channels   List channels in a team
  messages   List messages in a channel
  chats      List your chats

Examples:
  clonr pm teams list
  clonr pm teams channels <team-id>
  clonr pm teams messages <team-id> <channel-id>
  clonr pm teams chats`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var pmTeamsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your teams",
	RunE:  runPmTeamsList,
}

var pmTeamsChannelsCmd = &cobra.Command{
	Use:   "channels <team-id>",
	Short: "List channels in a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runPmTeamsChannels,
}

var pmTeamsMessagesCmd = &cobra.Command{
	Use:   "messages <team-id> <channel-id>",
	Short: "List messages in a channel",
	Args:  cobra.ExactArgs(2),
	RunE:  runPmTeamsMessages,
}

var pmTeamsChatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "List your chats",
	RunE:  runPmTeamsChats,
}

func getTeamsClient() (*microsoft.TeamsClient, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelTeams)
	if err != nil {
		return nil, fmt.Errorf("failed to get Teams config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Teams integration configured; add with: clonr profile teams add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Teams config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Teams config")
	}

	return microsoft.NewTeamsClient(accessToken, microsoft.TeamsClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
		TenantID:     config["tenant_id"],
	}), nil
}

func runPmTeamsList(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getTeamsClient()
	if err != nil {
		return err
	}

	teams, err := client.GetMyTeams(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	if len(teams) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No teams found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(teams)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Your Teams (%d):\n", len(teams))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, team := range teams {
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", team.DisplayName)
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("ID: "+team.ID))

		if team.Description != "" {
			desc := team.Description
			if len(desc) > 60 {
				desc = desc[:60] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(desc))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmTeamsChannels(cmd *cobra.Command, args []string) error {
	teamID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getTeamsClient()
	if err != nil {
		return err
	}

	channels, err := client.GetTeamChannels(context.Background(), teamID)
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	if len(channels) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No channels found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(channels)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Channels (%d):\n", len(channels))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, ch := range channels {
		_, _ = fmt.Fprintf(os.Stdout, "  #%s\n", ch.DisplayName)
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("ID: "+ch.ID))

		if ch.Description != "" {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(ch.Description))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmTeamsMessages(cmd *cobra.Command, args []string) error {
	teamID := args[0]
	channelID := args[1]
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getTeamsClient()
	if err != nil {
		return err
	}

	messages, err := client.GetChannelMessages(context.Background(), teamID, channelID, limit)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(messages)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Messages (%d):\n", len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, msg := range messages {
		from := "Unknown"
		if msg.From != nil && msg.From.User != nil {
			from = msg.From.User.DisplayName
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "  From: %s\n", from)
		_, _ = fmt.Fprintf(os.Stdout, "  Date: %s\n", msg.CreatedDateTime.Format(time.RFC1123))

		if msg.Body != nil && msg.Body.Content != "" {
			content := msg.Body.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", content)
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmTeamsChats(cmd *cobra.Command, _ []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getTeamsClient()
	if err != nil {
		return err
	}

	chats, err := client.GetMyChats(context.Background(), limit)
	if err != nil {
		return fmt.Errorf("failed to list chats: %w", err)
	}

	if len(chats) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No chats found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(chats)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Your Chats (%d):\n", len(chats))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, chat := range chats {
		topic := chat.Topic
		if topic == "" {
			topic = "(No topic)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", topic)
		_, _ = fmt.Fprintf(os.Stdout, "  %s  Type: %s\n", dimStyle.Render("ID: "+chat.ID), chat.ChatType)
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}
