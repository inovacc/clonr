package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [path]",
	Short: "Show git diff for a repository",
	Long: `Display git diff for a repository.

If no path is provided, shows an interactive repository selector.

Examples:
  clonr diff                    # Select repo, show diff
  clonr diff /path/to/repo      # Show diff for specific repo
  clonr diff --staged           # Show only staged changes
  clonr diff --stat             # Show diffstat summary
  clonr diff --name-only        # Show only changed file names
  clonr diff --json             # Output as JSON`,
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().Bool("staged", false, "Show staged changes only")
	diffCmd.Flags().Bool("cached", false, "Alias for --staged")
	diffCmd.Flags().Bool("stat", false, "Show diffstat summary")
	diffCmd.Flags().Bool("name-only", false, "Show only file names")
	diffCmd.Flags().Bool("json", false, "Output as JSON (non-interactive)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	staged, _ := cmd.Flags().GetBool("staged")
	cached, _ := cmd.Flags().GetBool("cached")
	stat, _ := cmd.Flags().GetBool("stat")
	nameOnly, _ := cmd.Flags().GetBool("name-only")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// --cached is an alias for --staged
	if cached {
		staged = true
	}

	opts := core.DiffOptions{
		Staged:   staged,
		Stat:     stat,
		NameOnly: nameOnly,
	}

	var repoPath string

	var repoURL string

	if len(args) > 0 {
		// Path provided as argument
		repoPath = args[0]
	} else {
		// Interactive repository selection
		m, err := cli.NewRepoList(false)
		if err != nil {
			return err
		}

		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		repoModel := finalModel.(cli.RepoListModel)
		selected := repoModel.GetSelectedRepo()

		if selected == nil {
			// User cancelled selection
			return nil
		}

		repoPath = selected.Path
		repoURL = selected.URL
	}

	result, err := core.GetDiff(repoPath, opts)
	if err != nil {
		return err
	}

	// Set repo URL if we have it
	if repoURL != "" {
		result.RepoURL = repoURL
	}

	if jsonOutput {
		return outputDiffJSON(result)
	}

	return outputDiffText(result, opts)
}

func outputDiffJSON(result *core.DiffResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(result)
}

func outputDiffText(result *core.DiffResult, opts core.DiffOptions) error {
	_, _ = fmt.Fprintf(os.Stdout, "Repository: %s\n", result.RepoPath)

	if result.RepoURL != "" {
		_, _ = fmt.Fprintf(os.Stdout, "URL: %s\n", result.RepoURL)
	}

	_, _ = fmt.Fprintln(os.Stdout)

	if !result.HasChanges {
		changeType := "unstaged"
		if opts.Staged {
			changeType = "staged"
		}

		_, _ = fmt.Fprintf(os.Stdout, "No %s changes\n", changeType)

		return nil
	}

	if opts.NameOnly {
		_, _ = fmt.Fprintf(os.Stdout, "Changed files (%d):\n", len(result.Files))

		for _, file := range result.Files {
			_, _ = fmt.Fprintf(os.Stdout, "  %s\n", file)
		}

		return nil
	}

	if opts.Stat {
		_, _ = fmt.Fprintln(os.Stdout, result.Stats)

		return nil
	}

	// Full diff output
	_, _ = fmt.Fprint(os.Stdout, result.Diff)

	return nil
}
