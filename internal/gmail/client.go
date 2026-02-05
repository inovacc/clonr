package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	gmailAPIBaseURL = "https://gmail.googleapis.com/gmail/v1"
)

// Client is a Gmail API client.
type Client struct {
	accessToken  string
	refreshToken string
	httpClient   *http.Client
}

// ClientOptions configures a Gmail client.
type ClientOptions struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
}

// NewClient creates a new Gmail API client.
func NewClient(accessToken string, opts ClientOptions) *Client {
	return &Client{
		accessToken:  accessToken,
		refreshToken: opts.RefreshToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Message represents a Gmail message.
type Message struct {
	ID           string            `json:"id"`
	ThreadID     string            `json:"threadId"`
	LabelIDs     []string          `json:"labelIds"`
	Snippet      string            `json:"snippet"`
	InternalDate string            `json:"internalDate"`
	Payload      *MessagePayload   `json:"payload"`
	SizeEstimate int               `json:"sizeEstimate"`
	Headers      map[string]string `json:"-"` // Parsed headers
}

// MessagePayload represents the email payload.
type MessagePayload struct {
	PartID   string           `json:"partId"`
	MimeType string           `json:"mimeType"`
	Filename string           `json:"filename"`
	Headers  []Header         `json:"headers"`
	Body     *MessageBody     `json:"body"`
	Parts    []MessagePayload `json:"parts"`
}

// Header represents an email header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MessageBody represents the body of a message part.
type MessageBody struct {
	AttachmentID string `json:"attachmentId"`
	Size         int    `json:"size"`
	Data         string `json:"data"` // Base64 encoded
}

// Label represents a Gmail label.
type Label struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	MessagesTotal         int    `json:"messagesTotal"`
	MessagesUnread        int    `json:"messagesUnread"`
	ThreadsTotal          int    `json:"threadsTotal"`
	ThreadsUnread         int    `json:"threadsUnread"`
	LabelListVisibility   string `json:"labelListVisibility"`
	MessageListVisibility string `json:"messageListVisibility"`
}

// ListMessagesOptions configures message listing.
type ListMessagesOptions struct {
	MaxResults    int
	Query         string   // Gmail search query
	LabelIDs      []string // Filter by labels
	PageToken     string
	IncludeSpam   bool
}

// ListMessagesResponse contains the list messages response.
type ListMessagesResponse struct {
	Messages           []MessageRef `json:"messages"`
	NextPageToken      string       `json:"nextPageToken"`
	ResultSizeEstimate int          `json:"resultSizeEstimate"`
}

// MessageRef is a reference to a message (ID only).
type MessageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// Profile represents the user's Gmail profile.
type Profile struct {
	EmailAddress  string `json:"emailAddress"`
	MessagesTotal int    `json:"messagesTotal"`
	ThreadsTotal  int    `json:"threadsTotal"`
	HistoryID     string `json:"historyId"`
}

// GetProfile retrieves the user's Gmail profile.
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	var profile Profile
	if err := c.get(ctx, "users/me/profile", nil, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// ListLabels lists all labels in the mailbox.
func (c *Client) ListLabels(ctx context.Context) ([]Label, error) {
	var resp struct {
		Labels []Label `json:"labels"`
	}

	if err := c.get(ctx, "users/me/labels", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Labels, nil
}

// ListMessages lists messages in the mailbox.
func (c *Client) ListMessages(ctx context.Context, opts ListMessagesOptions) (*ListMessagesResponse, error) {
	params := url.Values{}

	if opts.MaxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", opts.MaxResults))
	}

	if opts.Query != "" {
		params.Set("q", opts.Query)
	}

	if len(opts.LabelIDs) > 0 {
		for _, id := range opts.LabelIDs {
			params.Add("labelIds", id)
		}
	}

	if opts.PageToken != "" {
		params.Set("pageToken", opts.PageToken)
	}

	if opts.IncludeSpam {
		params.Set("includeSpamTrash", "true")
	}

	var resp ListMessagesResponse
	if err := c.get(ctx, "users/me/messages", params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetMessage retrieves a full message by ID.
func (c *Client) GetMessage(ctx context.Context, id string, format string) (*Message, error) {
	if format == "" {
		format = "full"
	}

	params := url.Values{}
	params.Set("format", format)

	var msg Message
	if err := c.get(ctx, fmt.Sprintf("users/me/messages/%s", id), params, &msg); err != nil {
		return nil, err
	}

	// Parse headers into map for convenience
	msg.Headers = make(map[string]string)
	if msg.Payload != nil {
		for _, h := range msg.Payload.Headers {
			msg.Headers[strings.ToLower(h.Name)] = h.Value
		}
	}

	return &msg, nil
}

// SearchMessages searches for messages using Gmail query syntax.
func (c *Client) SearchMessages(ctx context.Context, query string, maxResults int) (*ListMessagesResponse, error) {
	return c.ListMessages(ctx, ListMessagesOptions{
		Query:      query,
		MaxResults: maxResults,
	})
}

// GetMessageBody extracts the plain text body from a message.
func (c *Client) GetMessageBody(msg *Message) string {
	if msg.Payload == nil {
		return ""
	}

	return extractBody(msg.Payload, "text/plain")
}

// GetMessageHTMLBody extracts the HTML body from a message.
func (c *Client) GetMessageHTMLBody(msg *Message) string {
	if msg.Payload == nil {
		return ""
	}

	return extractBody(msg.Payload, "text/html")
}

// Attachment represents an email attachment.
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

// AttachmentData contains the actual attachment content.
type AttachmentData struct {
	Size int    `json:"size"`
	Data string `json:"data"` // Base64 URL-encoded
}

// GetMessageAttachments extracts attachment information from a message.
func (c *Client) GetMessageAttachments(msg *Message) []Attachment {
	if msg.Payload == nil {
		return nil
	}

	return extractAttachments(msg.Payload)
}

// extractAttachments recursively extracts attachments from message parts.
func extractAttachments(payload *MessagePayload) []Attachment {
	var attachments []Attachment

	// Check if this part is an attachment
	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentID != "" {
		attachments = append(attachments, Attachment{
			ID:       payload.Body.AttachmentID,
			Filename: payload.Filename,
			MimeType: payload.MimeType,
			Size:     payload.Body.Size,
		})
	}

	// Recurse into parts
	for _, part := range payload.Parts {
		attachments = append(attachments, extractAttachments(&part)...)
	}

	return attachments
}

// GetAttachment downloads an attachment by its ID.
func (c *Client) GetAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error) {
	var data AttachmentData

	endpoint := fmt.Sprintf("users/me/messages/%s/attachments/%s", messageID, attachmentID)
	if err := c.get(ctx, endpoint, nil, &data); err != nil {
		return nil, err
	}

	// Decode base64 URL-encoded data
	decoded, err := base64.URLEncoding.DecodeString(data.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode attachment: %w", err)
	}

	return decoded, nil
}

// HasAttachments checks if a message has attachments.
func (c *Client) HasAttachments(msg *Message) bool {
	return len(c.GetMessageAttachments(msg)) > 0
}

// extractBody recursively extracts body content of a specific MIME type.
func extractBody(payload *MessagePayload, mimeType string) string {
	if payload.MimeType == mimeType && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err != nil {
			return ""
		}

		return string(decoded)
	}

	for _, part := range payload.Parts {
		if body := extractBody(&part, mimeType); body != "" {
			return body
		}
	}

	return ""
}

// get performs a GET request to the Gmail API.
func (c *Client) get(ctx context.Context, endpoint string, params url.Values, result any) error {
	reqURL := fmt.Sprintf("%s/%s", gmailAPIBaseURL, endpoint)
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
