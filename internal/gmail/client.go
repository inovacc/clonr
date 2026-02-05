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
	MaxResults  int
	Query       string   // Gmail search query
	LabelIDs    []string // Filter by labels
	PageToken   string
	IncludeSpam bool
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

// CalendarEvent represents a parsed calendar event from an email.
type CalendarEvent struct {
	UID         string    `json:"uid"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Organizer   string    `json:"organizer,omitempty"`
	Attendees   []string  `json:"attendees,omitempty"`
	Status      string    `json:"status,omitempty"` // CONFIRMED, TENTATIVE, CANCELLED
	Method      string    `json:"method,omitempty"` // REQUEST, REPLY, CANCEL
	IsAllDay    bool      `json:"is_all_day"`
}

// DriveLink represents a Google Drive link found in an email.
type DriveLink struct {
	URL      string `json:"url"`
	FileID   string `json:"file_id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
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

// HasCalendarEvent checks if a message contains a calendar event.
func (c *Client) HasCalendarEvent(msg *Message) bool {
	if msg.Payload == nil {
		return false
	}

	return hasCalendarPart(msg.Payload)
}

// hasCalendarPart recursively checks for calendar MIME types.
func hasCalendarPart(payload *MessagePayload) bool {
	if payload.MimeType == "text/calendar" || payload.MimeType == "application/ics" {
		return true
	}

	for _, part := range payload.Parts {
		if hasCalendarPart(&part) {
			return true
		}
	}

	return false
}

// GetCalendarEvents extracts calendar events from a message.
// Note: This only extracts inline ICS data. Use GetCalendarEventsWithAttachments
// to also parse ICS attachments.
func (c *Client) GetCalendarEvents(msg *Message) []CalendarEvent {
	if msg.Payload == nil {
		return nil
	}

	var events []CalendarEvent

	icsData := extractCalendarData(msg.Payload)
	for _, ics := range icsData {
		parsed := parseICS(ics)
		events = append(events, parsed...)
	}

	return events
}

// GetCalendarEventsWithAttachments extracts calendar events from both inline
// content and ICS attachments.
func (c *Client) GetCalendarEventsWithAttachments(ctx context.Context, msg *Message) []CalendarEvent {
	// First get inline events
	events := c.GetCalendarEvents(msg)

	// Then check attachments for ICS files
	attachments := c.GetMessageAttachments(msg)
	for _, att := range attachments {
		if att.MimeType == "text/calendar" || att.MimeType == "application/ics" {
			data, err := c.GetAttachment(ctx, msg.ID, att.ID)
			if err != nil {
				continue
			}

			parsed := parseICS(string(data))
			events = append(events, parsed...)
		}
	}

	return events
}

// extractCalendarData recursively extracts ICS data from message parts.
func extractCalendarData(payload *MessagePayload) []string {
	var data []string

	if payload.MimeType == "text/calendar" || payload.MimeType == "application/ics" {
		if payload.Body != nil && payload.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
			if err == nil {
				data = append(data, string(decoded))
			}
		}
	}

	for _, part := range payload.Parts {
		data = append(data, extractCalendarData(&part)...)
	}

	return data
}

// parseICS parses ICS calendar data into CalendarEvent structs.
func parseICS(icsData string) []CalendarEvent {
	var events []CalendarEvent
	var currentEvent *CalendarEvent
	var method string

	lines := strings.Split(icsData, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Handle line folding (continuation lines start with space or tab)
		for i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
			i++
			line += strings.TrimSpace(lines[i])
		}

		if strings.HasPrefix(line, "METHOD:") {
			method = strings.TrimPrefix(line, "METHOD:")
		} else if line == "BEGIN:VEVENT" {
			currentEvent = &CalendarEvent{Method: method}
		} else if line == "END:VEVENT" && currentEvent != nil {
			events = append(events, *currentEvent)
			currentEvent = nil
		} else if currentEvent != nil {
			parseICSLine(line, currentEvent)
		}
	}

	return events
}

// parseICSLine parses a single ICS line into the event.
func parseICSLine(line string, event *CalendarEvent) {
	// Handle properties with parameters (e.g., DTSTART;TZID=America/New_York:20240101T090000)
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return
	}

	propPart := line[:colonIdx]
	value := line[colonIdx+1:]

	// Extract property name (before semicolon if parameters exist)
	propName := propPart
	if semiIdx := strings.Index(propPart, ";"); semiIdx != -1 {
		propName = propPart[:semiIdx]
	}

	switch propName {
	case "UID":
		event.UID = value
	case "SUMMARY":
		event.Summary = unescapeICS(value)
	case "DESCRIPTION":
		event.Description = unescapeICS(value)
	case "LOCATION":
		event.Location = unescapeICS(value)
	case "STATUS":
		event.Status = value
	case "DTSTART":
		event.Start, event.IsAllDay = parseICSDateTime(value, propPart)
	case "DTEND":
		event.End, _ = parseICSDateTime(value, propPart)
	case "ORGANIZER":
		// Format: ORGANIZER;CN=Name:mailto:email@example.com
		if strings.HasPrefix(value, "mailto:") {
			event.Organizer = strings.TrimPrefix(value, "mailto:")
		} else {
			event.Organizer = value
		}
	case "ATTENDEE":
		// Format: ATTENDEE;CN=Name:mailto:email@example.com
		email := value
		if strings.HasPrefix(value, "mailto:") {
			email = strings.TrimPrefix(value, "mailto:")
		}

		event.Attendees = append(event.Attendees, email)
	}
}

// parseICSDateTime parses ICS date/time formats.
func parseICSDateTime(value, propPart string) (time.Time, bool) {
	// Check for VALUE=DATE (all-day event)
	isAllDay := strings.Contains(propPart, "VALUE=DATE")

	// Common formats
	formats := []string{
		"20060102T150405Z",     // UTC
		"20060102T150405",      // Local
		"20060102",             // Date only (all-day)
		"2006-01-02T15:04:05Z", // ISO with dashes
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, isAllDay || len(value) == 8
		}
	}

	return time.Time{}, false
}

// unescapeICS unescapes ICS escaped characters.
func unescapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\\\", "\\")

	return s
}

// GetDriveLinks extracts Google Drive links from a message.
func (c *Client) GetDriveLinks(msg *Message) []DriveLink {
	if msg.Payload == nil {
		return nil
	}

	var links []DriveLink
	seen := make(map[string]bool)

	// Extract from HTML and plain text bodies
	bodies := []string{
		extractBody(msg.Payload, "text/plain"),
		extractBody(msg.Payload, "text/html"),
	}

	for _, body := range bodies {
		extracted := extractDriveLinks(body)
		for _, link := range extracted {
			if !seen[link.FileID] {
				seen[link.FileID] = true
				links = append(links, link)
			}
		}
	}

	return links
}

// extractDriveLinks finds Google Drive links in text.
func extractDriveLinks(text string) []DriveLink {
	var links []DriveLink

	// Patterns for Google Drive URLs
	patterns := []struct {
		prefix    string
		extractor func(string) (string, string)
	}{
		{"https://drive.google.com/file/d/", extractFileID},
		{"https://drive.google.com/open?id=", extractOpenID},
		{"https://docs.google.com/document/d/", extractDocID},
		{"https://docs.google.com/spreadsheets/d/", extractDocID},
		{"https://docs.google.com/presentation/d/", extractDocID},
		{"https://drive.google.com/drive/folders/", extractFolderID},
	}

	for _, p := range patterns {
		idx := 0
		for {
			pos := strings.Index(text[idx:], p.prefix)
			if pos == -1 {
				break
			}

			start := idx + pos
			end := start + len(p.prefix)

			// Find end of URL
			urlEnd := end
			for urlEnd < len(text) && !isURLTerminator(text[urlEnd]) {
				urlEnd++
			}

			fullURL := text[start:urlEnd]
			fileID, name := p.extractor(fullURL)

			if fileID != "" {
				links = append(links, DriveLink{
					URL:    fullURL,
					FileID: fileID,
					Name:   name,
				})
			}

			idx = urlEnd
		}
	}

	return links
}

// isURLTerminator checks if a character terminates a URL.
func isURLTerminator(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '"' || c == '\'' || c == '<' || c == '>'
}

// extractFileID extracts file ID from /file/d/ URLs.
func extractFileID(url string) (string, string) {
	// Format: https://drive.google.com/file/d/FILE_ID/view?...
	const prefix = "https://drive.google.com/file/d/"
	if !strings.HasPrefix(url, prefix) {
		return "", ""
	}

	rest := url[len(prefix):]
	if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
		return rest[:slashIdx], ""
	}

	if qIdx := strings.Index(rest, "?"); qIdx != -1 {
		return rest[:qIdx], ""
	}

	return rest, ""
}

// extractOpenID extracts file ID from /open?id= URLs.
func extractOpenID(url string) (string, string) {
	const prefix = "https://drive.google.com/open?id="
	if !strings.HasPrefix(url, prefix) {
		return "", ""
	}

	rest := url[len(prefix):]
	if ampIdx := strings.Index(rest, "&"); ampIdx != -1 {
		return rest[:ampIdx], ""
	}

	return rest, ""
}

// extractDocID extracts document ID from Google Docs URLs.
func extractDocID(url string) (string, string) {
	// Format: https://docs.google.com/document/d/DOC_ID/edit?...
	parts := strings.Split(url, "/d/")
	if len(parts) < 2 {
		return "", ""
	}

	rest := parts[1]
	if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
		return rest[:slashIdx], ""
	}

	if qIdx := strings.Index(rest, "?"); qIdx != -1 {
		return rest[:qIdx], ""
	}

	return rest, ""
}

// extractFolderID extracts folder ID from Drive folder URLs.
func extractFolderID(url string) (string, string) {
	const prefix = "https://drive.google.com/drive/folders/"
	if !strings.HasPrefix(url, prefix) {
		return "", ""
	}

	rest := url[len(prefix):]
	if qIdx := strings.Index(rest, "?"); qIdx != -1 {
		return rest[:qIdx], ""
	}

	return rest, ""
}

// HasDriveLinks checks if a message contains Google Drive links.
func (c *Client) HasDriveLinks(msg *Message) bool {
	return len(c.GetDriveLinks(msg)) > 0
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
