package core

import (
	"context"
	"fmt"
	"log/slog"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
)

// JiraClientOptions configures the Jira client creation
type JiraClientOptions struct {
	Logger *slog.Logger
}

// CreateJiraClient creates a new Jira API client with the provided credentials
func CreateJiraClient(creds *JiraCredentials, opts JiraClientOptions) (*jira.Client, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if creds == nil {
		return nil, fmt.Errorf("credentials are required")
	}

	if creds.Token == "" {
		return nil, fmt.Errorf("API token is required")
	}

	if creds.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	if creds.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	logger.Debug("creating Jira client",
		slog.String("url", creds.BaseURL),
		slog.String("email", creds.Email),
		slog.String("source", string(creds.Source)),
	)

	// Create basic auth transport
	tp := jira.BasicAuthTransport{
		Username: creds.Email,
		APIToken: creds.Token,
	}

	// Create the Jira client
	client, err := jira.NewClient(creds.BaseURL, tp.Client())
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return client, nil
}

// ValidateJiraConnection validates the Jira connection by making a simple API call
func ValidateJiraConnection(client *jira.Client) error {
	// Try to get current user to validate connection
	_, _, err := client.User.GetCurrentUser(context.Background())
	if err != nil {
		return fmt.Errorf("failed to validate Jira connection: %w", err)
	}

	return nil
}
