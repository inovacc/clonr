package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v82/github"
)

// Release represents a GitHub release
type Release struct {
	ID          int64          `json:"id"`
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body,omitempty"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
	CreatedAt   time.Time      `json:"created_at"`
	PublishedAt *time.Time     `json:"published_at,omitempty"`
	Author      string         `json:"author"`
	URL         string         `json:"url"`
	TarballURL  string         `json:"tarball_url,omitempty"`
	ZipballURL  string         `json:"zipball_url,omitempty"`
	Assets      []ReleaseAsset `json:"assets,omitempty"`
}

// ReleaseAsset represents a release asset
type ReleaseAsset struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Label         string    `json:"label,omitempty"`
	ContentType   string    `json:"content_type"`
	Size          int       `json:"size"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	DownloadURL   string    `json:"download_url"`
}

// ReleasesData contains releases for a repository
type ReleasesData struct {
	Repository string    `json:"repository"`
	FetchedAt  time.Time `json:"fetched_at"`
	TotalCount int       `json:"total_count"`
	Releases   []Release `json:"releases"`
}

// CreateReleaseOptions configures release creation
type CreateReleaseOptions struct {
	TagName         string   // Required: Tag name for the release
	TargetCommitish string   // Branch or commit SHA (default: default branch)
	Name            string   // Release name (default: tag name)
	Body            string   // Release notes
	Draft           bool     // Create as draft
	Prerelease      bool     // Mark as prerelease
	GenerateNotes   bool     // Auto-generate release notes
	Assets          []string // Local file paths to upload as assets
	Logger          *slog.Logger
}

// DownloadReleaseOptions configures release download
type DownloadReleaseOptions struct {
	Tag      string   // Specific tag or "latest" (default: latest)
	Patterns []string // Asset name patterns to download (glob-like)
	Dir      string   // Destination directory (default: current dir)
	Logger   *slog.Logger
}

// DownloadResult contains info about downloaded assets
type DownloadResult struct {
	Release Release          `json:"release"`
	Files   []DownloadedFile `json:"files"`
}

// DownloadedFile represents a downloaded asset
type DownloadedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ListReleasesOptions configures release listing
type ListReleasesOptions struct {
	Limit  int // Max releases to return (0 = unlimited)
	Logger *slog.Logger
}

// ListReleases lists releases for a repository
func ListReleases(token, owner, repo string, opts ListReleasesOptions) (*ReleasesData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := NewGitHubClient(ctx, token)

	listOpts := &github.ListOptions{PerPage: 100}

	var allReleases []*github.RepositoryRelease

	collected := 0

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, listOpts)
		if err != nil {
			var rateLimitErr *github.RateLimitError
			if errors.As(err, &rateLimitErr) {
				resetTime := rateLimitErr.Rate.Reset.Time
				waitDuration := time.Until(resetTime) + time.Second

				logger.Warn("rate limited, waiting",
					slog.Duration("wait", waitDuration),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitDuration):
					continue
				}
			}

			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		allReleases = append(allReleases, releases...)
		collected += len(releases)

		if opts.Limit > 0 && collected >= opts.Limit {
			if len(allReleases) > opts.Limit {
				allReleases = allReleases[:opts.Limit]
			}

			break
		}

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	return convertReleases(owner, repo, allReleases), nil
}

// GetRelease gets a specific release by tag
func GetRelease(token, owner, repo, tag string) (*Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := NewGitHubClient(ctx, token)

	var release *github.RepositoryRelease

	var err error

	if tag == "latest" || tag == "" {
		release, _, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
	} else {
		release, _, err = client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return convertRelease(release), nil
}

// CreateRelease creates a new release
func CreateRelease(token, owner, repo string, opts CreateReleaseOptions) (*Release, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if opts.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Longer for asset uploads
	defer cancel()

	client := NewGitHubClient(ctx, token)

	// Prepare a release request
	releaseReq := &github.RepositoryRelease{
		TagName:              github.String(opts.TagName),
		Draft:                github.Bool(opts.Draft),
		Prerelease:           github.Bool(opts.Prerelease),
		GenerateReleaseNotes: github.Bool(opts.GenerateNotes),
	}

	if opts.TargetCommitish != "" {
		releaseReq.TargetCommitish = github.String(opts.TargetCommitish)
	}

	if opts.Name != "" {
		releaseReq.Name = github.String(opts.Name)
	} else {
		releaseReq.Name = github.String(opts.TagName)
	}

	if opts.Body != "" {
		releaseReq.Body = github.String(opts.Body)
	}

	logger.Debug("creating release",
		slog.String("owner", owner),
		slog.String("repo", repo),
		slog.String("tag", opts.TagName),
	)

	// Create the release
	release, _, err := client.Repositories.CreateRelease(ctx, owner, repo, releaseReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	result := convertRelease(release)

	// Upload assets if specified
	if len(opts.Assets) > 0 {
		for _, assetPath := range opts.Assets {
			asset, err := uploadAsset(ctx, client, owner, repo, release.GetID(), assetPath, logger)
			if err != nil {
				logger.Warn("failed to upload asset",
					slog.String("path", assetPath),
					slog.String("error", err.Error()),
				)

				continue
			}

			result.Assets = append(result.Assets, *asset)
		}
	}

	return result, nil
}

// DownloadRelease downloads release assets
func DownloadRelease(token, owner, repo string, opts DownloadReleaseOptions) (*DownloadResult, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Set defaults
	if opts.Tag == "" {
		opts.Tag = "latest"
	}

	if opts.Dir == "" {
		opts.Dir = "."
	}

	// Get the release
	release, err := GetRelease(token, owner, repo, opts.Tag)
	if err != nil {
		return nil, err
	}

	// Create destination directory
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	result := &DownloadResult{
		Release: *release,
		Files:   make([]DownloadedFile, 0),
	}

	// Compile patterns
	var patterns []*regexp.Regexp

	for _, p := range opts.Patterns {
		// Convert glob-like pattern to regex
		regexPattern := globToRegex(p)

		re, err := regexp.Compile(regexPattern)
		if err != nil {
			logger.Warn("invalid pattern",
				slog.String("pattern", p),
				slog.String("error", err.Error()),
			)

			continue
		}

		patterns = append(patterns, re)
	}

	// Download matching assets
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tc := NewOAuth2HTTPClient(ctx, token)

	for _, asset := range release.Assets {
		// Check if asset matches patterns (or download all if no patterns)
		if len(patterns) > 0 {
			matched := false

			for _, re := range patterns {
				if re.MatchString(asset.Name) {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}
		}

		logger.Info("downloading asset",
			slog.String("name", asset.Name),
			slog.Int("size", asset.Size),
		)

		destPath := filepath.Join(opts.Dir, asset.Name)

		size, err := downloadFile(ctx, tc.Transport, asset.DownloadURL, destPath)
		if err != nil {
			logger.Warn("failed to download asset",
				slog.String("name", asset.Name),
				slog.String("error", err.Error()),
			)

			continue
		}

		result.Files = append(result.Files, DownloadedFile{
			Name: asset.Name,
			Path: destPath,
			Size: size,
		})
	}

	return result, nil
}

func uploadAsset(ctx context.Context, client *github.Client, owner, repo string, releaseID int64, assetPath string, logger *slog.Logger) (*ReleaseAsset, error) {
	file, err := os.Open(assetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	name := filepath.Base(assetPath)
	logger.Info("uploading asset",
		slog.String("name", name),
		slog.Int64("size", stat.Size()),
	)

	uploadOpts := &github.UploadOptions{
		Name: name,
	}

	asset, _, err := client.Repositories.UploadReleaseAsset(ctx, owner, repo, releaseID, uploadOpts, file)
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	return &ReleaseAsset{
		ID:            asset.GetID(),
		Name:          asset.GetName(),
		Label:         asset.GetLabel(),
		ContentType:   asset.GetContentType(),
		Size:          asset.GetSize(),
		DownloadCount: asset.GetDownloadCount(),
		CreatedAt:     asset.GetCreatedAt().Time,
		UpdatedAt:     asset.GetUpdatedAt().Time,
		DownloadURL:   asset.GetBrowserDownloadURL(),
	}, nil
}

func downloadFile(ctx context.Context, transport http.RoundTripper, url, destPath string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to download: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}

	defer func() {
		_ = out.Close()
	}()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	return written, nil
}

func convertReleases(owner, repo string, releases []*github.RepositoryRelease) *ReleasesData {
	data := &ReleasesData{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		FetchedAt:  time.Now(),
		TotalCount: len(releases),
		Releases:   make([]Release, 0, len(releases)),
	}

	for _, r := range releases {
		data.Releases = append(data.Releases, *convertRelease(r))
	}

	return data
}

func convertRelease(r *github.RepositoryRelease) *Release {
	release := &Release{
		ID:         r.GetID(),
		TagName:    r.GetTagName(),
		Name:       r.GetName(),
		Body:       r.GetBody(),
		Draft:      r.GetDraft(),
		Prerelease: r.GetPrerelease(),
		CreatedAt:  r.GetCreatedAt().Time,
		Author:     r.GetAuthor().GetLogin(),
		URL:        r.GetHTMLURL(),
		TarballURL: r.GetTarballURL(),
		ZipballURL: r.GetZipballURL(),
	}

	if !r.GetPublishedAt().IsZero() {
		t := r.GetPublishedAt().Time
		release.PublishedAt = &t
	}

	// Convert assets
	for _, a := range r.Assets {
		release.Assets = append(release.Assets, ReleaseAsset{
			ID:            a.GetID(),
			Name:          a.GetName(),
			Label:         a.GetLabel(),
			ContentType:   a.GetContentType(),
			Size:          a.GetSize(),
			DownloadCount: a.GetDownloadCount(),
			CreatedAt:     a.GetCreatedAt().Time,
			UpdatedAt:     a.GetUpdatedAt().Time,
			DownloadURL:   a.GetBrowserDownloadURL(),
		})
	}

	return release
}

// globToRegex converts a glob-like pattern to a regex pattern
func globToRegex(glob string) string {
	var sb strings.Builder
	sb.WriteString("^")

	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '*':
			sb.WriteString(".*")
		case '?':
			sb.WriteString(".")
		case '.', '+', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}

	sb.WriteString("$")

	return sb.String()
}
