package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/inovacc/clonr/internal/slack"
)

// SlackService provides Slack operations with account-based token retrieval.
type SlackService struct {
	accountService *SlackAccountService
}

// NewSlackService creates a new SlackService with Slack account support.
func NewSlackService(accountService *SlackAccountService) *SlackService {
	return &SlackService{accountService: accountService}
}

// GetSlackClient creates a Slack client from the active Slack account's token.
func (ss *SlackService) GetSlackClient() (*slack.Client, error) {
	account, err := ss.accountService.GetActiveAccount()
	if err != nil {
		if errors.Is(err, ErrNoActiveSlackAccount) {
			return nil, fmt.Errorf("no active Slack account configured")
		}

		return nil, fmt.Errorf("failed to get active Slack account: %w", err)
	}

	token, err := ss.accountService.GetDecryptedToken(account.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack token: %w", err)
	}

	return slack.NewClient(token, slack.ClientOptions{}), nil
}

// IsConnected checks if Slack is connected (has an active account).
func (ss *SlackService) IsConnected() (bool, string, error) {
	account, err := ss.accountService.GetActiveAccount()
	if err != nil {
		if errors.Is(err, ErrNoActiveSlackAccount) {
			return false, "", nil
		}

		return false, "", err
	}

	return true, account.Name, nil
}

// GetStatus returns Slack connection status.
func (ss *SlackService) GetStatus() (connected bool, accountName string, workspaceName string, err error) {
	account, err := ss.accountService.GetActiveAccount()
	if err != nil {
		if errors.Is(err, ErrNoActiveSlackAccount) {
			return false, "", "", nil
		}

		return false, "", "", err
	}

	return true, account.Name, account.WorkspaceName, nil
}

// ListChannels returns list of Slack channels.
func (ss *SlackService) ListChannels(ctx context.Context) ([]slack.Channel, error) {
	client, err := ss.GetSlackClient()
	if err != nil {
		return nil, err
	}

	result, err := client.ListChannels(ctx, slack.ListChannelsOptions{
		ExcludeArchived: true,
		Types:           "public_channel,private_channel",
		Limit:           100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	return result.Channels, nil
}

// GetChannelHistory returns messages from a channel.
func (ss *SlackService) GetChannelHistory(ctx context.Context, channelID string, limit int, cursor string) (*slack.GetChannelHistoryResult, error) {
	client, err := ss.GetSlackClient()
	if err != nil {
		return nil, err
	}

	result, err := client.GetChannelHistory(ctx, slack.GetChannelHistoryOptions{
		Channel: channelID,
		Limit:   limit,
		Cursor:  cursor,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return result, nil
}

// GetUser returns user info.
func (ss *SlackService) GetUser(ctx context.Context, userID string) (*slack.User, error) {
	client, err := ss.GetSlackClient()
	if err != nil {
		return nil, err
	}

	return client.GetUser(ctx, userID)
}

// SearchMessages searches Slack messages.
func (ss *SlackService) SearchMessages(ctx context.Context, query string, count int) (*slack.SearchResult, error) {
	client, err := ss.GetSlackClient()
	if err != nil {
		return nil, err
	}

	result, err := client.SearchMessages(ctx, slack.SearchMessagesOptions{
		Query: query,
		Sort:  "timestamp",
		Dir:   "desc",
		Count: count,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return result, nil
}
