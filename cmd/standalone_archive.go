package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/standalone"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	archiveOutput      string
	archiveNoGit       bool
	archiveCompression int
	archiveExclude     []string
	archiveAll         bool
	archiveFavorites   bool
	archiveWorkspace   string
)

var standaloneArchiveCmd = &cobra.Command{
	Use:   "archive [repo-paths...]",
	Short: "Create encrypted archive of repositories",
	Long: `Create an encrypted, compressed archive of one or more repositories.

The archive is encrypted with AES-256-GCM and compressed with DEFLATE.
This is useful for:
  - Secure backup of repositories
  - Transferring repositories between machines
  - Syncing repositories via the standalone sync feature

By default, the .git directory is included so you can restore the full
repository with all history. Use --no-git to exclude it.

Examples:
  # Archive specific repositories
  clonr standalone archive /path/to/repo1 /path/to/repo2 -o backup.clonr

  # Archive all managed repositories
  clonr standalone archive --all -o all-repos.clonr

  # Archive only favorite repositories
  clonr standalone archive --favorites -o favorites.clonr

  # Archive repositories in a workspace
  clonr standalone archive --workspace work -o work-repos.clonr

  # Archive without .git directory (smaller but no history)
  clonr standalone archive /path/to/repo --no-git -o backup.clonr

  # Custom compression level (0=store, 9=best)
  clonr standalone archive /path/to/repo --compression 9 -o backup.clonr`,
	RunE: runStandaloneArchive,
}

func init() {
	standaloneCmd.AddCommand(standaloneArchiveCmd)

	standaloneArchiveCmd.Flags().StringVarP(&archiveOutput, "output", "o", "", "Output file path (required)")
	standaloneArchiveCmd.Flags().BoolVar(&archiveNoGit, "no-git", false, "Exclude .git directory")
	standaloneArchiveCmd.Flags().IntVar(&archiveCompression, "compression", 6, "Compression level (0-9)")
	standaloneArchiveCmd.Flags().StringSliceVar(&archiveExclude, "exclude", nil, "Additional patterns to exclude")
	standaloneArchiveCmd.Flags().BoolVar(&archiveAll, "all", false, "Archive all managed repositories")
	standaloneArchiveCmd.Flags().BoolVar(&archiveFavorites, "favorites", false, "Archive only favorite repositories")
	standaloneArchiveCmd.Flags().StringVarP(&archiveWorkspace, "workspace", "w", "", "Archive repositories in workspace")

	_ = standaloneArchiveCmd.MarkFlagRequired("output")
}

func runStandaloneArchive(_ *cobra.Command, args []string) error {
	var repoPaths []string

	// Determine which repositories to archive
	if archiveAll || archiveFavorites || archiveWorkspace != "" {
		client, err := grpc.GetClient()
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}

		repos, err := client.GetRepos(archiveWorkspace, archiveFavorites)
		if err != nil {
			return fmt.Errorf("failed to get repositories: %w", err)
		}

		for _, repo := range repos {
			if repo.Path != "" {
				repoPaths = append(repoPaths, repo.Path)
			}
		}

		if len(repoPaths) == 0 {
			return fmt.Errorf("no repositories found matching criteria")
		}
	} else if len(args) > 0 {
		repoPaths = args
	} else {
		return fmt.Errorf("specify repository paths or use --all/--favorites/--workspace")
	}

	// Verify paths exist
	for _, path := range repoPaths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("invalid path %s: %w", path, err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}

	// Get password
	password, err := readArchivePassword("Enter encryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Confirm password
	confirm, err := readArchivePassword("Confirm password: ")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}

	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	// Build options
	opts := standalone.DefaultArchiveOptions()
	opts.Password = password
	opts.IncludeGitDir = !archiveNoGit
	opts.CompressionLevel = archiveCompression

	if len(archiveExclude) > 0 {
		opts.ExcludePatterns = append(opts.ExcludePatterns, archiveExclude...)
	}

	// Ensure output has correct extension
	outputPath := archiveOutput
	if !strings.HasSuffix(outputPath, standalone.ArchiveExtension) {
		outputPath += standalone.ArchiveExtension
	}

	_, _ = fmt.Fprintf(os.Stderr, "Creating archive with %d repositories...\n", len(repoPaths))

	startTime := time.Now()

	manifest, err := standalone.CreateRepoArchive(outputPath, repoPaths, opts)
	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	duration := time.Since(startTime)

	// Print summary
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintf(os.Stderr, "Archive created: %s\n", outputPath)
	_, _ = fmt.Fprintf(os.Stderr, "  Repositories: %d\n", len(manifest.Repositories))
	_, _ = fmt.Fprintf(os.Stderr, "  Total size: %s (uncompressed)\n", formatBytes(manifest.TotalSize))
	_, _ = fmt.Fprintf(os.Stderr, "  Compression: %s\n", manifest.Compression)
	_, _ = fmt.Fprintf(os.Stderr, "  Encryption: %s\n", manifest.Encryption)
	_, _ = fmt.Fprintf(os.Stderr, "  Duration: %s\n", duration.Round(time.Millisecond))

	// Get final file size
	if info, err := os.Stat(outputPath); err == nil {
		ratio := float64(info.Size()) / float64(manifest.TotalSize) * 100
		_, _ = fmt.Fprintf(os.Stderr, "  Archive size: %s (%.1f%% of original)\n", formatBytes(info.Size()), ratio)
	}

	_, _ = fmt.Fprintln(os.Stderr)

	_, _ = fmt.Fprintln(os.Stderr, "Archived repositories:")
	for _, repo := range manifest.Repositories {
		_, _ = fmt.Fprintf(os.Stderr, "  - %s (%d files, %s)\n", repo.Name, repo.FileCount, formatBytes(repo.Size))
	}

	return nil
}

// readArchivePassword reads a password from the terminal without echoing
func readArchivePassword(prompt string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, prompt)

	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(os.Stderr)

		if err != nil {
			return "", err
		}

		return string(password), nil
	}

	// Fallback for non-terminal
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	return "", fmt.Errorf("failed to read password")
}
