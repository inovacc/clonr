package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira operations for project management",
	Long: `Interact with Jira issues, sprints, and boards.

Available Commands:
  issues        Manage Jira issues (list, create, view, transition)
  sprints       Manage sprints (list, current)
  boards        List Jira boards

Authentication:
  Uses Jira API token from (in priority order):
  1. --token flag
  2. JIRA_API_TOKEN environment variable
  3. ATLASSIAN_TOKEN environment variable
  4. ~/.config/clonr/jira.json config file

Required Configuration:
  Jira requires both a base URL and email for API authentication.
  - Base URL: --url flag or JIRA_URL environment variable
  - Email: --email flag or JIRA_EMAIL environment variable

Examples:
  clonr pm jira issues list PROJ
  clonr pm jira issues create PROJ --summary "Bug report"
  clonr pm jira sprints current
  clonr pm jira boards list`,
}

var jiraIssuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage Jira issues",
	Long: `List, create, view, and transition Jira issues.

Examples:
  clonr pm jira issues list PROJ                           # List issues in project
  clonr pm jira issues list PROJ --status "In Progress"    # Filter by status
  clonr pm jira issues create PROJ --summary "Bug"         # Create an issue
  clonr pm jira issues view PROJ-123                       # View issue details
  clonr pm jira issues transition PROJ-123 "Done"          # Move issue to Done`,
}

var jiraSprintsCmd = &cobra.Command{
	Use:   "sprints",
	Short: "Manage Jira sprints",
	Long: `List and view Jira sprints.

Examples:
  clonr pm jira sprints list                 # List all sprints
  clonr pm jira sprints list --board 123     # List sprints for board
  clonr pm jira sprints current              # Show current sprint`,
}

var jiraBoardsCmd = &cobra.Command{
	Use:   "boards",
	Short: "Manage Jira boards",
	Long: `List Jira boards.

Examples:
  clonr pm jira boards list              # List all boards
  clonr pm jira boards list --project PROJ   # List boards for project`,
}

var jiraAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Open Jira/Atlassian token page in browser",
	Long: `Open the Atlassian API token settings page in your default browser.

This command helps you quickly access the token generation page:
  - Atlassian: https://id.atlassian.com/manage-profile/security/api-tokens

Examples:
  clonr pm jira auth`,
	RunE: runJiraAuth,
}

func init() {
	pmCmd.AddCommand(jiraCmd)
	jiraCmd.AddCommand(jiraIssuesCmd)
	jiraCmd.AddCommand(jiraSprintsCmd)
	jiraCmd.AddCommand(jiraBoardsCmd)
	jiraCmd.AddCommand(jiraAuthCmd)
}

// addJiraCommonFlags adds flags common to all jira subcommands
func addJiraCommonFlags(cmd *cobra.Command) {
	addPMCommonFlags(cmd)
	cmd.Flags().String("url", "", "Jira instance URL (e.g., https://company.atlassian.net)")
	cmd.Flags().String("email", "", "Jira account email")
}

func runJiraAuth(_ *cobra.Command, _ []string) error {
	_, _ = fmt.Fprintf(os.Stdout, "Opening Atlassian API token page: %s\n", core.JiraTokenURL)
	if err := core.OpenJiraTokenPage(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to open browser: %v\n", err)
		_, _ = fmt.Fprintf(os.Stdout, "Please visit: %s\n", core.JiraTokenURL)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nAfter creating a token, configure it:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  export JIRA_API_TOKEN=<token>\n")
	_, _ = fmt.Fprintf(os.Stdout, "  export JIRA_EMAIL=<your-email>\n")
	_, _ = fmt.Fprintf(os.Stdout, "  export JIRA_URL=https://company.atlassian.net\n")
	_, _ = fmt.Fprintf(os.Stdout, "\nOr create ~/.config/clonr/jira.json:\n")
	_, _ = fmt.Fprintf(os.Stdout, `  {
    "default_instance": "company",
    "instances": {
      "company": {
        "url": "https://company.atlassian.net",
        "email": "you@company.com",
        "token": "your-api-token"
      }
    }
  }
`)

	return nil
}
