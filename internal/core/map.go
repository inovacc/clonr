package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/inovacc/clonr/internal/grpcclient"
)

// MapOptions configures the repository mapping operation
type MapOptions struct {
	DryRun   bool     // Don't actually add repos, just show what would be added
	MaxDepth int      // Maximum directory depth to scan (0 = unlimited)
	Exclude  []string // Directory names to skip (e.g., node_modules, vendor)
	JSON     bool     // Output results as JSON
	Verbose  bool     // Show verbose output
}

// MapResult contains the result of a mapping operation
type MapResult struct {
	ScannedDir   string           `json:"scanned_dir"`
	Found        []MappedRepo     `json:"found"`
	AlreadyAdded []MappedRepo     `json:"already_added"`
	Errors       []MappedRepoErr  `json:"errors,omitempty"`
	TotalFound   int              `json:"total_found"`
	TotalAdded   int              `json:"total_added"`
	TotalSkipped int              `json:"total_skipped"`
	TotalErrors  int              `json:"total_errors"`
}

// MappedRepo represents a discovered repository
type MappedRepo struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

// MappedRepoErr represents an error during mapping
type MappedRepoErr struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// DefaultExcludeDirs are directories commonly excluded from scanning
var DefaultExcludeDirs = []string{
	"node_modules",
	"vendor",
	".cache",
	".npm",
	".yarn",
	"__pycache__",
	".venv",
	"venv",
	".tox",
	"target",        // Rust/Java
	"build",         // Various build outputs
	"dist",          // Distribution folders
	".gradle",       // Gradle
	".m2",           // Maven
	"Pods",          // CocoaPods
	".pub-cache",    // Dart/Flutter
	".cargo",        // Rust cargo
	".rustup",       // Rust toolchain
	"Library",       // macOS Library folder
	"Applications",  // macOS Applications
}

// MapRepos scans a directory for Git repositories and registers them
func MapRepos(args []string) error {
	opts := MapOptions{
		Exclude: DefaultExcludeDirs,
	}

	return MapReposWithOptions(args, opts)
}

// MapReposWithOptions scans a directory with custom options
func MapReposWithOptions(args []string, opts MapOptions) error {
	rootDir := "."

	if len(args) > 0 {
		rootDir = args[0]
	}

	// Resolve to absolute path
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absRoot)
	if err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absRoot)
	}

	result := &MapResult{
		ScannedDir:   absRoot,
		Found:        make([]MappedRepo, 0),
		AlreadyAdded: make([]MappedRepo, 0),
		Errors:       make([]MappedRepoErr, 0),
	}

	var client *grpcclient.Client

	if !opts.DryRun {
		client, err = grpcclient.GetClient()
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
	}

	// Build exclude map for fast lookups
	excludeMap := make(map[string]bool)
	for _, dir := range opts.Exclude {
		excludeMap[dir] = true
	}

	rootDepth := strings.Count(absRoot, string(os.PathSeparator))

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if opts.Verbose {
				log.Printf("Error accessing %s: %v\n", path, err)
			}

			return nil // Continue walking
		}

		if !d.IsDir() {
			return nil
		}

		// Check depth limit
		if opts.MaxDepth > 0 {
			currentDepth := strings.Count(path, string(os.PathSeparator)) - rootDepth
			if currentDepth > opts.MaxDepth {
				return fs.SkipDir
			}
		}

		// Check exclusions
		if excludeMap[d.Name()] {
			if opts.Verbose {
				log.Printf("Skipping excluded directory: %s\n", path)
			}

			return fs.SkipDir
		}

		// Check if this is a .git directory
		if d.Name() == ".git" {
			repoPath := filepath.Dir(path)

			dotGit, err := dotGitCheck(path)
			if err != nil {
				result.Errors = append(result.Errors, MappedRepoErr{
					Path:  repoPath,
					Error: err.Error(),
				})
				result.TotalErrors++

				if opts.Verbose {
					log.Printf("Error checking %s: %v\n", repoPath, err)
				}

				return fs.SkipDir
			}

			repo := MappedRepo{
				Path: repoPath,
				URL:  dotGit.URL.String(),
			}

			if opts.DryRun {
				result.Found = append(result.Found, repo)
				result.TotalFound++

				if !opts.JSON {
					log.Printf("Would add: %s (%s)\n", repoPath, dotGit.URL.String())
				}

				return fs.SkipDir
			}

			// Check if already tracked
			exists, err := client.RepoExistsByURL(dotGit.URL)
			if err != nil {
				result.Errors = append(result.Errors, MappedRepoErr{
					Path:  repoPath,
					Error: err.Error(),
				})
				result.TotalErrors++

				if opts.Verbose {
					log.Printf("DB check failed for %s: %v\n", repoPath, err)
				}

				return fs.SkipDir
			}

			if exists {
				result.AlreadyAdded = append(result.AlreadyAdded, repo)
				result.TotalSkipped++

				if opts.Verbose && !opts.JSON {
					log.Printf("Already tracked: %s\n", repoPath)
				}

				return fs.SkipDir
			}

			// Add to database
			if err := client.SaveRepo(dotGit.URL, repoPath); err != nil {
				result.Errors = append(result.Errors, MappedRepoErr{
					Path:  repoPath,
					Error: err.Error(),
				})
				result.TotalErrors++

				if !opts.JSON {
					log.Printf("Failed to add %s: %v\n", repoPath, err)
				}
			} else {
				result.Found = append(result.Found, repo)
				result.TotalFound++
				result.TotalAdded++

				if !opts.JSON {
					log.Printf("Added: %s\n", repoPath)
				}
			}

			return fs.SkipDir
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error scanning directories: %w", err)
	}

	// Output results
	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	// Summary
	_, _ = fmt.Fprintln(os.Stdout)

	if opts.DryRun {
		_, _ = fmt.Fprintf(os.Stdout, "Dry run complete: %d repositories found\n", result.TotalFound)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Mapping complete: %d added, %d already tracked, %d errors\n",
			result.TotalAdded, result.TotalSkipped, result.TotalErrors)
	}

	return nil
}
