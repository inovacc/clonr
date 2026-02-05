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

// OutlookClient is a Microsoft Outlook/Mail API client.
type OutlookClient struct {
	accessToken string
	httpClient  *http.Client
}

// OutlookClientOptions configures an Outlook client.
type OutlookClientOptions struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	TenantID     string
}

// NewOutlookClient creates a new Outlook API client.
func NewOutlookClient(accessToken string, opts OutlookClientOptions) *OutlookClient {
	return &OutlookClient{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MailMessage represents an Outlook email message.
type MailMessage struct {
	ID               string        `json:"id"`
	Subject          string        `json:"subject"`
	BodyPreview      string        `json:"bodyPreview"`
	Body             *MailBody     `json:"body"`
	From             *EmailAddress `json:"from"`
	ToRecipients     []Recipient   `json:"toRecipients"`
	CcRecipients     []Recipient   `json:"ccRecipients"`
	ReceivedDateTime time.Time     `json:"receivedDateTime"`
	SentDateTime     time.Time     `json:"sentDateTime"`
	HasAttachments   bool          `json:"hasAttachments"`
	IsRead           bool          `json:"isRead"`
	IsDraft          bool          `json:"isDraft"`
	Importance       string        `json:"importance"`
	WebLink          string        `json:"webLink"`
	ParentFolderID   string        `json:"parentFolderId"`
	ConversationID   string        `json:"conversationId"`
	Flag             *FollowUpFlag `json:"flag"`
}

// MailBody represents the body of an email.
type MailBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// EmailAddress represents an email address.
type EmailAddress struct {
	EmailAddress *EmailAddressDetail `json:"emailAddress"`
}

// EmailAddressDetail contains email address details.
type EmailAddressDetail struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Recipient represents an email recipient.
type Recipient struct {
	EmailAddress *EmailAddressDetail `json:"emailAddress"`
}

// FollowUpFlag represents a follow-up flag on a message.
type FollowUpFlag struct {
	FlagStatus string `json:"flagStatus"`
}

// MailFolder represents an Outlook mail folder.
type MailFolder struct {
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	ParentFolderID   string `json:"parentFolderId"`
	ChildFolderCount int    `json:"childFolderCount"`
	TotalItemCount   int    `json:"totalItemCount"`
	UnreadItemCount  int    `json:"unreadItemCount"`
}

// ListMailMessagesResponse contains the list messages response.
type ListMailMessagesResponse struct {
	Value    []MailMessage `json:"value"`
	NextLink string        `json:"@odata.nextLink"`
}

// ListFoldersResponse contains the list folders response.
type ListFoldersResponse struct {
	Value    []MailFolder `json:"value"`
	NextLink string       `json:"@odata.nextLink"`
}

// ListMailMessagesOptions configures message listing.
type ListMailMessagesOptions struct {
	Top      int
	Skip     int
	Filter   string
	Select   []string
	OrderBy  string
	FolderID string // defaults to "inbox"
}

// GetMailFolders lists all mail folders.
func (c *OutlookClient) GetMailFolders(ctx context.Context) ([]MailFolder, error) {
	var resp ListFoldersResponse
	if err := c.get(ctx, "/me/mailFolders", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetMailFolder retrieves a specific mail folder.
func (c *OutlookClient) GetMailFolder(ctx context.Context, folderID string) (*MailFolder, error) {
	var folder MailFolder
	if err := c.get(ctx, fmt.Sprintf("/me/mailFolders/%s", folderID), nil, &folder); err != nil {
		return nil, err
	}

	return &folder, nil
}

// ListMessages lists messages in a folder.
func (c *OutlookClient) ListMessages(ctx context.Context, opts ListMailMessagesOptions) ([]MailMessage, error) {
	folderID := opts.FolderID
	if folderID == "" {
		folderID = "inbox"
	}

	params := url.Values{}

	if opts.Top > 0 {
		params.Set("$top", fmt.Sprintf("%d", opts.Top))
	}

	if opts.Skip > 0 {
		params.Set("$skip", fmt.Sprintf("%d", opts.Skip))
	}

	if opts.Filter != "" {
		params.Set("$filter", opts.Filter)
	}

	if opts.OrderBy != "" {
		params.Set("$orderby", opts.OrderBy)
	} else {
		params.Set("$orderby", "receivedDateTime desc")
	}

	var resp ListMailMessagesResponse
	if err := c.get(ctx, fmt.Sprintf("/me/mailFolders/%s/messages", folderID), params, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetMessage retrieves a specific message by ID.
func (c *OutlookClient) GetMessage(ctx context.Context, messageID string) (*MailMessage, error) {
	var msg MailMessage
	if err := c.get(ctx, fmt.Sprintf("/me/messages/%s", messageID), nil, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// SearchMessages searches for messages using a query.
func (c *OutlookClient) SearchMessages(ctx context.Context, query string, top int) ([]MailMessage, error) {
	params := url.Values{}
	params.Set("$search", fmt.Sprintf("\"%s\"", query))

	if top > 0 {
		params.Set("$top", fmt.Sprintf("%d", top))
	}

	var resp ListMailMessagesResponse
	if err := c.get(ctx, "/me/messages", params, &resp); err != nil {
		return nil, err
	}

	return resp.Value, nil
}

// GetInboxStats returns inbox statistics.
func (c *OutlookClient) GetInboxStats(ctx context.Context) (*MailFolder, error) {
	return c.GetMailFolder(ctx, "inbox")
}

// get performs a GET request to the Microsoft Graph API.
func (c *OutlookClient) get(ctx context.Context, endpoint string, params url.Values, result any) error {
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
