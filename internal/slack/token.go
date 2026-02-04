package slack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/inovacc/clonr/internal/auth"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
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
// Priority: flagValue -> environment variables -> stored config -> config file.
func ResolveSlackToken(flagValue string) (token, source string, err error) {
	result, err := auth.NewResolver("Slack Bot Token").
		WithFlagValue(flagValue).
		WithEnvs(SlackTokenEnvVar, SlackBotTokenEnvVar).
		WithProvider(storedTokenProvider()).
		WithProvider(configFileProvider()).
		WithHelpMessage(fmt.Sprintf("Get a Slack Bot Token at: %s", SlackTokenURL)).
		Resolve()
	if err != nil {
		return "", "", err
	}

	return result.Token, string(result.Source), nil
}

// storedTokenProvider provides the token from the stored Slack config.
func storedTokenProvider() auth.TokenProvider {
	return func() (string, string, error) {
		manager, err := core.NewSlackManager()
		if err != nil {
			return "", "", err
		}

		config, err := manager.GetConfig()
		if err != nil {
			return "", "", err
		}

		if config == nil || len(config.EncryptedBotToken) == 0 {
			return "", "", nil
		}

		token, err := tpm.DecryptToken(config.EncryptedBotToken, "__slack__", "slack.com")
		if err != nil {
			return "", "", err
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
