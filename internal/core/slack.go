package core

import (
	"context"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/notify"
	"github.com/inovacc/clonr/internal/store"
)

const (
	// slackProfileName is used as a profile name for Slack encryption
	slackProfileName = "__slack__"
	// slackHost is used as a host for Slack encryption
	slackHost = "slack.com"
)

// SlackManager handles Slack integration operations.
type SlackManager struct {
	db store.Store
}

// NewSlackManager creates a new SlackManager.
func NewSlackManager() (*SlackManager, error) {
	db := store.GetDB()
	return &SlackManager{db: db}, nil
}

// GetConfig retrieves the current Slack configuration.
func (m *SlackManager) GetConfig() (*model.SlackConfig, error) {
	return m.db.GetSlackConfig()
}

// AddWebhook adds a Slack webhook integration.
func (m *SlackManager) AddWebhook(webhookURL, channel string) error {
	// Validate webhook URL
	if err := notify.ValidateWebhookURL(webhookURL); err != nil {
		return err
	}

	// Encrypt the webhook URL
	encryptedURL, err := tpm.EncryptToken(webhookURL, slackProfileName, slackHost)
	if err != nil {
		return fmt.Errorf("failed to encrypt webhook URL: %w", err)
	}

	// Get existing config or create new one
	config, err := m.db.GetSlackConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		config = &model.SlackConfig{
			Enabled:   true,
			Events:    model.DefaultSlackEvents(),
			CreatedAt: time.Now(),
		}
	}

	config.EncryptedWebhookURL = encryptedURL
	config.DefaultChannel = channel
	config.Enabled = true
	config.UpdatedAt = time.Now()

	if err := m.db.SaveSlackConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// AddBot adds a Slack bot integration.
func (m *SlackManager) AddBot(botToken, channel string) error {
	// Validate bot token
	if err := notify.ValidateBotToken(botToken); err != nil {
		return err
	}

	// Encrypt the bot token
	encryptedToken, err := tpm.EncryptToken(botToken, slackProfileName, slackHost)
	if err != nil {
		return fmt.Errorf("failed to encrypt bot token: %w", err)
	}

	// Get existing config or create new one
	config, err := m.db.GetSlackConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		config = &model.SlackConfig{
			Enabled:   true,
			Events:    model.DefaultSlackEvents(),
			CreatedAt: time.Now(),
		}
	}

	config.EncryptedBotToken = encryptedToken
	config.DefaultChannel = channel
	config.BotEnabled = true
	config.Enabled = true
	config.UpdatedAt = time.Now()

	if err := m.db.SaveSlackConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Remove removes the Slack integration.
func (m *SlackManager) Remove() error {
	return m.db.DeleteSlackConfig()
}

// Enable enables Slack notifications.
func (m *SlackManager) Enable() error {
	return m.db.EnableSlackNotifications()
}

// Disable disables Slack notifications.
func (m *SlackManager) Disable() error {
	return m.db.DisableSlackNotifications()
}

// Test sends a test notification.
func (m *SlackManager) Test(ctx context.Context, channel string) error {
	config, err := m.db.GetSlackConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("slack is not configured")
	}

	// Use provided channel or default
	targetChannel := channel
	if targetChannel == "" {
		targetChannel = config.DefaultChannel
	}

	// Try webhook first
	if len(config.EncryptedWebhookURL) > 0 {
		webhookURL, err := tpm.DecryptToken(config.EncryptedWebhookURL, slackProfileName, slackHost)
		if err != nil {
			return fmt.Errorf("failed to decrypt webhook URL: %w", err)
		}

		return notify.TestWebhook(ctx, webhookURL, targetChannel)
	}

	// Try bot token
	if len(config.EncryptedBotToken) > 0 {
		botToken, err := tpm.DecryptToken(config.EncryptedBotToken, slackProfileName, slackHost)
		if err != nil {
			return fmt.Errorf("failed to decrypt bot token: %w", err)
		}

		return notify.TestBotToken(ctx, botToken, targetChannel)
	}

	return fmt.Errorf("no webhook URL or bot token configured")
}

// ConfigureEvents updates the event configuration.
func (m *SlackManager) ConfigureEvents(events []model.SlackEventConfig) error {
	config, err := m.db.GetSlackConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil {
		return fmt.Errorf("slack is not configured")
	}

	config.Events = events
	config.UpdatedAt = time.Now()

	return m.db.SaveSlackConfig(config)
}

// GetSender creates a Slack sender from the current configuration.
func (m *SlackManager) GetSender() (notify.Sender, error) {
	config, err := m.db.GetSlackConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if config == nil || !config.Enabled {
		return nil, nil
	}

	// Build event configs map
	eventConfigs := make(map[string]notify.SlackEventConfig)
	for _, e := range config.Events {
		eventConfigs[e.Event] = notify.SlackEventConfig{
			Enabled:  e.Enabled,
			Channel:  e.Channel,
			Priority: e.Priority,
			Filters:  e.Filters,
		}
	}

	opts := []notify.SlackOption{
		notify.WithDefaultChannel(config.DefaultChannel),
		notify.WithEventConfigs(eventConfigs),
	}

	// Add webhook if configured
	if len(config.EncryptedWebhookURL) > 0 {
		webhookURL, err := tpm.DecryptToken(config.EncryptedWebhookURL, slackProfileName, slackHost)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt webhook URL: %w", err)
		}

		opts = append(opts, notify.WithWebhook(webhookURL))
	}

	// Add bot token if configured
	if len(config.EncryptedBotToken) > 0 && config.BotEnabled {
		botToken, err := tpm.DecryptToken(config.EncryptedBotToken, slackProfileName, slackHost)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt bot token: %w", err)
		}

		opts = append(opts, notify.WithBotToken(botToken))
	}

	return notify.NewSlackSender(opts...), nil
}

// InitializeNotifications sets up the global notification dispatcher with Slack.
func InitializeNotifications() error {
	manager, err := NewSlackManager()
	if err != nil {
		return err
	}

	sender, err := manager.GetSender()
	if err != nil {
		return err
	}

	if sender != nil {
		notify.GetDispatcher().Register(sender)
	}

	return nil
}
