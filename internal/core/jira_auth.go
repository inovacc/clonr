package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors for Jira configuration
var (
	// ErrJiraConfigNotFound indicates no Jira configuration file was found
	ErrJiraConfigNotFound = errors.New("jira config not found")
	// ErrJiraInstanceNotFound indicates the specified Jira instance was not found in config
	ErrJiraInstanceNotFound = errors.New("jira instance not found in config")
)

// JiraTokenSource indicates where the Jira credentials were found
type JiraTokenSource string

const (
	JiraTokenSourceFlag   JiraTokenSource = "flag"
	JiraTokenSourceEnv    JiraTokenSource = "JIRA_API_TOKEN"
	JiraTokenSourceEnvAlt JiraTokenSource = "ATLASSIAN_TOKEN"
	JiraTokenSourceConfig JiraTokenSource = "config"
	JiraTokenSourceNone   JiraTokenSource = "none"
)

// JiraCredentials holds the resolved Jira authentication credentials
type JiraCredentials struct {
	Token   string
	Email   string
	BaseURL string
	Source  JiraTokenSource
}

// JiraConfig represents the Jira configuration file structure
type JiraConfig struct {
	DefaultInstance string                  `json:"default_instance,omitempty"`
	Instances       map[string]JiraInstance `json:"instances,omitempty"`
}

// JiraInstance represents a single Jira instance configuration
type JiraInstance struct {
	URL            string `json:"url"`
	Email          string `json:"email"`
	Token          string `json:"token"`
	DefaultProject string `json:"default_project,omitempty"`
}

// ResolveJiraCredentials attempts to find Jira credentials from multiple sources.
// Priority order:
//  1. flagToken, flagEmail, flagURL (explicit flags)
//  2. JIRA_API_TOKEN, JIRA_EMAIL, JIRA_URL environment variables
//  3. ATLASSIAN_TOKEN environment variable (alternative token)
//  4. ~/.config/clonr/jira.json config file
func ResolveJiraCredentials(flagToken, flagEmail, flagURL string) (*JiraCredentials, error) {
	creds := &JiraCredentials{}

	// 1. Check flags first (highest priority)
	if flagToken != "" && flagEmail != "" && flagURL != "" {
		creds.Token = flagToken
		creds.Email = flagEmail
		creds.BaseURL = normalizeJiraURL(flagURL)
		creds.Source = JiraTokenSourceFlag

		return creds, nil
	}

	// Partial flag values - fill in from environment
	if flagToken != "" {
		creds.Token = flagToken
		creds.Source = JiraTokenSourceFlag
	}

	if flagEmail != "" {
		creds.Email = flagEmail
	}

	if flagURL != "" {
		creds.BaseURL = normalizeJiraURL(flagURL)
	}

	// 2. Check environment variables
	if creds.Token == "" {
		if token := os.Getenv("JIRA_API_TOKEN"); token != "" {
			creds.Token = token
			creds.Source = JiraTokenSourceEnv
		} else if token := os.Getenv("ATLASSIAN_TOKEN"); token != "" {
			creds.Token = token
			creds.Source = JiraTokenSourceEnvAlt
		}
	}

	if creds.Email == "" {
		creds.Email = os.Getenv("JIRA_EMAIL")
	}

	if creds.BaseURL == "" {
		creds.BaseURL = normalizeJiraURL(os.Getenv("JIRA_URL"))
	}

	// 3. Check config file for missing values
	if creds.Token == "" || creds.Email == "" || creds.BaseURL == "" {
		configCreds, err := loadJiraConfigCredentials("")
		if err == nil && configCreds != nil {
			if creds.Token == "" && configCreds.Token != "" {
				creds.Token = configCreds.Token
				creds.Source = JiraTokenSourceConfig
			}

			if creds.Email == "" {
				creds.Email = configCreds.Email
			}

			if creds.BaseURL == "" {
				creds.BaseURL = configCreds.BaseURL
			}
		}
	}

	// 4. Validate we have all required credentials
	if creds.Token == "" {
		return nil, fmt.Errorf(`jira API token required

Provide a token via one of:
  * JIRA_API_TOKEN env var     (recommended)
  * ATLASSIAN_TOKEN env var
  * --token flag
  * ~/.config/clonr/jira.json config file

Create an API token at: https://id.atlassian.com/manage-profile/security/api-tokens`)
	}

	if creds.Email == "" {
		return nil, fmt.Errorf(`jira email required

Provide your Atlassian account email via one of:
  * JIRA_EMAIL env var
  * --email flag
  * ~/.config/clonr/jira.json config file`)
	}

	if creds.BaseURL == "" {
		return nil, fmt.Errorf(`jira instance URL required

Provide your Jira instance URL via one of:
  * JIRA_URL env var
  * --url flag
  * ~/.config/clonr/jira.json config file

Example: https://yourcompany.atlassian.net`)
	}

	return creds, nil
}

// ResolveJiraToken resolves only the Jira API token (for simpler use cases)
func ResolveJiraToken(flagToken string) (token string, source JiraTokenSource, err error) {
	// 1. Flag has highest priority
	if flagToken != "" {
		return flagToken, JiraTokenSourceFlag, nil
	}

	// 2. Check JIRA_API_TOKEN env var
	if token = os.Getenv("JIRA_API_TOKEN"); token != "" {
		return token, JiraTokenSourceEnv, nil
	}

	// 3. Check ATLASSIAN_TOKEN env var
	if token = os.Getenv("ATLASSIAN_TOKEN"); token != "" {
		return token, JiraTokenSourceEnvAlt, nil
	}

	// 4. Try config file
	creds, err := loadJiraConfigCredentials("")
	if err == nil && creds != nil && creds.Token != "" {
		return creds.Token, JiraTokenSourceConfig, nil
	}

	// 5. No token found
	return "", JiraTokenSourceNone, fmt.Errorf(`jira API token required

Provide a token via one of:
  * JIRA_API_TOKEN env var     (recommended)
  * ATLASSIAN_TOKEN env var
  * --token flag
  * ~/.config/clonr/jira.json config file

Create an API token at: https://id.atlassian.com/manage-profile/security/api-tokens`)
}

// loadJiraConfigCredentials loads credentials from the Jira config file
func loadJiraConfigCredentials(instanceName string) (*JiraCredentials, error) {
	configPath, err := getJiraConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrJiraConfigNotFound
		}

		return nil, fmt.Errorf("failed to read Jira config: %w", err)
	}

	var config JiraConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Jira config: %w", err)
	}

	// Determine which instance to use
	if instanceName == "" {
		instanceName = config.DefaultInstance
	}

	if instanceName == "" {
		// Use first instance if no default
		for name := range config.Instances {
			instanceName = name
			break
		}
	}

	if instanceName == "" {
		return nil, ErrJiraInstanceNotFound
	}

	instance, ok := config.Instances[instanceName]
	if !ok {
		return nil, ErrJiraInstanceNotFound
	}

	creds := &JiraCredentials{
		Email:   instance.Email,
		BaseURL: normalizeJiraURL(instance.URL),
		Source:  JiraTokenSourceConfig,
	}

	// Handle token reference to env var
	if envVar, found := strings.CutPrefix(instance.Token, "env:"); found {
		creds.Token = os.Getenv(envVar)
	} else {
		creds.Token = instance.Token
	}

	return creds, nil
}

// getJiraConfigPath returns the path to the Jira config file
func getJiraConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine config directory: %w", err)
		}

		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "clonr", "jira.json"), nil
}

// normalizeJiraURL ensures the URL is properly formatted
func normalizeJiraURL(url string) string {
	if url == "" {
		return ""
	}

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Add https:// if no scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	return url
}

// GetJiraDefaultProject returns the default project for a Jira instance
func GetJiraDefaultProject(instanceName string) (string, error) {
	configPath, err := getJiraConfigPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config JiraConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	if instanceName == "" {
		instanceName = config.DefaultInstance
	}

	if instanceName == "" {
		for name := range config.Instances {
			instanceName = name
			break
		}
	}

	if instance, ok := config.Instances[instanceName]; ok {
		return instance.DefaultProject, nil
	}

	return "", nil
}
