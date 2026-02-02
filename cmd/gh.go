package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "GitHub operations for repositories",
	Long: `Interact with GitHub issues, PRs, actions, releases, and contributors.

Available Commands:
  issues        Manage GitHub issues (list, create, close)
  pr            Check pull request status
  actions       Check GitHub Actions workflow status
  release       Manage GitHub releases (create, download)
  contributors  View contributors and their activity journey

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Authentication:
  Uses GitHub token from (in priority order):
  1. --token flag
  2. --profile flag (clonr profile token)
  3. GITHUB_TOKEN environment variable
  4. GH_TOKEN environment variable
  5. Active clonr profile token
  6. gh CLI authentication`,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

// addGHCommonFlags adds flags common to all gh subcommands
func addGHCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("token", "", "GitHub token (default: auto-detect)")
	cmd.Flags().String("profile", "", "Use token from specified profile")
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().Bool("json", false, "Output as JSON")
}

// GHFlags holds common flags for all gh subcommands
type GHFlags struct {
	Token   string
	Profile string
	Repo    string
	JSON    bool
}

// extractGHFlags extracts common flags from a cobra command
func extractGHFlags(cmd *cobra.Command) GHFlags {
	token, _ := cmd.Flags().GetString("token")
	profile, _ := cmd.Flags().GetString("profile")
	repo, _ := cmd.Flags().GetString("repo")
	jsonOut, _ := cmd.Flags().GetBool("json")

	return GHFlags{
		Token:   token,
		Profile: profile,
		Repo:    repo,
		JSON:    jsonOut,
	}
}

// outputJSON encodes data as indented JSON to stdout
func outputJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// newGHLogger creates a logger appropriate for gh commands
// Uses JSON handler when JSON output is enabled, text otherwise
func newGHLogger(jsonOutput bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	if jsonOutput {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// detectRepo detects repository from args and flags
// Returns owner, repo, or error with usage hint
func detectRepo(args []string, repoFlag, usageHint string) (owner, repo string, err error) {
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	owner, repo, err = core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return "", "", fmt.Errorf("could not determine repository: %w\n\n%s", err, usageHint)
	}

	return owner, repo, nil
}
