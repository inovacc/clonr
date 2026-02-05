package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	driveAPIBaseURL = "https://www.googleapis.com/drive/v3"
)

// Client is a Google Drive API client.
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// ClientOptions configures a Google Drive client.
type ClientOptions struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
}

// NewClient creates a new Google Drive API client.
func NewClient(accessToken string, _ ClientOptions) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for downloads
		},
	}
}

// File represents a Google Drive file.
type File struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MimeType     string    `json:"mimeType"`
	Size         int64     `json:"size,string"`
	CreatedTime  time.Time `json:"createdTime"`
	ModifiedTime time.Time `json:"modifiedTime"`
	WebViewLink  string    `json:"webViewLink"`
	IconLink     string    `json:"iconLink"`
	Owners       []Owner   `json:"owners"`
	Shared       bool      `json:"shared"`
}

// Owner represents a file owner.
type Owner struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// GetFile retrieves file metadata by ID.
func (c *Client) GetFile(ctx context.Context, fileID string) (*File, error) {
	params := url.Values{}
	params.Set("fields", "id,name,mimeType,size,createdTime,modifiedTime,webViewLink,iconLink,owners,shared")

	var file File
	if err := c.get(ctx, fmt.Sprintf("files/%s", fileID), params, &file); err != nil {
		return nil, err
	}

	return &file, nil
}

// DownloadFile downloads a file's content by ID.
func (c *Client) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	// First get file metadata to check if it's a Google Doc type
	file, err := c.GetFile(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Google Docs need to be exported
	if isGoogleDoc(file.MimeType) {
		return c.exportFile(ctx, fileID, file.MimeType)
	}

	// Regular file download
	return c.downloadBinary(ctx, fileID)
}

// isGoogleDoc checks if the MIME type is a Google Workspace document.
func isGoogleDoc(mimeType string) bool {
	switch mimeType {
	case "application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/vnd.google-apps.drawing":
		return true
	}

	return false
}

// getExportMimeType returns the export MIME type for Google Docs.
func getExportMimeType(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.document":
		return "application/pdf" // or "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "application/vnd.google-apps.spreadsheet":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "application/vnd.google-apps.presentation":
		return "application/pdf"
	case "application/vnd.google-apps.drawing":
		return "image/png"
	}

	return "application/pdf"
}

// getExportExtension returns the file extension for exported Google Docs.
func getExportExtension(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.document":
		return ".pdf"
	case "application/vnd.google-apps.spreadsheet":
		return ".xlsx"
	case "application/vnd.google-apps.presentation":
		return ".pdf"
	case "application/vnd.google-apps.drawing":
		return ".png"
	}

	return ".pdf"
}

// GetExportExtension returns the file extension for a Google Doc export.
func GetExportExtension(mimeType string) string {
	return getExportExtension(mimeType)
}

// exportFile exports a Google Workspace file to a downloadable format.
func (c *Client) exportFile(ctx context.Context, fileID, mimeType string) ([]byte, error) {
	exportMime := getExportMimeType(mimeType)

	reqURL := fmt.Sprintf("%s/files/%s/export?mimeType=%s", driveAPIBaseURL, fileID, url.QueryEscape(exportMime))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// downloadBinary downloads a binary file.
func (c *Client) downloadBinary(ctx context.Context, fileID string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/files/%s?alt=media", driveAPIBaseURL, fileID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// ListFiles lists files in the user's Drive.
func (c *Client) ListFiles(ctx context.Context, query string, maxResults int) ([]File, error) {
	params := url.Values{}
	params.Set("fields", "files(id,name,mimeType,size,createdTime,modifiedTime,webViewLink)")

	if query != "" {
		params.Set("q", query)
	}

	if maxResults > 0 {
		params.Set("pageSize", fmt.Sprintf("%d", maxResults))
	}

	var resp struct {
		Files []File `json:"files"`
	}

	if err := c.get(ctx, "files", params, &resp); err != nil {
		return nil, err
	}

	return resp.Files, nil
}

// get performs a GET request to the Drive API.
func (c *Client) get(ctx context.Context, endpoint string, params url.Values, result any) error {
	reqURL := fmt.Sprintf("%s/%s", driveAPIBaseURL, endpoint)
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
