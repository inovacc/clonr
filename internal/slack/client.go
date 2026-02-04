package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	slackAPIBaseURL = "https://slack.com/api"
)

// Client is a Slack API client for reading data.
type Client struct {
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// ClientOptions configures a Slack client.
type ClientOptions struct {
	Logger *slog.Logger
}

// NewClient creates a new Slack API client.
func NewClient(token string, opts ClientOptions) *Client {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Channel represents a Slack channel.
type Channel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	IsChannel      bool     `json:"is_channel"`
	IsGroup        bool     `json:"is_group"`
	IsIM           bool     `json:"is_im"`
	IsMpIM         bool     `json:"is_mpim"`
	IsPrivate      bool     `json:"is_private"`
	IsArchived     bool     `json:"is_archived"`
	IsMember       bool     `json:"is_member"`
	NumMembers     int      `json:"num_members"`
	Topic          Topic    `json:"topic"`
	Purpose        Purpose  `json:"purpose"`
	Created        int64    `json:"created"`
	Creator        string   `json:"creator"`
	ContextTeamID  string   `json:"context_team_id"`
	Updated        int64    `json:"updated"`
	PreviousNames  []string `json:"previous_names"`
	NameNormalized string   `json:"name_normalized"`
}

// Topic represents a channel topic.
type Topic struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// Purpose represents a channel purpose.
type Purpose struct {
	Value   string `json:"value"`
	Creator string `json:"creator"`
	LastSet int64  `json:"last_set"`
}

// Message represents a Slack message.
type Message struct {
	Type        string       `json:"type"`
	User        string       `json:"user"`
	Text        string       `json:"text"`
	Timestamp   string       `json:"ts"`
	ThreadTS    string       `json:"thread_ts,omitempty"`
	ReplyCount  int          `json:"reply_count,omitempty"`
	Reactions   []Reaction   `json:"reactions,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Files       []File       `json:"files,omitempty"`
	BotID       string       `json:"bot_id,omitempty"`
	BotProfile  *BotProfile  `json:"bot_profile,omitempty"`
	Edited      *Edited      `json:"edited,omitempty"`
	// Computed fields
	ChannelID   string `json:"-"`
	ChannelName string `json:"-"`
	UserName    string `json:"-"`
}

// Reaction represents a message reaction.
type Reaction struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
	Count int      `json:"count"`
}

// Attachment represents a message attachment.
type Attachment struct {
	ID             int    `json:"id"`
	Fallback       string `json:"fallback"`
	Color          string `json:"color"`
	Pretext        string `json:"pretext"`
	AuthorName     string `json:"author_name"`
	AuthorLink     string `json:"author_link"`
	AuthorIcon     string `json:"author_icon"`
	Title          string `json:"title"`
	TitleLink      string `json:"title_link"`
	Text           string `json:"text"`
	ImageURL       string `json:"image_url"`
	ThumbURL       string `json:"thumb_url"`
	Footer         string `json:"footer"`
	FooterIcon     string `json:"footer_icon"`
	Timestamp      int64  `json:"ts"`
	FromURL        string `json:"from_url"`
	ServiceName    string `json:"service_name"`
	ServiceIcon    string `json:"service_icon"`
	OriginalURL    string `json:"original_url"`
	MessageBlocks  []any  `json:"message_blocks,omitempty"`
	MrkdwnIn       []any  `json:"mrkdwn_in,omitempty"`
	IsMsgUnfurl    bool   `json:"is_msg_unfurl,omitempty"`
	IsAppUnfurl    bool   `json:"is_app_unfurl,omitempty"`
	AppUnfurlURL   string `json:"app_unfurl_url,omitempty"`
	ChannelTeam    string `json:"channel_team,omitempty"`
	ChannelID      string `json:"channel_id,omitempty"`
	ChannelName    string `json:"channel_name,omitempty"`
	IsThreadRootfn bool   `json:"is_thread_rootfn,omitempty"`
}

// File represents a Slack file.
type File struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Title              string `json:"title"`
	Mimetype           string `json:"mimetype"`
	Filetype           string `json:"filetype"`
	PrettyType         string `json:"pretty_type"`
	User               string `json:"user"`
	Size               int64  `json:"size"`
	Mode               string `json:"mode"`
	IsExternal         bool   `json:"is_external"`
	ExternalType       string `json:"external_type"`
	IsPublic           bool   `json:"is_public"`
	PublicURLShared    bool   `json:"public_url_shared"`
	URLPrivate         string `json:"url_private"`
	URLPrivateDownload string `json:"url_private_download"`
	Permalink          string `json:"permalink"`
	PermalinkPublic    string `json:"permalink_public"`
	Timestamp          int64  `json:"timestamp"`
	Created            int64  `json:"created"`
}

// BotProfile represents a bot's profile.
type BotProfile struct {
	ID      string `json:"id"`
	AppID   string `json:"app_id"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
	Updated int64  `json:"updated"`
	TeamID  string `json:"team_id"`
}

// Edited represents edit information.
type Edited struct {
	User string `json:"user"`
	TS   string `json:"ts"`
}

// User represents a Slack user.
type User struct {
	ID                string      `json:"id"`
	TeamID            string      `json:"team_id"`
	Name              string      `json:"name"`
	Deleted           bool        `json:"deleted"`
	Color             string      `json:"color"`
	RealName          string      `json:"real_name"`
	TZ                string      `json:"tz"`
	TZLabel           string      `json:"tz_label"`
	TZOffset          int         `json:"tz_offset"`
	Profile           UserProfile `json:"profile"`
	IsAdmin           bool        `json:"is_admin"`
	IsOwner           bool        `json:"is_owner"`
	IsPrimaryOwner    bool        `json:"is_primary_owner"`
	IsRestricted      bool        `json:"is_restricted"`
	IsUltraRestricted bool        `json:"is_ultra_restricted"`
	IsBot             bool        `json:"is_bot"`
	IsAppUser         bool        `json:"is_app_user"`
	Updated           int64       `json:"updated"`
}

// UserProfile represents a user's profile.
type UserProfile struct {
	AvatarHash        string `json:"avatar_hash"`
	StatusText        string `json:"status_text"`
	StatusEmoji       string `json:"status_emoji"`
	RealName          string `json:"real_name"`
	DisplayName       string `json:"display_name"`
	RealNameNorm      string `json:"real_name_normalized"`
	DisplayNameNorm   string `json:"display_name_normalized"`
	Email             string `json:"email"`
	Image24           string `json:"image_24"`
	Image32           string `json:"image_32"`
	Image48           string `json:"image_48"`
	Image72           string `json:"image_72"`
	Image192          string `json:"image_192"`
	Image512          string `json:"image_512"`
	Image1024         string `json:"image_1024"`
	ImageOriginal     string `json:"image_original"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Title             string `json:"title"`
	Phone             string `json:"phone"`
	Skype             string `json:"skype"`
	StatusExpiration  int64  `json:"status_expiration"`
	Team              string `json:"team"`
	StatusTextCanon   string `json:"status_text_canonical"`
	StatusEmojiDisp   string `json:"status_emoji_display_info,omitempty"`
	IsCustomImage     bool   `json:"is_custom_image,omitempty"`
	PronounCode       string `json:"pronouns,omitempty"`
	HuddleState       string `json:"huddle_state,omitempty"`
	HuddleStateExpTS  int64  `json:"huddle_state_expiration_ts,omitempty"`
}

// SearchResult represents search results.
type SearchResult struct {
	Query    string          `json:"query"`
	Messages SearchMessages  `json:"messages"`
	Total    int             `json:"total"`
	Paging   SearchPaging    `json:"paging"`
	Matches  []SearchMessage `json:"-"`
}

// SearchMessages wraps message matches.
type SearchMessages struct {
	Total      int             `json:"total"`
	Pagination SearchPaging    `json:"pagination"`
	Paging     SearchPaging    `json:"paging"`
	Matches    []SearchMessage `json:"matches"`
}

// SearchMessage represents a search result message.
type SearchMessage struct {
	Type      string  `json:"type"`
	User      string  `json:"user"`
	Username  string  `json:"username"`
	Text      string  `json:"text"`
	Timestamp string  `json:"ts"`
	Channel   Channel `json:"channel"`
	Permalink string  `json:"permalink"`
	IID       string  `json:"iid"`
	Team      string  `json:"team"`
	Score     float64 `json:"score,omitempty"`
}

// SearchPaging represents pagination info.
type SearchPaging struct {
	Count      int `json:"count"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Pages      int `json:"pages"`
	First      int `json:"first"`
	Last       int `json:"last"`
	PerPage    int `json:"per_page"`
	TotalCount int `json:"total_count"`
}

// ResponseMetadata contains cursor information.
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

// slackResponse is the common Slack API response structure.
type slackResponse struct {
	OK               bool              `json:"ok"`
	Error            string            `json:"error,omitempty"`
	ResponseMetadata *ResponseMetadata `json:"response_metadata,omitempty"`
}

// ListChannelsOptions configures ListChannels.
type ListChannelsOptions struct {
	ExcludeArchived bool
	Types           string // comma-separated: public_channel,private_channel,mpim,im
	Limit           int
	Cursor          string
}

// ListChannelsResult contains the channels list result.
type ListChannelsResult struct {
	Channels   []Channel
	NextCursor string
}

// ListChannels lists conversations/channels.
func (c *Client) ListChannels(ctx context.Context, opts ListChannelsOptions) (*ListChannelsResult, error) {
	params := url.Values{}
	params.Set("exclude_archived", strconv.FormatBool(opts.ExcludeArchived))

	if opts.Types != "" {
		params.Set("types", opts.Types)
	} else {
		params.Set("types", "public_channel,private_channel")
	}

	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	} else {
		params.Set("limit", "100")
	}

	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	var resp struct {
		slackResponse

		Channels []Channel `json:"channels"`
	}

	if err := c.get(ctx, "conversations.list", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	result := &ListChannelsResult{
		Channels: resp.Channels,
	}

	if resp.ResponseMetadata != nil {
		result.NextCursor = resp.ResponseMetadata.NextCursor
	}

	return result, nil
}

// GetChannelHistoryOptions configures GetChannelHistory.
type GetChannelHistoryOptions struct {
	Channel string
	Limit   int
	Oldest  string // Unix timestamp
	Latest  string // Unix timestamp
	Cursor  string
}

// GetChannelHistoryResult contains the history result.
type GetChannelHistoryResult struct {
	Messages        []Message
	HasMore         bool
	NextCursor      string
	PinCount        int
	ResponseMetadat *ResponseMetadata
}

// GetChannelHistory gets messages from a channel.
func (c *Client) GetChannelHistory(ctx context.Context, opts GetChannelHistoryOptions) (*GetChannelHistoryResult, error) {
	params := url.Values{}
	params.Set("channel", opts.Channel)

	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	} else {
		params.Set("limit", "100")
	}

	if opts.Oldest != "" {
		params.Set("oldest", opts.Oldest)
	}

	if opts.Latest != "" {
		params.Set("latest", opts.Latest)
	}

	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	var resp struct {
		slackResponse

		Messages []Message `json:"messages"`
		HasMore  bool      `json:"has_more"`
		PinCount int       `json:"pin_count"`
	}

	if err := c.get(ctx, "conversations.history", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	result := &GetChannelHistoryResult{
		Messages: resp.Messages,
		HasMore:  resp.HasMore,
		PinCount: resp.PinCount,
	}

	if resp.ResponseMetadata != nil {
		result.NextCursor = resp.ResponseMetadata.NextCursor
	}

	return result, nil
}

// SearchMessagesOptions configures SearchMessages.
type SearchMessagesOptions struct {
	Query string
	Sort  string // score or timestamp
	Dir   string // asc or desc
	Count int
	Page  int
}

// SearchMessages searches for messages.
func (c *Client) SearchMessages(ctx context.Context, opts SearchMessagesOptions) (*SearchResult, error) {
	params := url.Values{}
	params.Set("query", opts.Query)

	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}

	if opts.Dir != "" {
		params.Set("sort_dir", opts.Dir)
	}

	if opts.Count > 0 {
		params.Set("count", strconv.Itoa(opts.Count))
	} else {
		params.Set("count", "20")
	}

	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}

	var resp struct {
		slackResponse

		Query    string         `json:"query"`
		Messages SearchMessages `json:"messages"`
	}

	if err := c.get(ctx, "search.messages", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	return &SearchResult{
		Query:    resp.Query,
		Messages: resp.Messages,
		Total:    resp.Messages.Total,
		Paging:   resp.Messages.Paging,
		Matches:  resp.Messages.Matches,
	}, nil
}

// GetUserOptions configures GetUser.
type GetUserOptions struct {
	UserID string
}

// GetUser gets information about a user.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	params := url.Values{}
	params.Set("user", userID)

	var resp struct {
		slackResponse

		User User `json:"user"`
	}

	if err := c.get(ctx, "users.info", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	return &resp.User, nil
}

// ListUsersOptions configures ListUsers.
type ListUsersOptions struct {
	Cursor string
	Limit  int
}

// ListUsersResult contains the users list result.
type ListUsersResult struct {
	Users      []User
	NextCursor string
}

// ListUsers lists workspace users.
func (c *Client) ListUsers(ctx context.Context, opts ListUsersOptions) (*ListUsersResult, error) {
	params := url.Values{}

	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	} else {
		params.Set("limit", "100")
	}

	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	var resp struct {
		slackResponse

		Members []User `json:"members"`
	}

	if err := c.get(ctx, "users.list", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	result := &ListUsersResult{
		Users: resp.Members,
	}

	if resp.ResponseMetadata != nil {
		result.NextCursor = resp.ResponseMetadata.NextCursor
	}

	return result, nil
}

// GetThreadRepliesOptions configures GetThreadReplies.
type GetThreadRepliesOptions struct {
	Channel  string
	ThreadTS string
	Cursor   string
	Limit    int
}

// GetThreadRepliesResult contains thread replies.
type GetThreadRepliesResult struct {
	Messages   []Message
	HasMore    bool
	NextCursor string
}

// GetThreadReplies gets replies to a thread.
func (c *Client) GetThreadReplies(ctx context.Context, opts GetThreadRepliesOptions) (*GetThreadRepliesResult, error) {
	params := url.Values{}
	params.Set("channel", opts.Channel)
	params.Set("ts", opts.ThreadTS)

	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	} else {
		params.Set("limit", "100")
	}

	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	var resp struct {
		slackResponse

		Messages []Message `json:"messages"`
		HasMore  bool      `json:"has_more"`
	}

	if err := c.get(ctx, "conversations.replies", params, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	result := &GetThreadRepliesResult{
		Messages: resp.Messages,
		HasMore:  resp.HasMore,
	}

	if resp.ResponseMetadata != nil {
		result.NextCursor = resp.ResponseMetadata.NextCursor
	}

	return result, nil
}

// AuthTest tests the authentication token.
func (c *Client) AuthTest(ctx context.Context) (*AuthTestResult, error) {
	var resp struct {
		slackResponse

		URL    string `json:"url"`
		Team   string `json:"team"`
		User   string `json:"user"`
		TeamID string `json:"team_id"`
		UserID string `json:"user_id"`
		BotID  string `json:"bot_id"`
	}

	if err := c.get(ctx, "auth.test", nil, &resp); err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("slack API error: %s", resp.Error)
	}

	return &AuthTestResult{
		URL:    resp.URL,
		Team:   resp.Team,
		User:   resp.User,
		TeamID: resp.TeamID,
		UserID: resp.UserID,
		BotID:  resp.BotID,
	}, nil
}

// AuthTestResult contains auth test information.
type AuthTestResult struct {
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	BotID  string `json:"bot_id"`
}

// get makes a GET request to the Slack API.
func (c *Client) get(ctx context.Context, method string, params url.Values, result any) error {
	u := fmt.Sprintf("%s/%s", slackAPIBaseURL, method)

	if params != nil {
		u = u + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("slack API request", "method", method, "params", params)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// ParseTimestamp parses a Slack timestamp to time.Time.
func ParseTimestamp(ts string) (time.Time, error) {
	// Slack timestamps are Unix timestamps with a decimal (e.g., "1234567890.123456")
	var sec, usec int64
	if _, err := fmt.Sscanf(ts, "%d.%d", &sec, &usec); err != nil {
		// Try parsing as just seconds
		if s, parseErr := strconv.ParseInt(ts, 10, 64); parseErr == nil {
			return time.Unix(s, 0), nil
		}

		return time.Time{}, fmt.Errorf("invalid timestamp: %s", ts)
	}

	return time.Unix(sec, usec*1000), nil
}

// FormatTimestamp formats a time.Time to Slack timestamp.
func FormatTimestamp(t time.Time) string {
	return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
}
