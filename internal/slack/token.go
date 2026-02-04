package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inovacc/clonr/internal/auth"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
)

const (
	// SlackTokenEnvVar is the environment variable for Slack token.
	SlackTokenEnvVar = "SLACK_TOKEN"
	// SlackBotTokenEnvVar is the alternative environment variable.
	SlackBotTokenEnvVar = "SLACK_BOT_TOKEN"
	// SlackTokenURL is the URL to create Slack tokens.
	SlackTokenURL = "https://api.slack.com/apps"
)

// slackConfig represents the Slack config file structure.
type slackConfig struct {
	Token string `json:"token"`
}

// ResolveSlackToken resolves a Slack token from various sources.
// Priority: flagValue -> env vars -> active profile -> global config -> config file.
func ResolveSlackToken(flagValue string) (token, source string, err error) {
	result, err := auth.NewResolver("Slack Bot Token").
		WithFlagValue(flagValue).
		WithEnvs(SlackTokenEnvVar, SlackBotTokenEnvVar).
		WithProvider(profileTokenProvider()).
		WithProvider(storedTokenProvider()).
		WithProvider(configFileProvider()).
		WithHelpMessage(fmt.Sprintf("Get a Slack Bot Token at: %s\n\nOr connect via OAuth:\n  clonr pm slack connect --client-id <id> --client-secret <secret>", SlackTokenURL)).
		Resolve()
	if err != nil {
		return "", "", err
	}

	return result.Token, string(result.Source), nil
}

// profileTokenProvider provides the token from the active profile's Slack channel.
func profileTokenProvider() auth.TokenProvider {
	return func() (string, string, error) {
		pm, err := core.NewProfileManager()
		if err != nil {
			return "", "", nil //nolint:nilerr // Skip if profile manager unavailable
		}

		profile, err := pm.GetActiveProfile()
		if err != nil || profile == nil {
			return "", "", nil
		}

		// Find Slack channel in profile
		channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelSlack)
		if err != nil || channel == nil {
			return "", "", nil
		}

		// Decrypt channel config
		config, err := pm.DecryptChannelConfig(profile.Name, channel)
		if err != nil {
			return "", "", nil //nolint:nilerr // Skip if decryption fails
		}

		// Get bot_token from config
		if token, ok := config["bot_token"]; ok && token != "" {
			return token, fmt.Sprintf("profile:%s", profile.Name), nil
		}

		return "", "", nil
	}
}

// storedTokenProvider provides the token from the stored Slack config (legacy).
func storedTokenProvider() auth.TokenProvider {
	return func() (string, string, error) {
		manager, err := core.NewSlackManager()
		if err != nil {
			return "", "", nil //nolint:nilerr // Skip if manager unavailable
		}

		config, err := manager.GetConfig()
		if err != nil {
			return "", "", nil //nolint:nilerr // Skip if config unavailable
		}

		if config == nil || len(config.EncryptedBotToken) == 0 {
			return "", "", nil
		}

		token, err := tpm.DecryptToken(config.EncryptedBotToken, "__slack__", "slack.com")
		if err != nil {
			return "", "", nil //nolint:nilerr // Skip if decryption fails
		}

		return token, "stored-config", nil
	}
}

// configFileProvider provides the token from a config file.
func configFileProvider() auth.TokenProvider {
	return func() (string, string, error) {
		configDir, err := os.UserConfigDir()
		if err != nil {
			// Can't get config dir, skip this provider
			return "", "", nil //nolint:nilerr // Intentional: skip provider if config dir unavailable
		}

		configPath := filepath.Join(configDir, "clonr", "slack.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Config file doesn't exist, skip this provider
			return "", "", nil
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			// Can't read config file, skip this provider
			return "", "", nil //nolint:nilerr // Intentional: skip provider if file unreadable
		}

		var cfg slackConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			// Invalid JSON, skip this provider
			return "", "", nil //nolint:nilerr // Intentional: skip provider if JSON invalid
		}

		return cfg.Token, "slack.json", nil
	}
}
