package standalone

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Archive format constants
const (
	ArchiveVersion    = 1
	ArchiveMagic      = "CLONR-REPO"
	ArchiveExtension  = ".clonr"
	ManifestFileName  = "manifest.json"
	DefaultBufferSize = 32 * 1024 // 32KB buffer for streaming
)

// ArchiveManifest contains metadata about the archived repositories.
type ArchiveManifest struct {
	Version      int                 `json:"version"`
	CreatedAt    time.Time           `json:"created_at"`
	InstanceID   string              `json:"instance_id,omitempty"`
	Repositories []ArchivedRepo      `json:"repositories"`
	TotalSize    int64               `json:"total_size"`
	Checksum     string              `json:"checksum"` // SHA256 of unencrypted zip
	Compression  string              `json:"compression"`
	Encryption   string              `json:"encryption"`
}

// ArchivedRepo represents a repository in the archive.
type ArchivedRepo struct {
	Name        string    `json:"name"`
	URL         string    `json:"url,omitempty"`
	Path        string    `json:"path"`          // Original path
	ArchivePath string    `json:"archive_path"`  // Path inside archive
	Size        int64     `json:"size"`
	FileCount   int       `json:"file_count"`
	LastCommit  string    `json:"last_commit,omitempty"`
	ArchivedAt  time.Time `json:"archived_at"`
}

// ArchiveOptions configures the archive creation process.
type ArchiveOptions struct {
	Password       string   // Encryption password
	IncludeGitDir  bool     // Include .git directory (default: true)
	ExcludePatterns []string // Glob patterns to exclude
	CompressionLevel int    // 0-9, where 0 is store only, 9 is best compression
	InstanceID     string   // Optional instance ID for manifest
}

// DefaultArchiveOptions returns sensible defaults for archiving.
func DefaultArchiveOptions() ArchiveOptions {
	return ArchiveOptions{
		IncludeGitDir:    true,
		ExcludePatterns:  []string{".git/objects/pack/*.pack", "node_modules/**", "vendor/**", "__pycache__/**", "*.pyc", ".env", ".env.*"},
		CompressionLevel: 6, // Balanced compression
	}
}

// CreateRepoArchive creates an encrypted archive of the specified repositories.
func CreateRepoArchive(outputPath string, repoPaths []string, opts ArchiveOptions) (*ArchiveManifest, error) {
	if opts.Password == "" {
		return nil, fmt.Errorf("password is required for encrypted archive")
	}

	// Create zip in memory buffer
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// Set compression level
	zipWriter.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return newFlateWriter(w, opts.CompressionLevel)
	})

	manifest := &ArchiveManifest{
		Version:      ArchiveVersion,
		CreatedAt:    time.Now(),
		InstanceID:   opts.InstanceID,
		Repositories: make([]ArchivedRepo, 0, len(repoPaths)),
		Compression:  "deflate",
		Encryption:   "AES-256-GCM",
	}

	var totalSize int64

	for _, repoPath := range repoPaths {
		// Normalize path
		repoPath = filepath.Clean(repoPath)

		info, err := os.Stat(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", repoPath, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", repoPath)
		}

		repoName := filepath.Base(repoPath)
		archivePath := repoName + "/"

		archivedRepo := ArchivedRepo{
			Name:        repoName,
			Path:        repoPath,
			ArchivePath: archivePath,
			ArchivedAt:  time.Now(),
		}

		// Get git remote URL if available
		archivedRepo.URL = getGitRemoteURL(repoPath)
		archivedRepo.LastCommit = getLastCommitHash(repoPath)

		// Walk the repository directory
		fileCount := 0
		err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get relative path
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return err
			}

			// Skip root directory
			if relPath == "." {
				return nil
			}

			// Check exclusions
			if shouldExclude(relPath, opts.ExcludePatterns) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip .git if not included
			if !opts.IncludeGitDir && (relPath == ".git" || strings.HasPrefix(relPath, ".git/") || strings.HasPrefix(relPath, ".git\\")) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Create archive path
			archiveFilePath := filepath.ToSlash(filepath.Join(archivePath, relPath))

			if info.IsDir() {
				// Create directory entry
				_, err := zipWriter.Create(archiveFilePath + "/")
				return err
			}

			// Create file entry
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = archiveFilePath
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			// Copy file content
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			written, err := io.Copy(writer, file)
			if err != nil {
				return err
			}

			totalSize += written
			fileCount++
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to archive %s: %w", repoPath, err)
		}

		archivedRepo.Size = totalSize
		archivedRepo.FileCount = fileCount
		manifest.Repositories = append(manifest.Repositories, archivedRepo)
	}

	manifest.TotalSize = totalSize

	// Add manifest to archive
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestWriter, err := zipWriter.Create(ManifestFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest entry: %w", err)
	}
	if _, err := manifestWriter.Write(manifestData); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Close zip writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	// Get final zip data and calculate checksum
	zipData := zipBuffer.Bytes()
	checksum := sha256.Sum256(zipData)
	manifest.Checksum = fmt.Sprintf("%x", checksum)

	// Encrypt the zip data
	encrypted, err := Encrypt(zipData, opts.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt archive: %w", err)
	}

	// Write to output file with magic header
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outputFile.Close() }()

	// Write header: magic (10) + version (1) + encrypted data
	if _, err := outputFile.WriteString(ArchiveMagic); err != nil {
		return nil, fmt.Errorf("failed to write magic: %w", err)
	}
	if _, err := outputFile.Write([]byte{byte(ArchiveVersion)}); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	if _, err := outputFile.Write(encrypted); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	return manifest, nil
}

// ExtractRepoArchive extracts an encrypted archive to the specified directory.
func ExtractRepoArchive(archivePath, outputDir, password string) (*ArchiveManifest, error) {
	// Read archive file
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	// Verify magic header
	if len(data) < len(ArchiveMagic)+1 {
		return nil, fmt.Errorf("invalid archive: file too small")
	}
	if string(data[:len(ArchiveMagic)]) != ArchiveMagic {
		return nil, fmt.Errorf("invalid archive: missing magic header")
	}

	version := int(data[len(ArchiveMagic)])
	if version > ArchiveVersion {
		return nil, fmt.Errorf("unsupported archive version: %d (max supported: %d)", version, ArchiveVersion)
	}

	// Decrypt
	encryptedData := data[len(ArchiveMagic)+1:]
	zipData, err := Decrypt(encryptedData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt archive (wrong password?): %w", err)
	}

	// Open zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	// Find and read manifest
	var manifest *ArchiveManifest
	for _, f := range zipReader.File {
		if f.Name == ManifestFileName {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open manifest: %w", err)
			}
			manifestData, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest: %w", err)
			}
			manifest = &ArchiveManifest{}
			if err := json.Unmarshal(manifestData, manifest); err != nil {
				return nil, fmt.Errorf("failed to parse manifest: %w", err)
			}
			break
		}
	}

	if manifest == nil {
		return nil, fmt.Errorf("archive is missing manifest")
	}

	// Note: We don't verify checksum here because AES-GCM already provides
	// integrity checking via its authentication tag. If the data was tampered
	// with, the decryption would have failed above.

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Extract files
	for _, f := range zipReader.File {
		// Skip manifest
		if f.Name == ManifestFileName {
			continue
		}

		destPath := filepath.Join(outputDir, filepath.FromSlash(f.Name))

		// Ensure path is within output directory (prevent zip slip)
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(outputDir)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("invalid file path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", f.Name, err)
		}

		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("failed to create %s: %w", destPath, err)
		}

		_, err = io.Copy(destFile, rc)
		_ = rc.Close()
		_ = destFile.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	return manifest, nil
}

// ListArchiveContents lists the contents of an encrypted archive without extracting.
func ListArchiveContents(archivePath, password string) (*ArchiveManifest, error) {
	// Read archive file
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	// Verify magic header
	if len(data) < len(ArchiveMagic)+1 {
		return nil, fmt.Errorf("invalid archive: file too small")
	}
	if string(data[:len(ArchiveMagic)]) != ArchiveMagic {
		return nil, fmt.Errorf("invalid archive: missing magic header")
	}

	// Decrypt
	encryptedData := data[len(ArchiveMagic)+1:]
	zipData, err := Decrypt(encryptedData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt archive (wrong password?): %w", err)
	}

	// Open zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	// Find and read manifest
	for _, f := range zipReader.File {
		if f.Name == ManifestFileName {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open manifest: %w", err)
			}
			manifestData, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read manifest: %w", err)
			}
			var manifest ArchiveManifest
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				return nil, fmt.Errorf("failed to parse manifest: %w", err)
			}
			return &manifest, nil
		}
	}

	return nil, fmt.Errorf("archive is missing manifest")
}

// shouldExclude checks if a path matches any exclusion pattern.
func shouldExclude(path string, patterns []string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)

	for _, pattern := range patterns {
		// Handle directory patterns
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
			continue
		}

		// Use filepath.Match for glob patterns
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}

		// Also try matching full path
		matched, _ = filepath.Match(pattern, path)
		if matched {
			return true
		}
	}
	return false
}

// getGitRemoteURL returns the remote origin URL if available.
func getGitRemoteURL(repoPath string) string {
	configPath := filepath.Join(repoPath, ".git", "config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	// Simple parser for git config
	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[remote \"origin\"]" {
			inOrigin = true
			continue
		}
		if inOrigin && strings.HasPrefix(line, "url = ") {
			return strings.TrimPrefix(line, "url = ")
		}
		if inOrigin && strings.HasPrefix(line, "[") {
			break
		}
	}
	return ""
}

// getLastCommitHash returns the HEAD commit hash if available.
func getLastCommitHash(repoPath string) string {
	headPath := filepath.Join(repoPath, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	head := strings.TrimSpace(string(data))

	// Direct commit reference
	if len(head) == 40 {
		return head[:8] // Short hash
	}

	// Symbolic reference (ref: refs/heads/main)
	if strings.HasPrefix(head, "ref: ") {
		refPath := filepath.Join(repoPath, ".git", strings.TrimPrefix(head, "ref: "))
		refData, err := os.ReadFile(refPath)
		if err != nil {
			return ""
		}
		hash := strings.TrimSpace(string(refData))
		if len(hash) >= 8 {
			return hash[:8]
		}
	}

	return ""
}

// newFlateWriter creates a flate compressor with the specified level.
func newFlateWriter(w io.Writer, level int) (io.WriteCloser, error) {
	return flate.NewWriter(w, level)
}
