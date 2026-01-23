package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your GitHub organizations",
	Long: `List all GitHub organizations you belong to.

Shows organization details and local mirror status:
  - Organization name and login
  - Number of repositories on GitHub
  - Whether the organization has been mirrored locally
  - Number of locally mirrored repositories

Authentication:
  Token is automatically detected from (in order):
  - --token flag
  - GITHUB_TOKEN environment variable
  - GH_TOKEN environment variable
  - gh CLI (if authenticated via 'gh auth login')

Examples:
  # List all organizations
  clonr org list

  # Include personal repositories
  clonr org list --include-user

  # JSON output for scripting
  clonr org list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		token, _ := cmd.Flags().GetString("token")
		includeUser, _ := cmd.Flags().GetBool("include-user")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Resolve token
		token, tokenSource, err := core.ResolveGitHubToken(token)
		if err != nil {
			return err
		}

		if jsonOutput {
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			logger.Debug("token resolved", slog.String("source", string(tokenSource)))
		}

		_, _ = fmt.Fprintln(os.Stdout, "Fetching organizations...")

		opts := core.ListOrganizationsOptions{
			IncludeUser: includeUser,
		}

		orgs, err := core.ListOrganizations(token, opts)
		if err != nil {
			return err
		}

		if len(orgs) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "No organizations found.")
			return nil
		}

		if jsonOutput {
			printOrgsJSON(orgs)
		} else {
			printOrgsTable(orgs)
		}

		return nil
	},
}

func printOrgsTable(orgs []core.Organization) {
	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	mirroredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	notMirroredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	// Calculate column widths
	maxLogin := 10
	maxName := 10

	for _, org := range orgs {
		if len(org.Login) > maxLogin {
			maxLogin = len(org.Login)
		}

		if len(org.Name) > maxName && len(org.Name) <= 30 {
			maxName = len(org.Name)
		}
	}

	if maxName > 30 {
		maxName = 30
	}

	// Print header
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s\n",
		headerStyle.Render(padRight("LOGIN", maxLogin)),
		headerStyle.Render(padRight("NAME", maxName)),
		headerStyle.Render(padRight("REPOS", 7)),
		headerStyle.Render(padRight("MIRRORED", 10)),
		headerStyle.Render("LOCAL"),
	)
	_, _ = fmt.Fprintln(os.Stdout, strings.Repeat("-", maxLogin+maxName+35))

	// Print rows
	for _, org := range orgs {
		name := org.Name
		if len(name) > maxName {
			name = name[:maxName-3] + "..."
		}

		var mirrorStatus string
		if org.IsMirrored {
			mirrorStatus = mirroredStyle.Render(padRight("Yes", 10))
		} else {
			mirrorStatus = notMirroredStyle.Render(padRight("No", 10))
		}

		localCount := "-"
		if org.IsMirrored {
			localCount = countStyle.Render(fmt.Sprintf("%d", org.LocalRepos))
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s  %s  %s  %s  %s\n",
			padRight(org.Login, maxLogin),
			padRight(name, maxName),
			countStyle.Render(padRight(fmt.Sprintf("%d", org.RepoCount), 7)),
			mirrorStatus,
			localCount,
		)
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "Total: %d organizations\n", len(orgs))
}

func printOrgsJSON(orgs []core.Organization) {
	_, _ = fmt.Fprintln(os.Stdout, "[")

	for i, org := range orgs {
		comma := ","
		if i == len(orgs)-1 {
			comma = ""
		}

		_, _ = fmt.Fprintf(os.Stdout, `  {"login": %q, "name": %q, "repos": %d, "mirrored": %t, "local_repos": %d}%s`+"\n",
			org.Login, org.Name, org.RepoCount, org.IsMirrored, org.LocalRepos, comma)
	}

	_, _ = fmt.Fprintln(os.Stdout, "]")
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}

	return s + strings.Repeat(" ", length-len(s))
}

func init() {
	orgCmd.AddCommand(orgListCmd)

	orgListCmd.Flags().String("token", "", "GitHub personal access token")
	orgListCmd.Flags().Bool("include-user", false, "Include personal repositories as pseudo-organization")
	orgListCmd.Flags().Bool("json", false, "Output in JSON format")
}
