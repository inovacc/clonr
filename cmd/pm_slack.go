package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
	"github.com/spf13/cobra"
)

var pmSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slack operations for consuming messages",
	Long: `Read and search Slack messages and channels.

Available Commands:
  channels      List Slack channels
  messages      Read messages from a channel
  search        Search for messages
  thread        View thread replies
  users         List workspace users
  auth          Open Slack token page in browser

Authentication:
  Uses Slack Bot Token from (in priority order):
  1. --token flag
  2. SLACK_TOKEN environment variable
  3. SLACK_BOT_TOKEN environment variable
  4. Stored clonr Slack configuration
  5. ~/.config/clonr/slack.json config file

Required Bot Token Scopes:
  - channels:read       (list public channels)
  - channels:history    (read public channel messages)
  - groups:read         (list private channels)
  - groups:history      (read private channel messages)
  - im:history          (read direct messages)
  - mpim:history        (read group DMs)
  - search:read         (search messages)
  - users:read          (list users)

Examples:
  clonr pm slack channels
  clonr pm slack messages --channel general --limit 20
  clonr pm slack search "deployment error"
  clonr pm slack thread --channel general --ts 1234567890.123456
  clonr pm slack users`,
}

var pmSlackChannelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "List Slack channels",
	Long: `List channels in the Slack workspace.

Shows:
  - Channel name and ID
  - Member count
  - Topic/purpose
  - Public/private status

Examples:
  clonr pm slack channels
  clonr pm slack channels --private
  clonr pm slack channels --json`,
	RunE: runPMSlackChannels,
}

var pmSlackMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Read messages from a channel",
	Long: `Read recent messages from a Slack channel.

Shows:
  - Message author and timestamp
  - Message text
  - Reactions and thread reply counts
  - Attachments and files

Examples:
  clonr pm slack messages --channel general
  clonr pm slack messages --channel C01234567 --limit 50
  clonr pm slack messages --channel dev --since 24h
  clonr pm slack messages --channel general --json`,
	RunE: runPMSlackMessages,
}

var pmSlackSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for messages",
	Long: `Search for messages in Slack.

Supports Slack search modifiers:
  in:#channel          Search in specific channel
  from:@user           Search from specific user
  has:link             Messages with links
  has:reaction         Messages with reactions
  before:YYYY-MM-DD    Before date
  after:YYYY-MM-DD     After date

Examples:
  clonr pm slack search "deployment error"
  clonr pm slack search "in:#dev bug"
  clonr pm slack search "from:@john error" --limit 50
  clonr pm slack search "has:reaction" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: runPMSlackSearch,
}

var pmSlackThreadCmd = &cobra.Command{
	Use:   "thread",
	Short: "View thread replies",
	Long: `View replies to a message thread.

Requires the channel and the parent message timestamp (ts).
You can find the timestamp from the message list or search results.

Examples:
  clonr pm slack thread --channel general --ts 1234567890.123456
  clonr pm slack thread --channel C01234567 --ts 1234567890.123456 --json`,
	RunE: runPMSlackThread,
}

var pmSlackUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List workspace users",
	Long: `List users in the Slack workspace.

Shows:
  - User name and display name
  - Real name and email
  - Status and timezone
  - Admin/bot status

Examples:
  clonr pm slack users
  clonr pm slack users --limit 50
  clonr pm slack users --json`,
	RunE: runPMSlackUsers,
}

var pmSlackAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Open Slack app settings page in browser",
	Long: `Open the Slack API app settings page in your default browser.

This command helps you access the page to create or manage bot tokens:
  - Slack Apps: https://api.slack.com/apps

After creating a bot token, configure it:
  export SLACK_TOKEN=xoxb-...

Or add it to clonr's Slack integration:
  clonr slack add --bot-token xoxb-... --channel #general

Examples:
  clonr pm slack auth`,
	RunE: runPMSlackAuth,
}

var pmSlackConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to Slack via OAuth",
	Long: `Authenticate with Slack using OAuth flow.

This command opens your browser to authorize clonr with your Slack workspace.
After authorization, the bot token is automatically saved.

Prerequisites:
  1. Create a Slack App at https://api.slack.com/apps
  2. Add OAuth redirect URL: http://localhost:8338/slack/callback
  3. Add required Bot Token Scopes:
     - channels:read, channels:history
     - groups:read, groups:history
     - search:read, users:read
  4. Install the app to your workspace

The Client ID and Client Secret can be provided via:
  - --client-id and --client-secret flags
  - SLACK_CLIENT_ID and SLACK_CLIENT_SECRET environment variables

Examples:
  clonr pm slack connect --client-id <id> --client-secret <secret>
  SLACK_CLIENT_ID=xxx SLACK_CLIENT_SECRET=yyy clonr pm slack connect`,
	RunE: runPMSlackConnect,
}

func init() {
	pmCmd.AddCommand(pmSlackCmd)
	pmSlackCmd.AddCommand(pmSlackChannelsCmd)
	pmSlackCmd.AddCommand(pmSlackMessagesCmd)
	pmSlackCmd.AddCommand(pmSlackSearchCmd)
	pmSlackCmd.AddCommand(pmSlackThreadCmd)
	pmSlackCmd.AddCommand(pmSlackUsersCmd)
	pmSlackCmd.AddCommand(pmSlackAuthCmd)
	pmSlackCmd.AddCommand(pmSlackConnectCmd)

	// Channels flags
	addPMCommonFlags(pmSlackChannelsCmd)
	pmSlackChannelsCmd.Flags().Bool("private", false, "Include private channels")
	pmSlackChannelsCmd.Flags().Bool("archived", false, "Include archived channels")
	pmSlackChannelsCmd.Flags().Int("limit", 100, "Maximum number of channels to return")

	// Messages flags
	addPMCommonFlags(pmSlackMessagesCmd)
	pmSlackMessagesCmd.Flags().StringP("channel", "c", "", "Channel name or ID (required)")
	pmSlackMessagesCmd.Flags().Int("limit", 20, "Maximum number of messages to return")
	pmSlackMessagesCmd.Flags().String("since", "", "Show messages since duration (e.g., 24h, 7d)")
	pmSlackMessagesCmd.Flags().String("before", "", "Show messages before timestamp")
	_ = pmSlackMessagesCmd.MarkFlagRequired("channel")

	// Search flags
	addPMCommonFlags(pmSlackSearchCmd)
	pmSlackSearchCmd.Flags().Int("limit", 20, "Maximum number of results")
	pmSlackSearchCmd.Flags().Int("page", 1, "Page number")
	pmSlackSearchCmd.Flags().String("sort", "timestamp", "Sort by: score or timestamp")
	pmSlackSearchCmd.Flags().String("dir", "desc", "Sort direction: asc or desc")

	// Thread flags
	addPMCommonFlags(pmSlackThreadCmd)
	pmSlackThreadCmd.Flags().StringP("channel", "c", "", "Channel name or ID (required)")
	pmSlackThreadCmd.Flags().String("ts", "", "Parent message timestamp (required)")
	pmSlackThreadCmd.Flags().Int("limit", 100, "Maximum number of replies")
	_ = pmSlackThreadCmd.MarkFlagRequired("channel")
	_ = pmSlackThreadCmd.MarkFlagRequired("ts")

	// Users flags
	addPMCommonFlags(pmSlackUsersCmd)
	pmSlackUsersCmd.Flags().Int("limit", 100, "Maximum number of users to return")

	// Connect flags
	pmSlackConnectCmd.Flags().String("client-id", "", "Slack App Client ID")
	pmSlackConnectCmd.Flags().String("client-secret", "", "Slack App Client Secret")
	pmSlackConnectCmd.Flags().Int("port", 8338, "Local callback server port")
	pmSlackConnectCmd.Flags().String("scopes", "", "OAuth scopes (comma-separated)")
	pmSlackConnectCmd.Flags().Bool("save", true, "Save token to active profile")
	pmSlackConnectCmd.Flags().StringP("channel", "c", "", "Default channel for notifications")
	pmSlackConnectCmd.Flags().StringP("profile", "p", "", "Profile to save token to (default: active profile)")
	pmSlackConnectCmd.Flags().String("name", "slack", "Name for the Slack channel configuration")
}

func runPMSlackChannels(cmd *cobra.Command, _ []string) error {
	tokenFlag, _ := cmd.Flags().GetString("token")
	outputJSON, _ := cmd.Flags().GetBool("json")
	includePrivate, _ := cmd.Flags().GetBool("private")
	includeArchived, _ := cmd.Flags().GetBool("archived")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := slack.ResolveSlackToken(tokenFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client := slack.NewClient(token, slack.ClientOptions{Logger: logger})

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching channels..."))
	}

	// Build channel types
	types := "public_channel"
	if includePrivate {
		types = "public_channel,private_channel"
	}

	// List channels
	result, err := client.ListChannels(cmd.Context(), slack.ListChannelsOptions{
		Types:           types,
		ExcludeArchived: !includeArchived,
		Limit:           limit,
	})
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Channels)
	}

	if len(result.Channels) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No channels found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSlack Channels (%d)\n\n", len(result.Channels))
	_, _ = fmt.Fprintf(os.Stdout, "  %-12s │ %-25s │ %-8s │ %s\n", "ID", "Name", "Members", "Topic")
	_, _ = fmt.Fprintln(os.Stdout, "  ─────────────┼───────────────────────────┼──────────┼─────────────────────")

	for _, ch := range result.Channels {
		name := ch.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		topic := ch.Topic.Value
		if len(topic) > 40 {
			topic = topic[:37] + "..."
		}

		prefix := "#"
		if ch.IsPrivate {
			prefix = "🔒"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %-12s │ %s%-24s │ %-8d │ %s\n",
			ch.ID, prefix, name, ch.NumMembers, topic)
	}

	return nil
}

func runPMSlackMessages(cmd *cobra.Command, _ []string) error {
	tokenFlag, _ := cmd.Flags().GetString("token")
	outputJSON, _ := cmd.Flags().GetBool("json")
	channel, _ := cmd.Flags().GetString("channel")
	limit, _ := cmd.Flags().GetInt("limit")
	since, _ := cmd.Flags().GetString("since")
	before, _ := cmd.Flags().GetString("before")

	// Resolve token
	token, _, err := slack.ResolveSlackToken(tokenFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client := slack.NewClient(token, slack.ClientOptions{Logger: logger})

	// Resolve channel ID if name given
	channelID, err := resolveSlackChannelID(cmd.Context(), client, channel, outputJSON)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Fetching messages from %s...\n"), channel)
	}

	// Build options
	opts := slack.GetChannelHistoryOptions{
		Channel: channelID,
		Limit:   limit,
	}

	if before != "" {
		opts.Latest = before
	}

	if since != "" {
		duration, parseErr := parseDuration(since)
		if parseErr != nil {
			return fmt.Errorf("invalid duration: %w", parseErr)
		}

		oldest := time.Now().Add(-duration)
		opts.Oldest = slack.FormatTimestamp(oldest)
	}

	// Get messages
	result, err := client.GetChannelHistory(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Fetch user info for display names
	userCache := make(map[string]string)

	if !outputJSON {
		for _, msg := range result.Messages {
			if msg.User != "" {
				if _, ok := userCache[msg.User]; !ok {
					if user, userErr := client.GetUser(cmd.Context(), msg.User); userErr == nil {
						userCache[msg.User] = user.Profile.DisplayName
						if userCache[msg.User] == "" {
							userCache[msg.User] = user.Name
						}
					} else {
						userCache[msg.User] = msg.User
					}
				}
			}
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Messages)
	}

	if len(result.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nMessages from #%s (%d)\n\n", channel, len(result.Messages))

	// Messages are returned newest first, reverse for chronological display
	for i := len(result.Messages) - 1; i >= 0; i-- {
		msg := result.Messages[i]

		ts, _ := slack.ParseTimestamp(msg.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		userName := userCache[msg.User]
		if userName == "" {
			userName = msg.User
			if msg.BotProfile != nil {
				userName = msg.BotProfile.Name + " (bot)"
			}
		}

		// Print message header
		_, _ = fmt.Fprintf(os.Stdout, "┌─ %s @ %s\n", okStyle.Render(userName), dimStyle.Render(timeStr))

		// Print message text (with indentation)
		text := msg.Text
		if len(text) > 500 {
			text = text[:497] + "..."
		}

		for line := range strings.SplitSeq(text, "\n") {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
		}

		// Print reactions
		if len(msg.Reactions) > 0 {
			var reactions []string
			for _, r := range msg.Reactions {
				reactions = append(reactions, fmt.Sprintf(":%s: %d", r.Name, r.Count))
			}

			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", dimStyle.Render(strings.Join(reactions, "  ")))
		}

		// Print thread info
		if msg.ReplyCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", dimStyle.Render(fmt.Sprintf("💬 %d replies", msg.ReplyCount)))
		}

		// Print files
		if len(msg.Files) > 0 {
			for _, f := range msg.Files {
				_, _ = fmt.Fprintf(os.Stdout, "│  📎 %s (%s)\n", f.Name, f.PrettyType)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "└─ ts: %s\n\n", dimStyle.Render(msg.Timestamp))
	}

	if result.HasMore {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("More messages available. Use --before flag with the oldest timestamp to paginate."))
	}

	return nil
}

func runPMSlackSearch(cmd *cobra.Command, args []string) error {
	tokenFlag, _ := cmd.Flags().GetString("token")
	outputJSON, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	sort, _ := cmd.Flags().GetString("sort")
	dir, _ := cmd.Flags().GetString("dir")

	query := strings.Join(args, " ")

	// Resolve token
	token, _, err := slack.ResolveSlackToken(tokenFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client := slack.NewClient(token, slack.ClientOptions{Logger: logger})

	if !outputJSON {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Searching for: %s\n"), query)
	}

	// Search
	result, err := client.SearchMessages(cmd.Context(), slack.SearchMessagesOptions{
		Query: query,
		Sort:  sort,
		Dir:   dir,
		Count: limit,
		Page:  page,
	})
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	if result.Total == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No results found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSearch Results for \"%s\" (%d of %d)\n\n", query, len(result.Matches), result.Total)

	for _, match := range result.Matches {
		ts, _ := slack.ParseTimestamp(match.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		channelName := match.Channel.Name
		if channelName == "" {
			channelName = match.Channel.ID
		}

		text := match.Text
		if len(text) > 200 {
			text = text[:197] + "..."
		}

		_, _ = fmt.Fprintf(os.Stdout, "┌─ #%s @ %s\n", okStyle.Render(channelName), dimStyle.Render(timeStr))
		_, _ = fmt.Fprintf(os.Stdout, "│  by: %s\n", match.Username)

		for line := range strings.SplitSeq(text, "\n") {
			if line != "" {
				_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "└─ %s\n\n", dimStyle.Render(match.Permalink))
	}

	if result.Paging.Pages > page {
		_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Page %d of %d. Use --page to see more.\n"), page, result.Paging.Pages)
	}

	return nil
}

func runPMSlackThread(cmd *cobra.Command, _ []string) error {
	tokenFlag, _ := cmd.Flags().GetString("token")
	outputJSON, _ := cmd.Flags().GetBool("json")
	channel, _ := cmd.Flags().GetString("channel")
	threadTS, _ := cmd.Flags().GetString("ts")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := slack.ResolveSlackToken(tokenFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client := slack.NewClient(token, slack.ClientOptions{Logger: logger})

	// Resolve channel ID if name given
	channelID, err := resolveSlackChannelID(cmd.Context(), client, channel, outputJSON)
	if err != nil {
		return err
	}

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching thread replies..."))
	}

	// Get thread
	result, err := client.GetThreadReplies(cmd.Context(), slack.GetThreadRepliesOptions{
		Channel:  channelID,
		ThreadTS: threadTS,
		Limit:    limit,
	})
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	// Fetch user info for display names
	userCache := make(map[string]string)

	if !outputJSON {
		for _, msg := range result.Messages {
			if msg.User != "" {
				if _, ok := userCache[msg.User]; !ok {
					if user, userErr := client.GetUser(cmd.Context(), msg.User); userErr == nil {
						userCache[msg.User] = user.Profile.DisplayName
						if userCache[msg.User] == "" {
							userCache[msg.User] = user.Name
						}
					} else {
						userCache[msg.User] = msg.User
					}
				}
			}
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Messages)
	}

	if len(result.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No replies found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nThread Replies (%d messages)\n\n", len(result.Messages))

	for i, msg := range result.Messages {
		ts, _ := slack.ParseTimestamp(msg.Timestamp)
		timeStr := ts.Format("Jan 02 15:04")

		userName := userCache[msg.User]
		if userName == "" {
			userName = msg.User
		}

		prefix := "├─"
		if i == 0 {
			prefix = "┌─ (original)"
		} else if i == len(result.Messages)-1 {
			prefix = "└─"
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s %s @ %s\n", prefix, okStyle.Render(userName), dimStyle.Render(timeStr))

		text := msg.Text
		if len(text) > 500 {
			text = text[:497] + "..."
		}

		for line := range strings.SplitSeq(text, "\n") {
			_, _ = fmt.Fprintf(os.Stdout, "│  %s\n", line)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runPMSlackUsers(cmd *cobra.Command, _ []string) error {
	tokenFlag, _ := cmd.Flags().GetString("token")
	outputJSON, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := slack.ResolveSlackToken(tokenFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJSON {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client := slack.NewClient(token, slack.ClientOptions{Logger: logger})

	if !outputJSON {
		_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Fetching users..."))
	}

	// List users
	result, err := client.ListUsers(cmd.Context(), slack.ListUsersOptions{
		Limit: limit,
	})
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Filter out bots and deleted users for display
	var activeUsers []slack.User

	for _, u := range result.Users {
		if !u.Deleted && !u.IsBot {
			activeUsers = append(activeUsers, u)
		}
	}

	// Output
	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result.Users)
	}

	if len(activeUsers) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No users found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSlack Users (%d active)\n\n", len(activeUsers))
	_, _ = fmt.Fprintf(os.Stdout, "  %-15s │ %-20s │ %-25s │ %s\n", "Username", "Display Name", "Real Name", "Status")
	_, _ = fmt.Fprintln(os.Stdout, "  ────────────────┼──────────────────────┼───────────────────────────┼─────────────")

	for _, u := range activeUsers {
		displayName := u.Profile.DisplayName
		if len(displayName) > 20 {
			displayName = displayName[:17] + "..."
		}

		realName := u.Profile.RealName
		if len(realName) > 25 {
			realName = realName[:22] + "..."
		}

		status := u.Profile.StatusEmoji + " " + u.Profile.StatusText
		if len(status) > 30 {
			status = status[:27] + "..."
		}

		adminBadge := ""
		if u.IsAdmin {
			adminBadge = " (admin)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  @%-14s │ %-20s │ %-25s │ %s%s\n",
			u.Name, displayName, realName, status, adminBadge)
	}

	return nil
}

func runPMSlackAuth(_ *cobra.Command, _ []string) error {
	_, _ = fmt.Fprintf(os.Stdout, "Opening Slack API apps page: %s\n", slack.SlackTokenURL)

	if err := core.OpenBrowser(slack.SlackTokenURL); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open browser: %v\n", err)
		_, _ = fmt.Fprintf(os.Stdout, "Please visit: %s\n", slack.SlackTokenURL)
	}

	_, _ = fmt.Fprintln(os.Stdout, "\nTo create a bot token for reading messages:")
	_, _ = fmt.Fprintln(os.Stdout, "  1. Create a new app or select an existing one")
	_, _ = fmt.Fprintln(os.Stdout, "  2. Go to 'OAuth & Permissions'")
	_, _ = fmt.Fprintln(os.Stdout, "  3. Add Bot Token Scopes:")
	_, _ = fmt.Fprintln(os.Stdout, "     - channels:read")
	_, _ = fmt.Fprintln(os.Stdout, "     - channels:history")
	_, _ = fmt.Fprintln(os.Stdout, "     - groups:read")
	_, _ = fmt.Fprintln(os.Stdout, "     - groups:history")
	_, _ = fmt.Fprintln(os.Stdout, "     - search:read")
	_, _ = fmt.Fprintln(os.Stdout, "     - users:read")
	_, _ = fmt.Fprintln(os.Stdout, "  4. Install the app to your workspace")
	_, _ = fmt.Fprintln(os.Stdout, "  5. Copy the Bot User OAuth Token (xoxb-...)")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Configure the token:")
	_, _ = fmt.Fprintln(os.Stdout, "  export SLACK_TOKEN=xoxb-...")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Or add to clonr's Slack integration:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr slack add --bot-token xoxb-... --channel #general")
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Or create ~/.config/clonr/slack.json:")
	_, _ = fmt.Fprintln(os.Stdout, `  {"token": "xoxb-..."}`)
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Or use OAuth flow:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack connect --client-id <id> --client-secret <secret>")

	return nil
}

func runPMSlackConnect(cmd *cobra.Command, _ []string) error {
	clientID, _ := cmd.Flags().GetString("client-id")
	clientSecret, _ := cmd.Flags().GetString("client-secret")
	port, _ := cmd.Flags().GetInt("port")
	scopes, _ := cmd.Flags().GetString("scopes")
	saveToken, _ := cmd.Flags().GetBool("save")
	channel, _ := cmd.Flags().GetString("channel")
	profileName, _ := cmd.Flags().GetString("profile")
	channelName, _ := cmd.Flags().GetString("name")

	// Try environment variables if flags not provided
	if clientID == "" {
		clientID = os.Getenv("SLACK_CLIENT_ID")
	}

	if clientSecret == "" {
		clientSecret = os.Getenv("SLACK_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`slack client ID and client secret are required

Provide them via flags:
  clonr pm slack connect --client-id <id> --client-secret <secret>

Or via environment variables:
  export SLACK_CLIENT_ID=<your-client-id>
  export SLACK_CLIENT_SECRET=<your-client-secret>

To get these credentials:
  1. Go to https://api.slack.com/apps
  2. Create a new app or select an existing one
  3. Go to 'Basic Information' > 'App Credentials'
  4. Copy the Client ID and Client Secret

Also configure OAuth redirect URL:
  1. Go to 'OAuth & Permissions'
  2. Add redirect URL: http://localhost:%d/slack/callback`, port)
	}

	config := slack.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Port:         port,
		Scopes:       scopes,
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Starting Slack OAuth flow..."))
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "A browser window will open for authorization.")
	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Waiting for authorization (timeout: 5 minutes)..."))
	_, _ = fmt.Fprintln(os.Stdout, "")

	result, err := slack.RunOAuthFlow(cmd.Context(), config, core.OpenBrowser)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Authorization successful!"))
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Display connection info
	_, _ = fmt.Fprintf(os.Stdout, "  Workspace:  %s (%s)\n", result.Team.Name, result.Team.ID)
	_, _ = fmt.Fprintf(os.Stdout, "  Bot User:   %s\n", result.BotUserID)
	_, _ = fmt.Fprintf(os.Stdout, "  App ID:     %s\n", result.AppID)
	_, _ = fmt.Fprintf(os.Stdout, "  Scopes:     %s\n", result.Scope)
	_, _ = fmt.Fprintln(os.Stdout, "")

	// Save token if requested
	if saveToken {
		pm, err := core.NewProfileManager()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, warnStyle.Render("Warning: Could not connect to server: %v\n"), err)
			_, _ = fmt.Fprintf(os.Stdout, "\nManually save your token:\n")
			_, _ = fmt.Fprintf(os.Stdout, "  export SLACK_TOKEN=%s\n", result.AccessToken)

			return nil
		}

		// Get target profile
		var profile *model.Profile

		if profileName != "" {
			profile, err = pm.GetProfile(profileName)
			if err != nil {
				return fmt.Errorf("failed to get profile %q: %w", profileName, err)
			}
		} else {
			profile, err = pm.GetActiveProfile()
			if err != nil {
				return fmt.Errorf("no active profile; use --profile to specify one or create a profile first")
			}
		}

		// Prepare default channel
		targetChannel := channel
		if targetChannel == "" {
			targetChannel = "#general"
		}

		// Create NotifyChannel for Slack
		notifyChannel := &model.NotifyChannel{
			ID:   channelName,
			Type: model.ChannelSlack,
			Name: fmt.Sprintf("Slack - %s", result.Team.Name),
			Config: map[string]string{
				"bot_token":       result.AccessToken,
				"default_channel": targetChannel,
				"workspace_id":    result.Team.ID,
				"workspace_name":  result.Team.Name,
				"bot_user_id":     result.BotUserID,
				"app_id":          result.AppID,
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save to profile (AddNotifyChannel encrypts sensitive fields)
		if err := pm.AddNotifyChannel(profile.Name, notifyChannel); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, warnStyle.Render("Warning: Could not save token: %v\n"), err)
			_, _ = fmt.Fprintf(os.Stdout, "\nManually save your token:\n")
			_, _ = fmt.Fprintf(os.Stdout, "  export SLACK_TOKEN=%s\n", result.AccessToken)

			return nil
		}

		_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Token saved to profile %q!", profile.Name)))
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "You can now use Slack commands:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack channels")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack messages --channel general")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack search \"keyword\"")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Your bot token (save it securely):\n")
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", result.AccessToken)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "To use this token:")
		_, _ = fmt.Fprintf(os.Stdout, "  export SLACK_TOKEN=%s\n", result.AccessToken)
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Or connect with OAuth to save to profile:")
		_, _ = fmt.Fprintln(os.Stdout, "  clonr pm slack connect --client-id <id> --client-secret <secret> --save")
	}

	return nil
}

// resolveSlackChannelID resolves a channel name to its ID.
func resolveSlackChannelID(ctx context.Context, client *slack.Client, channel string, quiet bool) (string, error) {
	// If it looks like an ID (starts with C, G, or D), use it directly
	if len(channel) > 0 && (channel[0] == 'C' || channel[0] == 'G' || channel[0] == 'D') {
		return channel, nil
	}

	// Strip # prefix if present
	channel = strings.TrimPrefix(channel, "#")

	if !quiet {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("Looking up channel #%s...\n"), channel)
	}

	// List channels to find the ID
	result, err := client.ListChannels(ctx, slack.ListChannelsOptions{
		Types: "public_channel,private_channel",
		Limit: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range result.Channels {
		if ch.Name == channel {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel #%s not found", channel)
}

// parseDuration parses duration strings like "24h", "7d", "2w".
func parseDuration(s string) (time.Duration, error) {
	// Check for days/weeks suffix
	if num, found := strings.CutSuffix(s, "d"); found {
		days, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}

		return time.Duration(days) * 24 * time.Hour, nil
	}

	if num, found := strings.CutSuffix(s, "w"); found {
		weeks, err := strconv.Atoi(num)
		if err != nil {
			return 0, err
		}

		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// Standard Go duration
	return time.ParseDuration(s)
}
