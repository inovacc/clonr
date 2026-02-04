package service

import (
	"context"
	"fmt"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
)

// SlackService provides Slack operations with profile-based token retrieval.
type SlackService struct {
	profileService *ProfileService
}

// NewSlackService creates a new SlackService.
func NewSlackService(profileService *ProfileService) *SlackService {
	return &SlackService{profileService: profileService}
}

// GetSlackClient creates a Slack client from the active profile's token.
func (ss *SlackService) GetSlackClient() (*slack.Client, error) {
	activeProfile, err := ss.profileService.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile: %w", err)
	}

	channel, err := ss.profileService.GetNotifyChannelByType(activeProfile.Name, model.ChannelSlack)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack channel: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("slack not connected")
	}

	// Decrypt the channel config to get the plain text token
	decryptedConfig, err := ss.profileService.DecryptChannelConfig(activeProfile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Slack config: %w", err)
	}

	token := decryptedConfig["bot_token"]
	if token == "" {
		return nil, fmt.Errorf("no Slack token found")
	}

	return slack.NewClient(token, slack.ClientOptions{}), nil
}

// IsConnected checks if Slack is connected for the active profile.
func (ss *SlackService) IsConnected() (bool, string, error) {
	activeProfile, err := ss.profileService.GetActiveProfile()
	if err != nil {
		return false, "", err
	}

	channel, _ := ss.profileService.GetNotifyChannelByType(activeProfile.Name, model.ChannelSlack)

	return channel != nil, activeProfile.Name, nil
}

// GetStatus returns Slack connection status.
func (ss *SlackService) GetStatus() (connected bool, profileName string, channelName string, err error) {
	activeProfile, err := ss.profileService.GetActiveProfile()
	if err != nil {
		return false, "", "", err
	}

	channel, _ := ss.profileService.GetNotifyChannelByType(activeProfile.Name, model.ChannelSlack)
	if channel == nil {
		return false, activeProfile.Name, "", nil
	}

	return true, activeProfile.Name, channel.Name, nil
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
