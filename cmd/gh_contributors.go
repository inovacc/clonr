package cmd

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var contributorsCmd = &cobra.Command{
	Use:   "contributors",
	Short: "View repository contributors and their activity",
	Long: `List contributors to a repository and view their contribution journey.

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Examples:
  clonr gh contributors list                     # List contributors
  clonr gh contributors list owner/repo          # List contributors for specific repo
  clonr gh contributors journey username         # View contributor's journey
  clonr gh contributors journey username --all   # Include commits, PRs, and issues`,
}

var contributorsListCmd = &cobra.Command{
	Use:   "list [owner/repo]",
	Short: "List repository contributors",
	Long: `List all contributors to a repository, sorted by contribution count.

Examples:
  clonr gh contributors list                    # List contributors in current repo
  clonr gh contributors list owner/repo         # List contributors in specified repo
  clonr gh contributors list --limit 10         # Show top 10 contributors`,
	RunE: runContributorsList,
}

var contributorsJourneyCmd = &cobra.Command{
	Use:   "journey <username> [owner/repo]",
	Short: "View a contributor's activity journey",
	Long: `View a contributor's complete activity journey in a repository.

Shows:
  - Commits made by the contributor
  - Pull requests created
  - Issues opened
  - First and last activity dates

Examples:
  clonr gh contributors journey octocat               # Journey in current repo
  clonr gh contributors journey octocat owner/repo    # Journey in specified repo
  clonr gh contributors journey octocat --commits     # Show only commits
  clonr gh contributors journey octocat --all         # Show commits, PRs, and issues`,
	Args: cobra.MinimumNArgs(1),
	RunE: runContributorsJourney,
}

func init() {
	ghCmd.AddCommand(contributorsCmd)
	contributorsCmd.AddCommand(contributorsListCmd)
	contributorsCmd.AddCommand(contributorsJourneyCmd)

	// List flags
	addGHCommonFlags(contributorsListCmd)
	contributorsListCmd.Flags().Int("limit", 30, "Maximum number of contributors to show (0 = unlimited)")

	// Journey flags
	addGHCommonFlags(contributorsJourneyCmd)
	contributorsJourneyCmd.Flags().Bool("commits", false, "Include commits")
	contributorsJourneyCmd.Flags().Bool("prs", false, "Include pull requests")
	contributorsJourneyCmd.Flags().Bool("issues", false, "Include issues")
	contributorsJourneyCmd.Flags().Bool("all", false, "Include all activity (commits, PRs, issues)")
	contributorsJourneyCmd.Flags().Int("limit", 20, "Maximum items per category")
}

func runContributorsList(cmd *cobra.Command, args []string) error {
	flags := extractGHFlags(cmd)
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := core.ResolveGitHubToken(flags.Token, flags.Profile)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := detectRepo(args, flags.Repo, "Specify a repository with: clonr gh contributors list owner/repo")
	if err != nil {
		return err
	}

	if !flags.JSON {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching contributors for %s/%s...\n", owner, repo)
	}

	opts := core.ListContributorsOptions{
		Limit: limit,
	}

	result, err := core.ListContributors(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to list contributors: %w", err)
	}

	if flags.JSON {
		return outputJSON(result)
	}

	// Text output
	if len(result.Contributors) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No contributors found for %s/%s\n", owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nContributors for %s (%d)\n\n", result.Repository, result.TotalCount)

	for i, c := range result.Contributors {
		_, _ = fmt.Fprintf(os.Stdout, "%3d. @%-20s %5d contributions\n", i+1, c.Login, c.Contributions)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nUse 'clonr gh contributors journey <username>' to view a contributor's activity\n")

	return nil
}

func runContributorsJourney(cmd *cobra.Command, args []string) error {
	flags := extractGHFlags(cmd)
	includeCommits, _ := cmd.Flags().GetBool("commits")
	includePRs, _ := cmd.Flags().GetBool("prs")
	includeIssues, _ := cmd.Flags().GetBool("issues")
	includeAll, _ := cmd.Flags().GetBool("all")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := core.ResolveGitHubToken(flags.Token, flags.Profile)
	if err != nil {
		return err
	}

	username := args[0]

	// Parse repo argument (second arg if present)
	var repoArg string
	if len(args) > 1 {
		repoArg = args[1]
	}

	// Detect repository
	owner, repo, err := detectRepo([]string{repoArg}, flags.Repo, "Specify a repository with: clonr gh contributors journey <username> owner/repo")
	if err != nil {
		return err
	}

	if !flags.JSON {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching journey for @%s in %s/%s...\n", username, owner, repo)
	}

	// If no specific flags, default to all
	if !includeCommits && !includePRs && !includeIssues && !includeAll {
		includeAll = true
	}

	if includeAll {
		includeCommits = true
		includePRs = true
		includeIssues = true
	}

	opts := core.GetContributorJourneyOptions{
		IncludeCommits: includeCommits,
		IncludePRs:     includePRs,
		IncludeIssues:  includeIssues,
		Limit:          limit,
	}

	journey, err := core.GetContributorJourney(token, owner, repo, username, opts)
	if err != nil {
		return fmt.Errorf("failed to get contributor journey: %w", err)
	}

	if flags.JSON {
		return outputJSON(journey)
	}

	// Text output
	printContributorJourney(journey, owner, repo)

	return nil
}

func printContributorJourney(journey *core.ContributorJourney, owner, repo string) {
	// Header
	_, _ = fmt.Fprintf(os.Stdout, "\n@%s's Journey in %s/%s\n", journey.Contributor.Login, owner, repo)

	if journey.Contributor.Name != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Name: %s\n", journey.Contributor.Name)
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	// Activity summary
	_, _ = fmt.Fprintln(os.Stdout, "Activity Summary:")
	_, _ = fmt.Fprintf(os.Stdout, "  Commits:        %d\n", journey.TotalCommits)
	_, _ = fmt.Fprintf(os.Stdout, "  Pull Requests:  %d\n", journey.TotalPRs)
	_, _ = fmt.Fprintf(os.Stdout, "  Issues:         %d\n", journey.TotalIssues)

	if journey.FirstActivity != nil && journey.LastActivity != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  First Activity: %s\n", journey.FirstActivity.Format("Jan 2, 2006"))
		_, _ = fmt.Fprintf(os.Stdout, "  Last Activity:  %s\n", journey.LastActivity.Format("Jan 2, 2006"))
	}

	// Commits
	if len(journey.Commits) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nRecent Commits (%d):\n", journey.TotalCommits)

		for _, c := range journey.Commits {
			_, _ = fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", c.SHA, c.Message, formatAge(c.Date))
		}
	}

	// PRs
	if len(journey.PullRequests) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nPull Requests (%d):\n", journey.TotalPRs)

		for _, pr := range journey.PullRequests {
			stateIcon := "🟢"
			if pr.Merged {
				stateIcon = "🟣"
			} else if pr.State == "closed" {
				stateIcon = "🔴"
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %s #%-5d %s (%s)\n", stateIcon, pr.Number, pr.Title, formatAge(pr.CreatedAt))
		}
	}

	// Issues
	if len(journey.Issues) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nIssues Created (%d):\n", journey.TotalIssues)

		for _, issue := range journey.Issues {
			stateIcon := "🟢"
			if issue.State == "closed" {
				stateIcon = "🟣"
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %s #%-5d %s (%s)\n", stateIcon, issue.Number, issue.Title, formatAge(issue.CreatedAt))
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nProfile: %s\n", journey.Contributor.URL)
}
