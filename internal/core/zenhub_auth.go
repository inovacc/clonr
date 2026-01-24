package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ZenHubTokenSource indicates where the ZenHub token was found
type ZenHubTokenSource string

const (
	ZenHubTokenSourceFlag   ZenHubTokenSource = "flag"
	ZenHubTokenSourceEnv    ZenHubTokenSource = "ZENHUB_TOKEN"
	ZenHubTokenSourceConfig ZenHubTokenSource = "config"
	ZenHubTokenSourceNone   ZenHubTokenSource = "none"
)

// ZenHubConfig represents the ZenHub configuration file structure
type ZenHubConfig struct {
	Token            string `json:"token"`
	DefaultWorkspace string `json:"default_workspace,omitempty"`
}

// ResolveZenHubToken attempts to find a ZenHub token from multiple sources.
// Priority order:
//  1. flagToken (explicit --token flag)
//  2. ZENHUB_TOKEN environment variable
//  3. ~/.config/clonr/zenhub.json config file
func ResolveZenHubToken(flagToken string) (token string, source ZenHubTokenSource, err error) {
	// 1. Flag has highest priority
	if flagToken != "" {
		return flagToken, ZenHubTokenSourceFlag, nil
	}

	// 2. Check ZENHUB_TOKEN env var
	if token = os.Getenv("ZENHUB_TOKEN"); token != "" {
		return token, ZenHubTokenSourceEnv, nil
	}

	// 3. Try config file
	configToken, err := loadZenHubConfigToken()
	if err == nil && configToken != "" {
		return configToken, ZenHubTokenSourceConfig, nil
	}

	// 4. No token found
	return "", ZenHubTokenSourceNone, fmt.Errorf(`ZenHub API token required

Provide a token via one of:
  * ZENHUB_TOKEN env var     (recommended)
  * --token flag
  * ~/.config/clonr/zenhub.json config file

Get your ZenHub API token at: https://app.zenhub.com/settings/tokens`)
}

// loadZenHubConfigToken loads token from the ZenHub config file
func loadZenHubConfigToken() (string, error) {
	configPath, err := getZenHubConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("failed to read ZenHub config: %w", err)
	}

	var config ZenHubConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse ZenHub config: %w", err)
	}

	// Handle token reference to env var
	if strings.HasPrefix(config.Token, "env:") {
		envVar := strings.TrimPrefix(config.Token, "env:")
		return os.Getenv(envVar), nil
	}

	return config.Token, nil
}

// getZenHubConfigPath returns the path to the ZenHub config file
func getZenHubConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine config directory: %w", err)
		}

		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "clonr", "zenhub.json"), nil
}

// GetZenHubDefaultWorkspace returns the default workspace ID from config
func GetZenHubDefaultWorkspace() (string, error) {
	configPath, err := getZenHubConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", nil // No config file, no default workspace
	}

	var config ZenHubConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", nil
	}

	return config.DefaultWorkspace, nil
}
