package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SlackSender sends notifications to Slack via webhook or bot API.
type SlackSender struct {
	webhookURL     string
	botToken       string
	defaultChannel string
	httpClient     *http.Client
	eventConfigs   map[string]SlackEventConfig
}

// SlackEventConfig mirrors model.SlackEventConfig for internal use.
type SlackEventConfig struct {
	Enabled  bool
	Channel  string
	Priority string
	Filters  []string
}

// SlackOption configures a SlackSender.
type SlackOption func(*SlackSender)

// WithWebhook sets the webhook URL.
func WithWebhook(url string) SlackOption {
	return func(s *SlackSender) {
		s.webhookURL = url
	}
}

// WithBotToken sets the bot token for the Slack API.
func WithBotToken(token string) SlackOption {
	return func(s *SlackSender) {
		s.botToken = token
	}
}

// WithDefaultChannel sets the default channel.
func WithDefaultChannel(channel string) SlackOption {
	return func(s *SlackSender) {
		s.defaultChannel = channel
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) SlackOption {
	return func(s *SlackSender) {
		s.httpClient = client
	}
}

// WithEventConfigs sets the event configurations.
func WithEventConfigs(configs map[string]SlackEventConfig) SlackOption {
	return func(s *SlackSender) {
		s.eventConfigs = configs
	}
}

// NewSlackSender creates a new Slack notification sender.
func NewSlackSender(opts ...SlackOption) *SlackSender {
	s := &SlackSender{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		eventConfigs: make(map[string]SlackEventConfig),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Name returns the sender name.
func (s *SlackSender) Name() string {
	return "slack"
}

// Send sends a notification for the given event.
func (s *SlackSender) Send(ctx context.Context, event *Event) error {
	// Check if this event type is enabled
	config, ok := s.eventConfigs[event.Type]
	if ok && !config.Enabled {
		return nil // Event type disabled, skip silently
	}

	// Check filters
	if ok && len(config.Filters) > 0 {
		if !s.matchFilters(event, config.Filters) {
			return nil // Doesn't match filters, skip silently
		}
	}

	// Determine target channel
	channel := s.defaultChannel
	if ok && config.Channel != "" {
		channel = config.Channel
	}

	// Format the message
	msg := FormatSlackMessage(event, channel)

	// Send via webhook or bot API
	if s.webhookURL != "" {
		return s.sendWebhook(ctx, msg)
	}

	if s.botToken != "" {
		return s.sendBotAPI(ctx, msg)
	}

	return fmt.Errorf("no webhook URL or bot token configured")
}

// Test sends a test notification.
func (s *SlackSender) Test(ctx context.Context) error {
	msg := FormatTestMessage(s.defaultChannel)

	if s.webhookURL != "" {
		return s.sendWebhook(ctx, msg)
	}

	if s.botToken != "" {
		return s.sendBotAPI(ctx, msg)
	}

	return fmt.Errorf("no webhook URL or bot token configured")
}

// sendWebhook sends a message via Slack webhook.
func (s *SlackSender) sendWebhook(ctx context.Context, msg *SlackMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// sendBotAPI sends a message via the Slack Bot API.
func (s *SlackSender) sendBotAPI(ctx context.Context, msg *SlackMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.botToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// matchFilters checks if an event matches the configured filters.
func (s *SlackSender) matchFilters(event *Event, filters []string) bool {
	for _, filter := range filters {
		parts := strings.SplitN(filter, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key, pattern := parts[0], parts[1]

		switch key {
		case "branch":
			if !matchPattern(event.Branch, pattern) {
				return false
			}
		case "repo":
			if !matchPattern(event.Repository, pattern) {
				return false
			}
		case "author":
			if !matchPattern(event.Author, pattern) {
				return false
			}
		case "workspace":
			if !matchPattern(event.Workspace, pattern) {
				return false
			}
		case "profile":
			if !matchPattern(event.Profile, pattern) {
				return false
			}
		}
	}

	return true
}

// matchPattern performs simple wildcard matching.
func matchPattern(value, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Handle suffix wildcard (e.g., "feature-*")
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		return strings.HasPrefix(value, prefix)
	}

	// Handle prefix wildcard (e.g., "*-fix")
	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		return strings.HasSuffix(value, suffix)
	}

	// Exact match
	return value == pattern
}

// ValidateWebhookURL checks if a webhook URL is valid.
func ValidateWebhookURL(url string) error {
	if url == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if !strings.HasPrefix(url, "https://hooks.slack.com/services/") {
		return fmt.Errorf("invalid Slack webhook URL: must start with https://hooks.slack.com/services/")
	}

	return nil
}

// ValidateBotToken checks if a bot token is valid format.
func ValidateBotToken(token string) error {
	if token == "" {
		return fmt.Errorf("bot token is required")
	}

	if !strings.HasPrefix(token, "xoxb-") {
		return fmt.Errorf("invalid bot token: must start with xoxb-")
	}

	return nil
}

// TestWebhook sends a test message to verify webhook configuration.
func TestWebhook(ctx context.Context, webhookURL, channel string) error {
	sender := NewSlackSender(
		WithWebhook(webhookURL),
		WithDefaultChannel(channel),
	)

	return sender.Test(ctx)
}

// TestBotToken sends a test message to verify bot token configuration.
func TestBotToken(ctx context.Context, botToken, channel string) error {
	sender := NewSlackSender(
		WithBotToken(botToken),
		WithDefaultChannel(channel),
	)

	return sender.Test(ctx)
}
