package model

import "time"

// SlackConfig represents global Slack integration configuration.
// This stores the primary Slack workspace connection used for notifications.
type SlackConfig struct {
	// ID is the unique identifier (always 1 for singleton config)
	ID int `json:"id"`

	// Enabled indicates if Slack notifications are active
	Enabled bool `json:"enabled"`

	// WorkspaceID is the Slack workspace identifier
	WorkspaceID string `json:"workspace_id,omitempty"`

	// WorkspaceName is the human-readable workspace name
	WorkspaceName string `json:"workspace_name,omitempty"`

	// EncryptedWebhookURL stores the TPM-encrypted webhook URL
	EncryptedWebhookURL []byte `json:"encrypted_webhook_url,omitempty"`

	// EncryptedBotToken stores the TPM-encrypted bot token
	EncryptedBotToken []byte `json:"encrypted_bot_token,omitempty"`

	// DefaultChannel is the default channel for notifications (e.g., "#dev")
	DefaultChannel string `json:"default_channel,omitempty"`

	// BotEnabled indicates if the bot integration is active (vs. webhook only)
	BotEnabled bool `json:"bot_enabled"`

	// Events configures which events trigger notifications
	Events []SlackEventConfig `json:"events,omitempty"`

	// CreatedAt is when the config was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the config was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// SlackEventConfig defines notification settings for a specific event type.
type SlackEventConfig struct {
	// Event is the event type (push, clone, pull, commit, etc.)
	Event string `json:"event"`

	// Enabled indicates if notifications are sent for this event
	Enabled bool `json:"enabled"`

	// Channel overrides the default channel for this event
	Channel string `json:"channel,omitempty"`

	// Priority affects message formatting (low, normal, high)
	Priority string `json:"priority,omitempty"`

	// Filters are patterns to match (e.g., "branch:main", "repo:critical-*")
	Filters []string `json:"filters,omitempty"`
}

// DefaultSlackEvents returns the default event configuration for Slack.
func DefaultSlackEvents() []SlackEventConfig {
	return []SlackEventConfig{
		{Event: EventPush, Enabled: true, Priority: PriorityNormal},
		{Event: EventClone, Enabled: false, Priority: PriorityLow},
		{Event: EventPull, Enabled: false, Priority: PriorityLow},
		{Event: EventCommit, Enabled: false, Priority: PriorityLow},
		{Event: EventPRCreate, Enabled: true, Priority: PriorityNormal},
		{Event: EventPRMerge, Enabled: true, Priority: PriorityNormal},
		{Event: EventCIPass, Enabled: false, Priority: PriorityLow},
		{Event: EventCIFail, Enabled: true, Priority: PriorityHigh},
		{Event: EventRelease, Enabled: true, Priority: PriorityNormal},
		{Event: EventError, Enabled: true, Priority: PriorityHigh},
	}
}
