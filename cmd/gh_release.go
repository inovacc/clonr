package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage GitHub releases",
	Long: `Create and download GitHub releases.

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Examples:
  clonr gh release list                         # List releases
  clonr gh release create v1.0.0                # Create a release
  clonr gh release download                     # Download latest release assets`,
}

var releaseListCmd = &cobra.Command{
	Use:   "list [owner/repo]",
	Short: "List releases for a repository",
	Long: `List GitHub releases for a repository.

Examples:
  clonr gh release list                    # List releases
  clonr gh release list --limit 5          # Limit results
  clonr gh release list owner/repo         # Specify repo`,
	RunE: runReleaseList,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create <tag> [owner/repo]",
	Short: "Create a new release",
	Long: `Create a new GitHub release.

The tag is required and will be created if it doesn't exist.

Examples:
  clonr gh release create v1.0.0
  clonr gh release create v1.0.0 --title "Release 1.0.0"
  clonr gh release create v1.0.0 --notes "Release notes here"
  clonr gh release create v1.0.0 --draft
  clonr gh release create v1.0.0 --prerelease
  clonr gh release create v1.0.0 --generate-notes
  clonr gh release create v1.0.0 --assets dist/*.tar.gz`,
	RunE: runReleaseCreate,
}

var releaseDownloadCmd = &cobra.Command{
	Use:   "download [owner/repo]",
	Short: "Download release assets",
	Long: `Download assets from a GitHub release.

By default, downloads all assets from the latest release.

Examples:
  clonr gh release download                     # Download all from latest
  clonr gh release download --tag v1.0.0        # Specific release
  clonr gh release download --pattern "*.tar.gz" # Filter assets
  clonr gh release download --dir ./downloads   # Specify directory`,
	RunE: runReleaseDownload,
}

func init() {
	ghCmd.AddCommand(releaseCmd)
	releaseCmd.AddCommand(releaseListCmd)
	releaseCmd.AddCommand(releaseCreateCmd)
	releaseCmd.AddCommand(releaseDownloadCmd)

	// List flags
	addGHCommonFlags(releaseListCmd)
	releaseListCmd.Flags().Int("limit", 10, "Maximum number of releases to list")

	// Create flags
	addGHCommonFlags(releaseCreateCmd)
	releaseCreateCmd.Flags().String("title", "", "Release title (default: tag name)")
	releaseCreateCmd.Flags().String("notes", "", "Release notes")
	releaseCreateCmd.Flags().String("notes-file", "", "Read release notes from file")
	releaseCreateCmd.Flags().String("target", "", "Target branch or commit SHA")
	releaseCreateCmd.Flags().Bool("draft", false, "Create as draft release")
	releaseCreateCmd.Flags().Bool("prerelease", false, "Mark as prerelease")
	releaseCreateCmd.Flags().Bool("generate-notes", false, "Auto-generate release notes")
	releaseCreateCmd.Flags().StringSlice("assets", nil, "Files to upload as release assets")

	// Download flags
	addGHCommonFlags(releaseDownloadCmd)
	releaseDownloadCmd.Flags().String("tag", "latest", "Release tag to download (default: latest)")
	releaseDownloadCmd.Flags().StringSlice("pattern", nil, "Asset name patterns to download (glob-like)")
	releaseDownloadCmd.Flags().String("dir", ".", "Destination directory")
}

func runReleaseList(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")

	// Get repo argument if provided
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh release list owner/repo", err)
	}

	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching releases for %s/%s...\n", owner, repo)
	}

	opts := core.ListReleasesOptions{
		Limit: limit,
	}

	data, err := core.ListReleases(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(data)
	}

	// Text output
	if len(data.Releases) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No releases found in %s/%s\n", owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nReleases for %s/%s\n\n", owner, repo)

	for _, release := range data.Releases {
		printReleaseSummary(&release)
	}

	return nil
}

func runReleaseCreate(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	title, _ := cmd.Flags().GetString("title")
	notes, _ := cmd.Flags().GetString("notes")
	notesFile, _ := cmd.Flags().GetString("notes-file")
	target, _ := cmd.Flags().GetString("target")
	draft, _ := cmd.Flags().GetBool("draft")
	prerelease, _ := cmd.Flags().GetBool("prerelease")
	generateNotes, _ := cmd.Flags().GetBool("generate-notes")
	assets, _ := cmd.Flags().GetStringSlice("assets")

	// Parse arguments - first is tag, second is optional repo
	var tag, repoArg string

	for i, arg := range args {
		if i == 0 {
			tag = arg
		} else {
			repoArg = arg
		}
	}

	if tag == "" {
		return fmt.Errorf("tag is required\n\nUsage: clonr gh release create <tag>")
	}

	// Read notes from file if specified
	if notesFile != "" {
		content, err := os.ReadFile(notesFile)
		if err != nil {
			return fmt.Errorf("failed to read notes file: %w", err)
		}

		notes = string(content)
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh release create <tag> owner/repo", err)
	}

	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Creating release %s in %s/%s...\n", tag, owner, repo)
	}

	opts := core.CreateReleaseOptions{
		TagName:         tag,
		TargetCommitish: target,
		Name:            title,
		Body:            notes,
		Draft:           draft,
		Prerelease:      prerelease,
		GenerateNotes:   generateNotes,
		Assets:          assets,
	}

	release, err := core.CreateRelease(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(release)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Created release %s\n", release.TagName)
	_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", release.Name)

	if release.Draft {
		_, _ = fmt.Fprintln(os.Stdout, "  Status: draft")
	} else if release.Prerelease {
		_, _ = fmt.Fprintln(os.Stdout, "  Status: prerelease")
	}

	if len(release.Assets) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  Assets: %d uploaded\n", len(release.Assets))
	}

	_, _ = fmt.Fprintf(os.Stdout, "  URL: %s\n", release.URL)

	return nil
}

func runReleaseDownload(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	tag, _ := cmd.Flags().GetString("tag")
	patterns, _ := cmd.Flags().GetStringSlice("pattern")
	dir, _ := cmd.Flags().GetString("dir")

	// Get repo argument if provided
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh release download owner/repo", err)
	}

	tagDisplay := tag
	if tag == "latest" || tag == "" {
		tagDisplay = "latest release"
	}

	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Downloading %s from %s/%s...\n", tagDisplay, owner, repo)
	}

	opts := core.DownloadReleaseOptions{
		Tag:      tag,
		Patterns: patterns,
		Dir:      dir,
	}

	result, err := core.DownloadRelease(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to download release: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	// Text output
	if len(result.Files) == 0 {
		if len(result.Release.Assets) == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Release %s has no assets to download\n", result.Release.TagName)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "No assets matched the specified patterns\n")
		}

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Downloaded %d file(s) from %s\n\n", len(result.Files), result.Release.TagName)

	for _, f := range result.Files {
		_, _ = fmt.Fprintf(os.Stdout, "  %s (%s)\n", f.Name, formatFileSize(f.Size))
	}

	return nil
}

func printReleaseSummary(release *core.Release) {
	// Status indicators
	statusParts := []string{}
	if release.Draft {
		statusParts = append(statusParts, "draft")
	}

	if release.Prerelease {
		statusParts = append(statusParts, "prerelease")
	}

	statusStr := ""
	if len(statusParts) > 0 {
		statusStr = fmt.Sprintf(" [%s]", strings.Join(statusParts, ", "))
	}

	// Name/tag
	name := release.Name
	if name == "" {
		name = release.TagName
	}

	_, _ = fmt.Fprintf(os.Stdout, "📦 %s%s\n", name, statusStr)

	// Tag and date
	publishedStr := "draft"

	if release.PublishedAt != nil {
		publishedStr = formatAge(*release.PublishedAt)
	}

	_, _ = fmt.Fprintf(os.Stdout, "   %s · %s by @%s\n", release.TagName, publishedStr, release.Author)

	// Assets
	if len(release.Assets) > 0 {
		totalSize := 0
		for _, a := range release.Assets {
			totalSize += a.Size
		}

		_, _ = fmt.Fprintf(os.Stdout, "   %d assets (%s)\n", len(release.Assets), formatFileSize(int64(totalSize)))
	}

	_, _ = fmt.Fprintln(os.Stdout)
}

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
