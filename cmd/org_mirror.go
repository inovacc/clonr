package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var orgMirrorCmd = &cobra.Command{
	Use:   "mirror <org_name>",
	Short: "Mirror all repositories from a GitHub organization",
	Long: `Mirror all repositories from a GitHub organization.

This command will:
  1. Fetch all repositories from the specified GitHub organization
  2. Clone repositories that don't exist locally
  3. Update (git pull) repositories that already exist
  4. Organize repositories under <clone_dir>/<org_name>/<repo_name>

Authentication:
  Token is automatically detected from (in order):
  - --token flag
  - GITHUB_TOKEN environment variable
  - GH_TOKEN environment variable
  - gh CLI (if authenticated via 'gh auth login')

Dirty Repository Handling:
  When updating repositories with uncommitted changes, use --dirty-strategy:
  - skip:  Skip the repository (default)
  - stash: Stash changes, pull, then unstash
  - reset: Reset to clean state (WARNING: destroys local changes)

Examples:
  # Basic mirror
  clonr org mirror kubernetes

  # Dry run to preview what will be done
  clonr org mirror kubernetes --dry-run

  # Custom token and parallel operations
  clonr org mirror myorg --token ghp_xxx --parallel 5

  # Filter specific repos (regex)
  clonr org mirror myorg --filter "^api-"

  # Include archived repos
  clonr org mirror myorg --skip-archived=false

  # Handle dirty repos by stashing changes
  clonr org mirror myorg --dirty-strategy=stash

  # JSON log output for scripting
  clonr org mirror myorg --json --log-level=debug`,
	Args: cobra.ExactArgs(1),
	RunE: runMirror,
}

func runMirror(cmd *cobra.Command, args []string) error {
	orgName := args[0]

	// Validate organization name
	if err := core.ValidateOrgName(orgName); err != nil {
		return err
	}

	// Get flags
	token, _ := cmd.Flags().GetString("token")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	skipArchived, _ := cmd.Flags().GetBool("skip-archived")
	publicOnly, _ := cmd.Flags().GetBool("public-only")
	filterStr, _ := cmd.Flags().GetString("filter")
	parallel, _ := cmd.Flags().GetInt("parallel")

	// New flags
	dirtyStrategy, _ := cmd.Flags().GetString("dirty-strategy")
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	networkRetries, _ := cmd.Flags().GetInt("network-retries")
	logLevel, _ := cmd.Flags().GetString("log-level")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	shallow, _ := cmd.Flags().GetBool("shallow")

	// Validate parallel flag
	if parallel < 1 || parallel > 10 {
		return fmt.Errorf("parallel must be between 1 and 10")
	}

	// Validate max-retries flag
	if maxRetries < 1 || maxRetries > 20 {
		return fmt.Errorf("max-retries must be between 1 and 20")
	}

	// Validate network-retries flag
	if networkRetries < 1 || networkRetries > 10 {
		return fmt.Errorf("network-retries must be between 1 and 10")
	}

	// Setup logger
	logger := setupMirrorLogger(logLevel, jsonOutput)

	// Resolve token from multiple sources
	token, tokenSource, err := core.ResolveGitHubToken(token)
	if err != nil {
		return err
	}

	logger.Debug("token resolved",
		slog.String("source", string(tokenSource)),
	)

	// Parse filter regex if provided
	var filterRegex *regexp.Regexp
	if filterStr != "" {
		filterRegex, err = regexp.Compile(filterStr)
		if err != nil {
			return fmt.Errorf("invalid filter regex: %w", err)
		}
	}

	// Build rate limit config
	rateCfg := core.RateLimitConfig{
		MaxRetries:        maxRetries,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        2 * time.Minute,
		BackoffMultiplier: 2.0,
	}

	// Build mirror options
	opts := core.MirrorOptions{
		SkipArchived:    skipArchived,
		PublicOnly:      publicOnly,
		Filter:          filterRegex,
		Parallel:        parallel,
		DirtyStrategy:   core.ParseDirtyStrategy(dirtyStrategy),
		RateLimitConfig: rateCfg,
		NetworkRetries:  networkRetries,
		Shallow:         shallow,
		Logger:          logger,
	}

	logger.Info("starting mirror operation",
		slog.String("org", orgName),
		slog.Int("parallel", parallel),
		slog.String("dirty_strategy", dirtyStrategy),
	)

	// Call core logic to prepare mirror operation
	fmt.Printf("Fetching repositories from organization '%s'...\n", orgName)
	mirrorPlan, err := core.PrepareMirror(orgName, token, opts)
	if err != nil {
		return fmt.Errorf("failed to prepare mirror: %w", err)
	}

	if len(mirrorPlan.Repos) == 0 {
		logger.Warn("no repositories found to mirror", slog.String("org", orgName))
		fmt.Println("\nNo repositories found to mirror.")
		return nil
	}

	if dryRun {
		// Print what would be done and exit
		core.PrintDryRunPlan(mirrorPlan)
		if jsonOutput {
			core.LogDryRunPlan(mirrorPlan, logger)
		}
		return nil
	}

	// Check for --no-tui flag
	noTUI, _ := cmd.Flags().GetBool("no-tui")

	if noTUI {
		// Batch mode (no TUI)
		fmt.Printf("\nMirroring %d repositories (parallel: %d)...\n\n", len(mirrorPlan.Repos), parallel)

		batchOpts := core.MirrorBatchOptions{
			Plan:   mirrorPlan,
			Logger: logger,
		}

		result, err := core.ExecuteMirrorBatch(batchOpts)
		if err != nil {
			return fmt.Errorf("mirror failed: %w", err)
		}

		core.PrintBatchSummary(result)
		if jsonOutput {
			core.LogMirrorSummary(result.Results, logger)
		}

		if result.Failed > 0 {
			return fmt.Errorf("%d repositories failed to mirror", result.Failed)
		}

		return nil
	}

	// Launch TUI
	m := cli.NewMirrorModel(mirrorPlan)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("UI error: %w", err)
	}

	// Display summary
	mirrorModel := finalModel.(*cli.MirrorModel)
	if mirrorModel.Error() != nil {
		return mirrorModel.Error()
	}

	core.PrintMirrorSummary(mirrorModel.Results())
	if jsonOutput {
		core.LogMirrorSummary(mirrorModel.Results(), logger)
	}

	return nil
}

// setupMirrorLogger creates a configured slog.Logger
func setupMirrorLogger(levelStr string, jsonOutput bool) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// addMirrorFlags adds the common mirror flags to a command
func addMirrorFlags(cmd *cobra.Command) {
	// Authentication
	cmd.Flags().String("token", "", "GitHub personal access token (overrides GITHUB_TOKEN env var)")

	// Operation mode
	cmd.Flags().Bool("dry-run", false, "Preview operations without executing")
	cmd.Flags().Bool("no-tui", false, "Run without interactive TUI (for scripts/CI)")
	cmd.Flags().Bool("shallow", false, "Shallow clone (--depth 1) for faster cloning")

	// Filtering
	cmd.Flags().Bool("skip-archived", true, "Skip archived repositories")
	cmd.Flags().String("filter", "", "Regex pattern to filter repository names")
	cmd.Flags().Bool("public-only", false, "Only mirror public repositories")

	// Performance
	cmd.Flags().Int("parallel", 3, "Number of concurrent operations (1-10)")

	// Error recovery
	cmd.Flags().String("dirty-strategy", "skip", "Strategy for dirty repos: skip, stash, reset")
	cmd.Flags().Int("max-retries", 5, "Max GitHub API retry attempts (1-20)")
	cmd.Flags().Int("network-retries", 3, "Max git network retry attempts (1-10)")

	// Logging
	cmd.Flags().String("log-level", "info", "Log level: debug, info, warn, error")
	cmd.Flags().Bool("json", false, "Output logs in JSON format")
}

func init() {
	orgCmd.AddCommand(orgMirrorCmd)
	addMirrorFlags(orgMirrorCmd)
}
