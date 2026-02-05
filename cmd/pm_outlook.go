package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/microsoft"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	pmCmd.AddCommand(pmOutlookCmd)
	pmOutlookCmd.AddCommand(pmOutlookFoldersCmd)
	pmOutlookCmd.AddCommand(pmOutlookMessagesCmd)
	pmOutlookCmd.AddCommand(pmOutlookReadCmd)
	pmOutlookCmd.AddCommand(pmOutlookSearchCmd)

	// Flags
	pmOutlookFoldersCmd.Flags().Bool("json", false, "Output as JSON")

	pmOutlookMessagesCmd.Flags().IntP("limit", "n", 10, "Maximum number of messages")
	pmOutlookMessagesCmd.Flags().StringP("folder", "f", "inbox", "Folder to list messages from")
	pmOutlookMessagesCmd.Flags().Bool("json", false, "Output as JSON")

	pmOutlookReadCmd.Flags().Bool("json", false, "Output as JSON")

	pmOutlookSearchCmd.Flags().IntP("limit", "n", 10, "Maximum number of results")
	pmOutlookSearchCmd.Flags().Bool("json", false, "Output as JSON")
}

var pmOutlookCmd = &cobra.Command{
	Use:   "outlook",
	Short: "Microsoft Outlook operations",
	Long: `Microsoft Outlook operations for the active profile.

Available Commands:
  folders    List mail folders
  messages   List messages in a folder
  read       Read a specific message
  search     Search messages

Examples:
  clonr pm outlook folders
  clonr pm outlook messages
  clonr pm outlook messages --folder sentitems
  clonr pm outlook read <message-id>
  clonr pm outlook search "project update"`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var pmOutlookFoldersCmd = &cobra.Command{
	Use:   "folders",
	Short: "List mail folders",
	RunE:  runPmOutlookFolders,
}

var pmOutlookMessagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List messages in a folder",
	RunE:  runPmOutlookMessages,
}

var pmOutlookReadCmd = &cobra.Command{
	Use:   "read <message-id>",
	Short: "Read a specific message",
	Args:  cobra.ExactArgs(1),
	RunE:  runPmOutlookRead,
}

var pmOutlookSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search messages",
	Long: `Search messages in Outlook.

The search query uses Microsoft's search syntax.

Examples:
  clonr pm outlook search "project update"
  clonr pm outlook search "from:boss@company.com"
  clonr pm outlook search "hasattachment:true"`,
	Args: cobra.ExactArgs(1),
	RunE: runPmOutlookSearch,
}

func getOutlookClient() (*microsoft.OutlookClient, error) {
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	profile, err := pm.GetActiveProfile()
	if err != nil {
		return nil, fmt.Errorf("no active profile")
	}

	channel, err := pm.GetNotifyChannelByType(profile.Name, model.ChannelOutlook)
	if err != nil {
		return nil, fmt.Errorf("failed to get Outlook config: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("no Outlook integration configured; add with: clonr profile outlook add")
	}

	config, err := pm.DecryptChannelConfig(profile.Name, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt Outlook config: %w", err)
	}

	accessToken := config["access_token"]
	if accessToken == "" {
		return nil, fmt.Errorf("no access token found in Outlook config")
	}

	return microsoft.NewOutlookClient(accessToken, microsoft.OutlookClientOptions{
		RefreshToken: config["refresh_token"],
		ClientID:     config["client_id"],
		ClientSecret: config["client_secret"],
		TenantID:     config["tenant_id"],
	}), nil
}

func runPmOutlookFolders(cmd *cobra.Command, _ []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getOutlookClient()
	if err != nil {
		return err
	}

	folders, err := client.GetMailFolders(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list folders: %w", err)
	}

	if len(folders) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No folders found.")
		return nil
	}

	if jsonOutput {
		return outputJSON(folders)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, "Mail Folders:")
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, folder := range folders {
		unread := ""
		if folder.UnreadItemCount > 0 {
			unread = fmt.Sprintf(" (%d unread)", folder.UnreadItemCount)
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s - %d messages%s\n", folder.DisplayName, folder.TotalItemCount, unread)
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render("ID: "+folder.ID))
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmOutlookMessages(cmd *cobra.Command, _ []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	folder, _ := cmd.Flags().GetString("folder")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getOutlookClient()
	if err != nil {
		return err
	}

	messages, err := client.ListMessages(context.Background(), microsoft.ListMailMessagesOptions{
		Top:      limit,
		FolderID: folder,
	})
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found.")
		return nil
	}

	if jsonOutput {
		type msgInfo struct {
			ID          string `json:"id"`
			Subject     string `json:"subject"`
			From        string `json:"from"`
			Received    string `json:"received"`
			IsRead      bool   `json:"is_read"`
			BodyPreview string `json:"body_preview"`
		}

		var infos []msgInfo

		for _, msg := range messages {
			from := ""
			if msg.From != nil && msg.From.EmailAddress != nil {
				from = msg.From.EmailAddress.Address
			}

			infos = append(infos, msgInfo{
				ID:          msg.ID,
				Subject:     msg.Subject,
				From:        from,
				Received:    msg.ReceivedDateTime.Format(time.RFC3339),
				IsRead:      msg.IsRead,
				BodyPreview: msg.BodyPreview,
			})
		}

		return outputJSON(infos)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Messages in %s (%d):\n", folder, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for _, msg := range messages {
		from := "Unknown"
		if msg.From != nil && msg.From.EmailAddress != nil {
			from = fmt.Sprintf("%s <%s>", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
		}

		readStatus := ""
		if !msg.IsRead {
			readStatus = " [UNREAD]"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "  Subject: %s%s\n", msg.Subject, readStatus)
		_, _ = fmt.Fprintf(os.Stdout, "  From:    %s\n", from)
		_, _ = fmt.Fprintf(os.Stdout, "  Date:    %s\n", msg.ReceivedDateTime.Format("2006-01-02 15:04"))

		if msg.BodyPreview != "" {
			preview := msg.BodyPreview
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "  Preview: %s\n", dimStyle.Render(preview))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}

func runPmOutlookRead(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getOutlookClient()
	if err != nil {
		return err
	}

	msg, err := client.GetMessage(context.Background(), messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	if jsonOutput {
		return outputJSON(msg)
	}

	from := "Unknown"
	if msg.From != nil && msg.From.EmailAddress != nil {
		from = fmt.Sprintf("%s <%s>", msg.From.EmailAddress.Name, msg.From.EmailAddress.Address)
	}

	var toAddrs []string
	for _, r := range msg.ToRecipients {
		if r.EmailAddress != nil {
			toAddrs = append(toAddrs, r.EmailAddress.Address)
		}
	}

	to := "Unknown"
	if len(toAddrs) > 0 {
		to = toAddrs[0]
		if len(toAddrs) > 1 {
			to += fmt.Sprintf(" (+%d more)", len(toAddrs)-1)
		}
	}

	printBoxHeader("MESSAGE")
	printBoxLine("ID", msg.ID)
	printBoxLine("Subject", msg.Subject)
	printBoxLine("From", from)
	printBoxLine("To", to)
	printBoxLine("Date", msg.ReceivedDateTime.Format("2006-01-02 15:04:05"))

	if msg.HasAttachments {
		printBoxLine("Attachments", "Yes")
	}

	printBoxFooter()

	_, _ = fmt.Fprintln(os.Stdout, "")

	if msg.Body != nil && msg.Body.Content != "" {
		_, _ = fmt.Fprintln(os.Stdout, msg.Body.Content)
	} else if msg.BodyPreview != "" {
		_, _ = fmt.Fprintln(os.Stdout, msg.BodyPreview)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("(no body content)"))
	}

	return nil
}

func runPmOutlookSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	client, err := getOutlookClient()
	if err != nil {
		return err
	}

	messages, err := client.SearchMessages(context.Background(), query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(messages) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No messages found matching query.")
		return nil
	}

	if jsonOutput {
		return outputJSON(messages)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintf(os.Stdout, "Search results for %q (%d):\n", query, len(messages))
	_, _ = fmt.Fprintln(os.Stdout, "")

	for i, msg := range messages {
		from := "Unknown"
		if msg.From != nil && msg.From.EmailAddress != nil {
			from = msg.From.EmailAddress.Address
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %d. %s\n", i+1, dimStyle.Render(msg.ID))
		_, _ = fmt.Fprintf(os.Stdout, "     Subject: %s\n", msg.Subject)
		_, _ = fmt.Fprintf(os.Stdout, "     From:    %s\n", from)
		_, _ = fmt.Fprintf(os.Stdout, "     Date:    %s\n", msg.ReceivedDateTime.Format("2006-01-02 15:04"))

		if msg.BodyPreview != "" {
			preview := msg.BodyPreview
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}

			_, _ = fmt.Fprintf(os.Stdout, "     Preview: %s\n", dimStyle.Render(preview))
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}
