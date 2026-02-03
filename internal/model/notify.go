package model

import "time"

// ChannelType represents the type of notification channel.
type ChannelType string

const (
	ChannelSlack   ChannelType = "slack"
	ChannelTeams   ChannelType = "teams"
	ChannelDiscord ChannelType = "discord"
	ChannelEmail   ChannelType = "email"
	ChannelWebhook ChannelType = "webhook"
)

// NotifyChannel represents a notification channel attached to a profile.
// All sensitive data (tokens, webhooks, passwords) are encrypted with the profile.
type NotifyChannel struct {
	// ID is the unique identifier for this channel
	ID string `json:"id"`

	// Type is the channel type (slack, teams, discord, email, webhook)
	Type ChannelType `json:"type"`

	// Name is a user-friendly name for this channel
	Name string `json:"name"`

	// Config contains type-specific configuration (all encrypted with profile)
	// For Slack: webhook_url, bot_token, default_channel
	// For Teams: webhook_url, connector_token
	// For Discord: webhook_url, bot_token, default_channel
	// For Email: provider, host, port, username, password, api_key, from, to
	// For Webhook: url, method, headers, hmac_secret, template
	Config map[string]string `json:"config"`

	// Events contains the event configuration for this channel
	Events []EventConfig `json:"events"`

	// Enabled indicates if this channel is active
	Enabled bool `json:"enabled"`

	// CreatedAt is when the channel was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the channel was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// EventConfig defines which events trigger notifications on a channel.
type EventConfig struct {
	// Event is the event type (push, pr-merge, ci-fail, etc.)
	Event string `json:"event"`

	// Filters are patterns to match (branch:main, repo:critical-*)
	Filters []string `json:"filters,omitempty"`

	// Target is the destination (channel name, email address, etc.)
	Target string `json:"target,omitempty"`

	// Priority is the notification priority (low, normal, high)
	Priority string `json:"priority,omitempty"`

	// Template is the custom template name to use
	Template string `json:"template,omitempty"`
}

// Supported event types for notifications.
const (
	EventClone     = "clone"
	EventPush      = "push"
	EventPull      = "pull"
	EventCommit    = "commit"
	EventPRCreate  = "pr-create"
	EventPRMerge   = "pr-merge"
	EventCIPass    = "ci-pass"
	EventCIFail    = "ci-fail"
	EventRelease   = "release"
	EventSync      = "sync"
	EventError     = "error"
)

// Notification priorities.
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)
