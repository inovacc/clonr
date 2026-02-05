package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TeamsClient is a Microsoft Teams API client.
type TeamsClient struct {
	accessToken string
	httpClient  *http.Client
}

// TeamsClientOptions configures a Teams client.
type TeamsClientOptions struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	TenantID     string
}

// NewTeamsClient creates a new Teams API client.
func NewTeamsClient(accessToken string, opts TeamsClientOptions) *TeamsClient {
	return &TeamsClient{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Team represents a Microsoft Teams team.
type Team struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl"`
}

// Channel represents a Teams channel.
type Channel struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl"`
}

// ChatMessage represents a message in Teams.
type ChatMessage struct {
	ID               string       `json:"id"`
	MessageType      string       `json:"messageType"`
	CreatedDateTime  time.Time    `json:"createdDateTime"`
	LastModifiedTime time.Time    `json:"lastModifiedDateTime"`
	Subject          string       `json:"subject"`
	Body             *MessageBody `json:"body"`
	From             *MessageFrom `json:"from"`
	WebURL           string       `json:"webUrl"`
}

// MessageBody represents the body of a Teams message.
type MessageBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// MessageFrom represents the sender of a message.
type MessageFrom struct {
	User *MessageUser `json:"user"`
}

// MessageUser represents a user in a message.
type MessageUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// Chat represents a Teams chat.
type Chat struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	ChatType  string    `json:"chatType"`
	CreatedAt time.Time `json:"createdDateTime"`
	WebURL    string    `json:"webUrl"`
}

// ListTeamsResponse contains the list teams response.
type ListTeamsResponse struct {
	Value    []Team `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

// ListChannelsResponse contains the list channels response.
type ListChannelsResponse struct {
	Value    []Channel `json:"value"`
	NextLink string    `json:"@odata.nextLink"`
}

// ListMessagesResponse contains the list messages response.
type ListMessagesResponse struct {
	Value    []ChatMessage `json:"value"`
	NextLink string        `json:"@odata.nextLink"`
}

// ListChatsResponse contains the list chats response.
type ListChatsResponse struct {
	Value    []Chat `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

// GetMyTeams lists all teams the current user is a member of.
func (c *TeamsClient) GetMyTeams(ctx context.Context) ([]Team, error) {
	var resp ListTeamsResponse
	if err := c.get(ctx, "/me/joinedTeams", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetTeamChannels lists all channels in a team.
func (c *TeamsClient) GetTeamChannels(ctx context.Context, teamID string) ([]Channel, error) {
	var resp ListChannelsResponse
	if err := c.get(ctx, fmt.Sprintf("/teams/%s/channels", teamID), nil, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetChannelMessages lists messages in a channel.
func (c *TeamsClient) GetChannelMessages(ctx context.Context, teamID, channelID string, top int) ([]ChatMessage, error) {
	params := url.Values{}
	if top > 0 {
		params.Set("$top", fmt.Sprintf("%d", top))
	}

	var resp ListMessagesResponse
	if err := c.get(ctx, fmt.Sprintf("/teams/%s/channels/%s/messages", teamID, channelID), params, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetMyChats lists the current user's chats.
func (c *TeamsClient) GetMyChats(ctx context.Context, top int) ([]Chat, error) {
	params := url.Values{}
	if top > 0 {
		params.Set("$top", fmt.Sprintf("%d", top))
	}

	var resp ListChatsResponse
	if err := c.get(ctx, "/me/chats", params, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetChatMessages lists messages in a chat.
func (c *TeamsClient) GetChatMessages(ctx context.Context, chatID string, top int) ([]ChatMessage, error) {
	params := url.Values{}
	if top > 0 {
		params.Set("$top", fmt.Sprintf("%d", top))
	}

	var resp ListMessagesResponse
	if err := c.get(ctx, fmt.Sprintf("/me/chats/%s/messages", chatID), params, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// get performs a GET request to the Microsoft Graph API.
func (c *TeamsClient) get(ctx context.Context, endpoint string, params url.Values, result any) error {
	reqURL := graphAPIBaseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
