package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/spf13/cobra"
)

var (
	extractOutput string
	extractList   bool
)

var standaloneExtractCmd = &cobra.Command{
	Use:   "extract <archive-path>",
	Short: "Extract encrypted repository archive",
	Long: `Extract repositories from an encrypted archive.

The archive must have been created with 'clonr standalone archive'.
You will be prompted for the password used to encrypt the archive.

Examples:
  # Extract to current directory
  clonr standalone extract backup.clonr

  # Extract to specific directory
  clonr standalone extract backup.clonr -o /path/to/output

  # List archive contents without extracting
  clonr standalone extract backup.clonr --list`,
	Args: cobra.ExactArgs(1),
	RunE: runStandaloneExtract,
}

func init() {
	standaloneCmd.AddCommand(standaloneExtractCmd)

	standaloneExtractCmd.Flags().StringVarP(&extractOutput, "output", "o", ".", "Output directory")
	standaloneExtractCmd.Flags().BoolVar(&extractList, "list", false, "List contents without extracting")
}

func runStandaloneExtract(_ *cobra.Command, args []string) error {
	archivePath := args[0]

	// Verify archive exists
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("archive not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not an archive", archivePath)
	}

	// Get password
	password, err := readArchivePassword("Enter decryption password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if extractList {
		// List mode
		manifest, err := standalone.ListArchiveContents(archivePath, password)
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Archive: %s\n", archivePath)
		_, _ = fmt.Fprintf(os.Stdout, "Created: %s\n", manifest.CreatedAt.Format(time.RFC3339))
		_, _ = fmt.Fprintf(os.Stdout, "Version: %d\n", manifest.Version)
		_, _ = fmt.Fprintf(os.Stdout, "Total size: %s (uncompressed)\n", formatBytes(manifest.TotalSize))
		_, _ = fmt.Fprintf(os.Stdout, "Checksum: %s\n", manifest.Checksum[:16]+"...")
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintf(os.Stdout, "Repositories (%d):\n", len(manifest.Repositories))

		for _, repo := range manifest.Repositories {
			_, _ = fmt.Fprintf(os.Stdout, "\n  %s\n", repo.Name)
			if repo.URL != "" {
				_, _ = fmt.Fprintf(os.Stdout, "    URL: %s\n", repo.URL)
			}

			_, _ = fmt.Fprintf(os.Stdout, "    Original path: %s\n", repo.Path)
			_, _ = fmt.Fprintf(os.Stdout, "    Files: %d\n", repo.FileCount)

			_, _ = fmt.Fprintf(os.Stdout, "    Size: %s\n", formatBytes(repo.Size))
			if repo.LastCommit != "" {
				_, _ = fmt.Fprintf(os.Stdout, "    Last commit: %s\n", repo.LastCommit)
			}

			_, _ = fmt.Fprintf(os.Stdout, "    Archived: %s\n", repo.ArchivedAt.Format(time.RFC3339))
		}

		return nil
	}

	// Extract mode
	_, _ = fmt.Fprintf(os.Stderr, "Extracting archive to %s...\n", extractOutput)

	startTime := time.Now()

	manifest, err := standalone.ExtractRepoArchive(archivePath, extractOutput, password)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	duration := time.Since(startTime)

	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintf(os.Stderr, "Extraction complete!\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Output: %s\n", extractOutput)
	_, _ = fmt.Fprintf(os.Stderr, "  Repositories: %d\n", len(manifest.Repositories))
	_, _ = fmt.Fprintf(os.Stderr, "  Duration: %s\n", duration.Round(time.Millisecond))
	_, _ = fmt.Fprintln(os.Stderr)

	_, _ = fmt.Fprintln(os.Stderr, "Extracted repositories:")
	for _, repo := range manifest.Repositories {
		_, _ = fmt.Fprintf(os.Stderr, "  - %s\n", repo.Name)
	}

	return nil
}
