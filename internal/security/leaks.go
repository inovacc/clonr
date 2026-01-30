// Package security provides security scanning capabilities
package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zricethezav/gitleaks/v8/detect"
	"github.com/zricethezav/gitleaks/v8/report"
	"github.com/zricethezav/gitleaks/v8/sources"
)

// LeakScanner provides secret detection capabilities
type LeakScanner struct {
	detector *detect.Detector
}

// ScanResult contains the results of a leak scan
type ScanResult struct {
	Findings    []Finding
	HasLeaks    bool
	ScannedPath string
}

// Finding represents a detected secret
type Finding struct {
	RuleID      string
	Description string
	File        string
	Line        int
	Secret      string // Redacted
	Match       string
	Commit      string
	Author      string
	Date        string
}

// NewLeakScanner creates a new leak scanner with default gitleaks rules
func NewLeakScanner() (*LeakScanner, error) {
	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load gitleaks config: %w", err)
	}

	detector.Redact = 80 // Redact 80% of the secret

	return &LeakScanner{
		detector: detector,
	}, nil
}

// ScanDirectory scans a directory for secrets
func (s *LeakScanner) ScanDirectory(ctx context.Context, path string) (*ScanResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	source := &sources.Files{
		Path:   absPath,
		Config: &s.detector.Config,
		Sema:   s.detector.Sema,
	}

	findings, err := s.detector.DetectSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	return s.buildResult(findings, absPath), nil
}

// ScanGitRepo scans git history for secrets
func (s *LeakScanner) ScanGitRepo(ctx context.Context, repoPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Use git log to scan history
	gitCmd, err := sources.NewGitLogCmdContext(ctx, absPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create git command: %w", err)
	}

	source := &sources.Git{
		Cmd:    gitCmd,
		Config: &s.detector.Config,
		Sema:   s.detector.Sema,
	}

	findings, err := s.detector.DetectSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("git scan failed: %w", err)
	}

	return s.buildResult(findings, absPath), nil
}

// ScanUnpushedCommits scans commits that haven't been pushed yet
func (s *LeakScanner) ScanUnpushedCommits(ctx context.Context, repoPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Scan from last push to HEAD
	gitCmd, err := sources.NewGitLogCmdContext(ctx, absPath, "@{push}..HEAD")
	if err != nil {
		// If no upstream tracking, scan staged changes instead
		return s.ScanStagedChanges(ctx, repoPath)
	}

	source := &sources.Git{
		Cmd:    gitCmd,
		Config: &s.detector.Config,
		Sema:   s.detector.Sema,
	}

	findings, err := s.detector.DetectSource(ctx, source)
	if err != nil {
		// Fallback to staged changes on error
		return s.ScanStagedChanges(ctx, repoPath)
	}

	return s.buildResult(findings, absPath), nil
}

// ScanStagedChanges scans only staged git changes (for pre-commit/pre-push)
func (s *LeakScanner) ScanStagedChanges(ctx context.Context, repoPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Use git diff --staged to get staged changes
	gitCmd, err := sources.NewGitDiffCmd(absPath, true)
	if err != nil {
		// Fallback to directory scan
		return s.ScanDirectory(ctx, repoPath)
	}

	source := &sources.Git{
		Cmd:    gitCmd,
		Config: &s.detector.Config,
		Sema:   s.detector.Sema,
	}

	findings, err := s.detector.DetectSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("staged scan failed: %w", err)
	}

	return s.buildResult(findings, absPath), nil
}

func (s *LeakScanner) buildResult(findings []report.Finding, path string) *ScanResult {
	result := &ScanResult{
		ScannedPath: path,
		HasLeaks:    len(findings) > 0,
		Findings:    make([]Finding, 0, len(findings)),
	}

	for _, f := range findings {
		result.Findings = append(result.Findings, Finding{
			RuleID:      f.RuleID,
			Description: f.Description,
			File:        f.File,
			Line:        f.StartLine,
			Secret:      f.Secret, // Already redacted by detector
			Match:       f.Match,
			Commit:      f.Commit,
			Author:      f.Author,
			Date:        f.Date,
		})
	}

	return result
}

// LoadGitleaksIgnore loads ignore patterns from .gitleaksignore
func (s *LeakScanner) LoadGitleaksIgnore(repoPath string) error {
	ignorePath := filepath.Join(repoPath, ".gitleaksignore")
	if _, err := os.Stat(ignorePath); err == nil {
		return s.detector.AddGitleaksIgnore(ignorePath)
	}
	return nil
}

// FormatFindings formats findings for display
func FormatFindings(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n⚠️  Found %d potential secret(s):\n\n", len(findings)))

	for i, f := range findings {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, f.Description))
		sb.WriteString(fmt.Sprintf("     Rule: %s\n", f.RuleID))
		sb.WriteString(fmt.Sprintf("     File: %s:%d\n", f.File, f.Line))
		if f.Commit != "" {
			sb.WriteString(fmt.Sprintf("     Commit: %s\n", f.Commit))
		}
		sb.WriteString(fmt.Sprintf("     Secret: %s\n", f.Secret))
		sb.WriteString("\n")
	}

	return sb.String()
}
