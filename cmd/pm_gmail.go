package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/gdrive"
	"github.com/inovacc/clonr/internal/gmail"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	pmCmd.AddCommand(pmGmailCmd)
	pmGmailCmd.AddCommand(pmGmailProfileCmd)
	pmGmailCmd.AddCommand(pmGmailLabelsCmd)
	pmGmailCmd.AddCommand(pmGmailMessagesCmd)
	pmGmailCmd.AddCommand(pmGmailReadCmd)
	pmGmailCmd.AddCommand(pmGmailSearchCmd)
	pmGmailCmd.AddCommand(pmGmailAttachmentsCmd)
	pmGmailCmd.AddCommand(pmGmailDownloadCmd)
	pmGmailCmd.AddCommand(pmGmailCalendarCmd)
	pmGmailCmd.AddCommand(pmGmailDriveCmd)
	pmGmailCmd.AddCommand(pmGmailDriveDownloadCmd)

	// Messages flags
	pmGmailMessagesCmd.Flags().IntP("limit", "n", 10, "Maximum number of messages to list")
	pmGmailMessagesCmd.Flags().StringP("label", "l", "INBOX", "Label to filter messages (INBOX, SENT, etc.)")
	pmGmailMessagesCmd.Flags().Bool("json", false, "Output as JSON")

	// Search flags
	pmGmailSearchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	pmGmailSearchCmd.Flags().Bool("json", false, "Output as JSON")

	// Read flags
	pmGmailReadCmd.Flags().Bool("html", false, "Show HTML body instead of plain text")
	pmGmailReadCmd.Flags().Bool("json", false, "Output as JSON")

	// Attachments flags
	pmGmailAttachmentsCmd.Flags().Bool("json", false, "Output as JSON")

	// Download flags
	pmGmailDownloadCmd.Flags().StringP("output", "o", "", "Output directory (default: current directory)")

	// Calendar flags
	pmGmailCalendarCmd.Flags().Bool("json", false, "Output as JSON")

	// Drive flags
	pmGmailDriveCmd.Flags().Bool("json", false, "Output as JSON")
	pmGmailDriveDownloadCmd.Flags().StringP("output", "o", "", "Output directory (default: current directory)")
}

var pmGmailCmd = &cobra.Command{
	Use:   "gmail",
	Short: "Gmail operations",
	Long: `Gmail operations for the active profile.

Available Commands:
  profile      Show Gmail account profile
  labels       List Gmail labels
  messages     List recent messages
  read         Read a specific message
  search       Search messages
  attachments  List attachments in a message
  download     Download an attachment
  calendar     Show calendar events in a message
  drive        List Google Drive links in a message
  drive-download  Download a file from Google Drive

Examples:
  clonr pm gmail profile
  clonr pm gmail labels
  clonr pm gmail messages
  clonr pm gmail messages --limit 20 --label INBOX
  clonr pm gmail read <message-id>
  clonr pm gmail search "from:someone@example.com"
  clonr pm gmail calendar <message-id>
  clonr pm gmail drive <message-id>
  clonr pm gmail drive-download <file-id>`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var pmGmailProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show Gmail account profile",
	RunE:  runPmGmailProfile,
}

var pmGmailLabelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "List Gmail labels",
	RunE:  runPmGmailLabels,
}

var pmGmailMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List recent messages",
	RunE:  runPmGmailMessages,
}

var pmGmailReadCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific message",
	Args:  cobra.ExactArgs(1),
	RunE:  runPmGmailRead,
}

var pmGmailSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages using Gmail query syntax",
	Long: `Search messages using Gmail query syntax.

Query examples:
  from:someone@example.com     Messages from a specific sender
  to:me                        Messages sent to you
  subject:meeting              Messages with "meeting" in subject
  is:unread                    Unread messages
  has:attachment               Messages with attachments
  after:2024/01/01             Messages after a date
  before:2024/12/31            Messages before a date
  label:important              Messages with a specific label

Examples:
  clonr pm gmail search "from:boss@company.com"
  clonr pm gmail search "is:unread has:attachment"
  clonr pm gmail search "subject:invoice after:2024/01/01"`,
	Args: cobra.ExactArgs(1),
	RunE: runPmGmailSearch,
}

var pmGmailAttachmentsCmd = &cobra.Command{
	Use:   "attachments <message-id>",
	Short: "List attachments in a message",
	Long: `List all attachments in a Gmail message.

Examples:
  clonr pm gmail attachments 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runPmGmailAttachments,
}

var pmGmailDownloadCmd = &cobra.Command{
	Use:   "download <message-id> <attachment-id>",
	Short: "Download an attachment from a message",
	Long: `Download an attachment from a Gmail message.

Use 'clonr pm gmail attachments <message-id>' to get attachment IDs.

Examples:
  clonr pm gmail download 19c2d20451b4bb54 ANGjdJ8abc123
  clonr pm gmail download 19c2d20451b4bb54 ANGjdJ8abc123 -o ~/Downloads`,
	Args: cobra.ExactArgs(2),
	RunE: runPmGmailDownload,
}

var pmGmailCalendarCmd = &cobra.Command{
	Use:   "calendar <message-id>",
	Short: "Show calendar events in a message",
	Long: `Extract and display calendar events from a Gmail message.

Detects ICS calendar attachments and displays event details including:
- Event title, date/time, location
- Organizer and attendees
- Event status (confirmed, tentative, cancelled)

Examples:
  clonr pm gmail calendar 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runPmGmailCalendar,
}

var pmGmailDriveCmd = &cobra.Command{
	Use:   "drive <message-id>",
	Short: "List Google Drive links in a message",
	Long: `Extract and display Google Drive links from a Gmail message.

Detects links to:
- Google Drive files and folders
- Google Docs, Sheets, and Slides

Examples:
  clonr pm gmail drive 19c2d20451b4bb54`,
	Args: cobra.ExactArgs(1),
	RunE: runPmGmailDrive,
}

var pmGmailDriveDownloadCmd = &cobra.Command{
	Use:   "drive-download <file-id>",
	Short: "Download a file from Google Drive",
	Long: `Download a file from Google Drive using its file ID.

Get file IDs using 'clonr pm gmail drive <message-id>'.

For Google Docs/Sheets/Slides, the file is exported to PDF/XLSX format.

Examples:
  clonr pm gmail drive-download 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs
  clonr pm gmail drive-download 1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs -o ~/Downloads`,
	Args: cobra.ExactArgs(1),
	RunE: runPmGmailDriveDownload,
}

func getGmailClient() (*gmail.Client, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Gmail integration configured; add with: clonr profile gmail add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Gmail config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Gmail config")
	}

	return gmail.NewClient(accessToken, gmail.ClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
	}), nil
}

func runPmGmailProfile(_ *cobra.Command, _ []string) error {
	client, err := getGmailClient()
	if err != nil {
		return err
	}

	profile, err := client.GetProfile(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	printBoxHeader("GMAIL PROFILE")
	printBoxLine("Email", profile.EmailAddress)
	printBoxLine("Messages", fmt.Sprintf("%d", profile.MessagesTotal))
	printBoxLine("Threads", fmt.Sprintf("%d", profile.ThreadsTotal))
	printBoxFooter()

	return nil
}

func runPmGmailLabels(_ *cobra.Command, _ []string) error {
	client, err := getGmailClient()
	if err != nil {
		return err
	}

	labels, err := client.ListLabels(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Gmail Labels:")
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, label := range labels {
		if label.Type == "system" {
			_, _ = fmt.Fprintf(os.Stdout, "  %s (system)\n", label.Name)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", label.Name)
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	return nil
}

func runPmGmailMessages(cmd *cobra.Command, _ []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	label, _ := cmd.Flags().GetString("label")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	opts := gmail.ListMessagesOptions{
		MaxResults: limit,
		LabelIDs:   []string{label},
	}

	resp, err := client.ListMessages(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(resp.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found.")
		return nil
	}

	type messageInfo struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}

	var messages []messageInfo

	for _, ref := range resp.Messages {
		msg, msgErr := client.GetMessage(context.Background(), ref.ID, "metadata")
		if msgErr != nil {
			continue
		}

		info := messageInfo{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Snippet: msg.Snippet,
		}
		messages = append(messages, info)
	}

	if jsonOutput {
		return outputJSON(messages)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Messages in %s (%d):\n", label, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, msg := range messages {
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "  From:    %s\n", msg.From)
		_, _ = fmt.Fprintf(os.Stdout, "  Subject: %s\n", msg.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "  Date:    %s\n", msg.Date)

		if msg.Snippet != "" {
			snippet := msg.Snippet
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  Preview: %s\n", dimStyle.Render(snippet))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmGmailRead(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	showHTML, _ := cmd.Flags().GetBool("html")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if jsonOutput {
		type messageDetail struct {
			ID      string `json:"id"`
			From    string `json:"from"`
			To      string `json:"to"`
			Subject string `json:"subject"`
			Date    string `json:"date"`
			Body    string `json:"body"`
		}

		var body string
		if showHTML {
			body = client.GetMessageHTMLBody(msg)
		} else {
			body = client.GetMessageBody(msg)
		}

		detail := messageDetail{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			To:      msg.Headers["to"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Body:    body,
		}

		return outputJSON(detail)
	}

	attachments := client.GetMessageAttachments(msg)
	hasCalendar := client.HasCalendarEvent(msg)
	driveLinks := client.GetDriveLinks(msg)

	printBoxHeader("MESSAGE")
	printBoxLine("ID", msg.ID)
	printBoxLine("From", msg.Headers["from"])
	printBoxLine("To", msg.Headers["to"])
	printBoxLine("Subject", msg.Headers["subject"])
	printBoxLine("Date", msg.Headers["date"])

	if len(attachments) > 0 {
		printBoxLine("Attachments", fmt.Sprintf("%d file(s)", len(attachments)))
	}

	if hasCalendar {
		printBoxLine("Calendar", "Event detected")
	}

	if len(driveLinks) > 0 {
		printBoxLine("Drive Links", fmt.Sprintf("%d link(s)", len(driveLinks)))
	}

	printBoxFooter()

	// Show attachments if any
	if len(attachments) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Attachments:")

		for _, att := range attachments {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s)\n", att.Filename, formatAttachmentSize(att.Size))
		}
	}

	// Show calendar events if detected
	if hasCalendar {
		events := client.GetCalendarEventsWithAttachments(context.Background(), msg)
		if len(events) > 0 {
			// Deduplicate events by UID
			seen := make(map[string]bool)
			var uniqueEvents []gmail.CalendarEvent

			for _, event := range events {
				if !seen[event.UID] {
					seen[event.UID] = true
					uniqueEvents = append(uniqueEvents, event)
				}
			}

			_, _ = fmt.Fprintln(os.Stdout, "")
			_, _ = fmt.Fprintln(os.Stdout, "Calendar Events:")

			for _, event := range uniqueEvents {
				if event.IsAllDay {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s, all day)\n", event.Summary, event.Start.Format("2006-01-02"))
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s)\n", event.Summary, event.Start.Format("2006-01-02 15:04"))
				}
			}

			_, _ = fmt.Fprintln(os.Stdout, "")
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("Use 'clonr pm gmail calendar "+msg.ID+"' for details"))
		}
	}

	// Show Drive links if detected
	if len(driveLinks) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Google Drive Links:")

		for _, link := range driveLinks {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", dimStyle.Render(link.FileID))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("Use 'clonr pm gmail drive "+msg.ID+"' for details"))
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	var body string
	if showHTML {
		body = client.GetMessageHTMLBody(msg)
	} else {
		body = client.GetMessageBody(msg)
	}

	if body != "" {
		_, _ = fmt.Fprintln(os.Stdout, body)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("(no body content)"))
	}

	return nil
}

func runPmGmailSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	resp, err := client.SearchMessages(context.Background(), query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(resp.Messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found matching query.")
		return nil
	}

	type searchResult struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}

	var results []searchResult

	for _, ref := range resp.Messages {
		msg, msgErr := client.GetMessage(context.Background(), ref.ID, "metadata")
		if msgErr != nil {
			continue
		}

		result := searchResult{
			ID:      msg.ID,
			From:    msg.Headers["from"],
			Subject: msg.Headers["subject"],
			Date:    msg.Headers["date"],
			Snippet: msg.Snippet,
		}
		results = append(results, result)
	}

	if jsonOutput {
		return outputJSON(results)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Search results for %q (%d):\n", query, len(results))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, result := range results {
		_, _ = fmt.Fprintf(os.Stdout, "  %s %s\n", dimStyle.Render(strconv.Itoa(i+1)+"."), dimStyle.Render(result.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     From:    %s\n", result.From)
		_, _ = fmt.Fprintf(os.Stdout, "     Subject: %s\n", result.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "     Date:    %s\n", result.Date)

		if result.Snippet != "" {
			snippet := result.Snippet
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Preview: %s\n", dimStyle.Render(snippet))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

// formatEmailDate formats an email date string for display
func formatEmailDate(dateStr string) string {
	// Try parsing common email date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}

	return dateStr
}

func runPmGmailAttachments(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	attachments := client.GetMessageAttachments(msg)

	if len(attachments) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No attachments found in this message.")
		return nil
	}

	if jsonOutput {
		return outputJSON(attachments)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Attachments (%d):\n", len(attachments))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, att := range attachments {
		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, att.Filename)
		_, _ = fmt.Fprintf(os.Stdout, "     ID:   %s\n", dimStyle.Render(att.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     Type: %s\n", att.MimeType)
		_, _ = fmt.Fprintf(os.Stdout, "     Size: %s\n", formatAttachmentSize(att.Size))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	_, _ = fmt.Fprintln(os.Stdout, "To download: clonr pm gmail download", messageID, "<attachment-id>")

	return nil
}

func runPmGmailDownload(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	attachmentID := args[1]
	outputDir, _ := cmd.Flags().GetString("output")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	// First get the message to find the attachment filename
	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	attachments := client.GetMessageAttachments(msg)

	var filename string

	for _, att := range attachments {
		if att.ID == attachmentID {
			filename = att.Filename

			break
		}
	}

	if filename == "" {
		return fmt.Errorf("attachment not found: %s", attachmentID)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Downloading %s...\n", filename)

	// Download the attachment
	data, err := client.GetAttachment(context.Background(), messageID, attachmentID)
	if err != nil {
		return fmt.Errorf("failed to download attachment: %w", err)
	}

	// Determine output path
	outputPath := filename
	if outputDir != "" {
		outputPath = outputDir + "/" + filename
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Saved: %s (%s)", outputPath, formatAttachmentSize(len(data)))))

	return nil
}

// formatAttachmentSize formats attachment size for display
func formatAttachmentSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}

	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func runPmGmailCalendar(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getGmailClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if !client.HasCalendarEvent(msg) {
		_, _ = fmt.Fprintln(os.Stdout, "No calendar events found in this message.")

		return nil
	}

	// Use method that also parses ICS attachments
	allEvents := client.GetCalendarEventsWithAttachments(context.Background(), msg)

	if len(allEvents) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No calendar events could be parsed from this message.")

		return nil
	}

	// Deduplicate events by UID
	seen := make(map[string]bool)
	var events []gmail.CalendarEvent

	for _, event := range allEvents {
		if !seen[event.UID] {
			seen[event.UID] = true
			events = append(events, event)
		}
	}

	if jsonOutput {
		return outputJSON(events)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Calendar Events (%d):\n", len(events))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, event := range events {
		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, event.Summary)

		if event.Method != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Type:      %s\n", event.Method)
		}

		if event.IsAllDay {
			_, _ = fmt.Fprintf(os.Stdout, "     Date:      %s (all day)\n", event.Start.Format("2006-01-02"))
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "     Start:     %s\n", event.Start.Format("2006-01-02 15:04"))
			_, _ = fmt.Fprintf(os.Stdout, "     End:       %s\n", event.End.Format("2006-01-02 15:04"))
		}

		if event.Location != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Location:  %s\n", event.Location)
		}

		if event.Organizer != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Organizer: %s\n", event.Organizer)
		}

		if len(event.Attendees) > 0 {
			if len(event.Attendees) <= 3 {
				_, _ = fmt.Fprintf(os.Stdout, "     Attendees: %s\n", strings.Join(event.Attendees, ", "))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "     Attendees: %s (+%d more)\n",
					strings.Join(event.Attendees[:3], ", "), len(event.Attendees)-3)
			}
		}

		if event.Status != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Status:    %s\n", event.Status)
		}

		if event.Description != "" {
			desc := event.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Details:   %s\n", dimStyle.Render(desc))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmGmailDrive(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	gmailClient, err := getGmailClient()
	if err != nil {
		return err
	}

	msg, err := gmailClient.GetMessage(context.Background(), messageID, "full")
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	links := gmailClient.GetDriveLinks(msg)

	if len(links) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No Google Drive links found in this message.")

		return nil
	}

	// Try to get metadata for each link
	driveClient, driveErr := getDriveClient()

	type linkInfo struct {
		URL      string `json:"url"`
		FileID   string `json:"file_id"`
		Name     string `json:"name,omitempty"`
		MimeType string `json:"mime_type,omitempty"`
		Size     string `json:"size,omitempty"`
		Owner    string `json:"owner,omitempty"`
	}

	var infos []linkInfo

	for _, link := range links {
		info := linkInfo{
			URL:    link.URL,
			FileID: link.FileID,
		}

		// Try to get file metadata if Drive client is available
		if driveErr == nil && driveClient != nil {
			if file, fileErr := driveClient.GetFile(context.Background(), link.FileID); fileErr == nil {
				info.Name = file.Name
				info.MimeType = file.MimeType
				info.Size = formatFileSizeInt64(file.Size)

				if len(file.Owners) > 0 {
					info.Owner = file.Owners[0].EmailAddress
				}
			}
		}

		infos = append(infos, info)
	}

	if jsonOutput {
		return outputJSON(infos)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Google Drive Links (%d):\n", len(infos))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, info := range infos {
		name := info.Name
		if name == "" {
			name = "(unable to retrieve metadata)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, name)
		_, _ = fmt.Fprintf(os.Stdout, "     ID:   %s\n", dimStyle.Render(info.FileID))

		if info.MimeType != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Type: %s\n", info.MimeType)
		}

		if info.Size != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Size: %s\n", info.Size)
		}

		if info.Owner != "" {
			_, _ = fmt.Fprintf(os.Stdout, "     Owner: %s\n", info.Owner)
		}

		_, _ = fmt.Fprintf(os.Stdout, "     URL:  %s\n", dimStyle.Render(info.URL))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	_, _ = fmt.Fprintln(os.Stdout, "To download: clonr pm gmail drive-download <file-id>")

	return nil
}

func runPmGmailDriveDownload(cmd *cobra.Command, args []string) error {
	fileID := args[0]
	outputDir, _ := cmd.Flags().GetString("output")

	driveClient, err := getDriveClient()
	if err != nil {
		return err
	}

	// Get file metadata first
	file, err := driveClient.GetFile(context.Background(), fileID)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Downloading %s...\n", file.Name)

	// Download the file
	data, err := driveClient.DownloadFile(context.Background(), fileID)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Determine output filename
	filename := file.Name

	// Add extension for Google Docs exports
	if strings.HasPrefix(file.MimeType, "application/vnd.google-apps.") {
		ext := gdrive.GetExportExtension(file.MimeType)
		if ext != "" && !strings.HasSuffix(filename, ext) {
			filename += ext
		}
	}

	// Determine output path
	outputPath := filename
	if outputDir != "" {
		outputPath = filepath.Join(outputDir, filename)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(fmt.Sprintf("Saved: %s (%s)", outputPath, formatFileSizeInt64(int64(len(data))))))

	return nil
}

func getDriveClient() (*gdrive.Client, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelGmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gmail config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Gmail integration configured; add with: clonr profile gmail add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Gmail config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Gmail config")
	}

	return gdrive.NewClient(accessToken, gdrive.ClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
	}), nil
}

// formatFileSizeInt64 formats file size for display (int64 version)
func formatFileSizeInt64(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}

	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}

	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}
