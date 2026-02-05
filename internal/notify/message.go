package notify

import (
	"fmt"
	"strings"
	"time"
)

// SlackMessage represents a Slack message with optional Block Kit formatting.
type SlackMessage struct {
	// Channel is the target channel (e.g., "#dev" or "C1234567890")
	Channel string `json:"channel,omitempty"`

	// Text is the fallback text for notifications
	Text string `json:"text"`

	// Blocks contains Block Kit blocks for rich formatting
	Blocks []Block `json:"blocks,omitempty"`

	// Attachments contains legacy attachments (for color bars)
	Attachments []Attachment `json:"attachments,omitempty"`

	// UnfurlLinks disables link previews
	UnfurlLinks bool `json:"unfurl_links"`

	// UnfurlMedia disables media previews
	UnfurlMedia bool `json:"unfurl_media"`
}

// Block represents a Slack Block Kit block.
type Block struct {
	Type      string       `json:"type"`
	Text      *TextObject  `json:"text,omitempty"`
	Elements  []Element    `json:"elements,omitempty"`
	Fields    []TextObject `json:"fields,omitempty"`
	Accessory *Element     `json:"accessory,omitempty"`
}

// TextObject represents text content in a block.
type TextObject struct {
	Type  string `json:"type"` // "plain_text" or "mrkdwn"
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// Element represents an interactive element or context item.
type Element struct {
	Type     string      `json:"type"`
	Text     *TextObject `json:"text,omitempty"`
	URL      string      `json:"url,omitempty"`
	ImageURL string      `json:"image_url,omitempty"`
	AltText  string      `json:"alt_text,omitempty"`
}

// Attachment represents a legacy Slack attachment (used for color bars).
type Attachment struct {
	Color    string  `json:"color,omitempty"`
	Fallback string  `json:"fallback,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
}

// FormatSlackMessage creates a Slack message from an event.
func FormatSlackMessage(event *Event, channel string) *SlackMessage {
	msg := &SlackMessage{
		Channel:     channel,
		UnfurlLinks: false,
		UnfurlMedia: false,
	}

	// Determine color based on event type and success
	color := getEventColor(event)

	// Build the message based on event type
	switch event.Type {
	case EventPush:
		msg.Text = formatPushText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatPushBlocks(event),
		}}
	case EventClone:
		msg.Text = formatCloneText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatCloneBlocks(event),
		}}
	case EventPull:
		msg.Text = formatPullText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatPullBlocks(event),
		}}
	case EventCommit:
		msg.Text = formatCommitText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatCommitBlocks(event),
		}}
	case EventCIFail:
		msg.Text = formatCIFailText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatCIFailBlocks(event),
		}}
	case EventCIPass:
		msg.Text = formatCIPassText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatCIPassBlocks(event),
		}}
	case EventError:
		msg.Text = formatErrorText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatErrorBlocks(event),
		}}
	default:
		msg.Text = formatGenericText(event)
		msg.Attachments = []Attachment{{
			Color:  color,
			Blocks: formatGenericBlocks(event),
		}}
	}

	return msg
}

// getEventColor returns the color for an event.
func getEventColor(event *Event) string {
	if !event.Success {
		return "#E01E5A" // Red for errors
	}

	switch event.Type {
	case EventCIFail, EventError:
		return "#E01E5A" // Red
	case EventCIPass:
		return "#2EB67D" // Green
	case EventPush, EventCommit:
		return "#36C5F0" // Blue
	case EventClone, EventPull:
		return "#4A154B" // Purple
	default:
		return "#ECB22E" // Yellow/Orange
	}
}

// formatPushText creates the fallback text for a push event.
func formatPushText(event *Event) string {
	return fmt.Sprintf("[%s] %s pushed to %s", event.Repository, event.Author, event.Branch)
}

// formatPushBlocks creates Block Kit blocks for a push event.
func formatPushBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Push to %s*\n`%s` branch", event.Repository, event.Branch),
			},
		},
	}

	// Add commit info if available
	if event.Commit != "" {
		commitText := truncate(event.CommitMessage, 100)
		if commitText == "" {
			commitText = "No commit message"
		}

		shortSHA := event.Commit
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		blocks = append(blocks, Block{
			Type: "section",
			Fields: []TextObject{
				{Type: "mrkdwn", Text: fmt.Sprintf("*Commit*\n`%s`", shortSHA)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Author*\n%s", event.Author)},
			},
		})

		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf(">%s", commitText),
			},
		})
	}

	// Add link button if URL is available
	if event.URL != "" {
		blocks = append(blocks, Block{
			Type: "actions",
			Elements: []Element{
				{
					Type: "button",
					Text: &TextObject{Type: "plain_text", Text: "View on GitHub", Emoji: true},
					URL:  event.URL,
				},
			},
		})
	}

	// Add context with timestamp
	blocks = append(blocks, formatContextBlock(event))

	return blocks
}

// formatCloneText creates the fallback text for a clone event.
func formatCloneText(event *Event) string {
	return fmt.Sprintf("[%s] Repository cloned", event.Repository)
}

// formatCloneBlocks creates Block Kit blocks for a clone event.
func formatCloneBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Repository Cloned*\n%s", event.Repository),
			},
		},
	}

	if event.Extra["path"] != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Location*\n`%s`", event.Extra["path"]),
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatPullText creates the fallback text for a pull event.
func formatPullText(event *Event) string {
	return fmt.Sprintf("[%s] Pulled latest changes", event.Repository)
}

// formatPullBlocks creates Block Kit blocks for a pull event.
func formatPullBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Pull Completed*\n%s", event.Repository),
			},
		},
	}

	if event.Branch != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Branch*\n`%s`", event.Branch),
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatCommitText creates the fallback text for a commit event.
func formatCommitText(event *Event) string {
	return fmt.Sprintf("[%s] New commit: %s", event.Repository, truncate(event.CommitMessage, 50))
}

// formatCommitBlocks creates Block Kit blocks for a commit event.
func formatCommitBlocks(event *Event) []Block {
	shortSHA := event.Commit
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*New Commit*\n%s", event.Repository),
			},
		},
		{
			Type: "section",
			Fields: []TextObject{
				{Type: "mrkdwn", Text: fmt.Sprintf("*Commit*\n`%s`", shortSHA)},
				{Type: "mrkdwn", Text: fmt.Sprintf("*Author*\n%s", event.Author)},
			},
		},
	}

	if event.CommitMessage != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf(">%s", truncate(event.CommitMessage, 200)),
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatCIFailText creates the fallback text for a CI fail event.
func formatCIFailText(event *Event) string {
	return fmt.Sprintf("[%s] CI Failed: %s", event.Repository, event.Error)
}

// formatCIFailBlocks creates Block Kit blocks for a CI fail event.
func formatCIFailBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf(":x: *CI Failed*\n%s", event.Repository),
			},
		},
	}

	if event.Error != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Error*\n```%s```", truncate(event.Error, 500)),
			},
		})
	}

	if event.URL != "" {
		blocks = append(blocks, Block{
			Type: "actions",
			Elements: []Element{
				{
					Type: "button",
					Text: &TextObject{Type: "plain_text", Text: "View Workflow", Emoji: true},
					URL:  event.URL,
				},
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatCIPassText creates the fallback text for a CI pass event.
func formatCIPassText(event *Event) string {
	return fmt.Sprintf("[%s] CI Passed", event.Repository)
}

// formatCIPassBlocks creates Block Kit blocks for a CI pass event.
func formatCIPassBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf(":white_check_mark: *CI Passed*\n%s", event.Repository),
			},
		},
	}

	if event.URL != "" {
		blocks = append(blocks, Block{
			Type: "actions",
			Elements: []Element{
				{
					Type: "button",
					Text: &TextObject{Type: "plain_text", Text: "View Workflow", Emoji: true},
					URL:  event.URL,
				},
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatErrorText creates the fallback text for an error event.
func formatErrorText(event *Event) string {
	return fmt.Sprintf("[clonr] Error: %s", event.Error)
}

// formatErrorBlocks creates Block Kit blocks for an error event.
func formatErrorBlocks(event *Event) []Block {
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: ":warning: *Error*",
			},
		},
	}

	if event.Error != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("```%s```", truncate(event.Error, 500)),
			},
		})
	}

	if event.Repository != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Repository*\n%s", event.Repository),
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatGenericText creates the fallback text for a generic event.
func formatGenericText(event *Event) string {
	if event.Repository != "" {
		return fmt.Sprintf("[%s] %s", event.Repository, event.Type)
	}
	return fmt.Sprintf("[clonr] %s", event.Type)
}

// formatGenericBlocks creates Block Kit blocks for a generic event.
func formatGenericBlocks(event *Event) []Block {
	title := toTitle(strings.ReplaceAll(event.Type, "-", " "))
	blocks := []Block{
		{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*%s*", title),
			},
		},
	}

	if event.Repository != "" {
		blocks = append(blocks, Block{
			Type: "section",
			Text: &TextObject{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Repository*\n%s", event.Repository),
			},
		})
	}

	blocks = append(blocks, formatContextBlock(event))
	return blocks
}

// formatContextBlock creates a context block with timestamp and profile info.
func formatContextBlock(event *Event) Block {
	elements := []Element{
		{
			Type:    "mrkdwn",
			AltText: event.Timestamp.Format(time.RFC3339),
		},
	}

	contextText := fmt.Sprintf("<!date^%d^{date_short_pretty} at {time}|%s>",
		event.Timestamp.Unix(),
		event.Timestamp.Format("Jan 2, 2006 3:04 PM"))

	if event.Profile != "" {
		contextText += fmt.Sprintf(" | Profile: %s", event.Profile)
	}

	if event.Workspace != "" {
		contextText += fmt.Sprintf(" | Workspace: %s", event.Workspace)
	}

	elements[0] = Element{
		Type: "mrkdwn",
		Text: &TextObject{Type: "mrkdwn", Text: contextText},
	}

	return Block{
		Type:     "context",
		Elements: elements,
	}
}

// toTitle converts a string to title case.
func toTitle(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// truncate shortens a string to the specified length.
func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	// Remove newlines for single-line display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatTestMessage creates a test message for Slack.
func FormatTestMessage(channel string) *SlackMessage {
	return &SlackMessage{
		Channel: channel,
		Text:    "Test notification from clonr",
		Blocks: []Block{
			{
				Type: "section",
				Text: &TextObject{
					Type: "mrkdwn",
					Text: ":wave: *Test Notification*\nYour Slack integration is working correctly!",
				},
			},
			{
				Type: "section",
				Text: &TextObject{
					Type: "mrkdwn",
					Text: "This message was sent by `clonr slack test` to verify your configuration.",
				},
			},
			{
				Type: "context",
				Elements: []Element{
					{
						Type: "mrkdwn",
						Text: &TextObject{
							Type: "mrkdwn",
							Text: fmt.Sprintf("Sent at <!date^%d^{date_short_pretty} at {time}|%s>",
								time.Now().Unix(),
								time.Now().Format("Jan 2, 2006 3:04 PM")),
						},
					},
				},
			},
		},
	}
}
