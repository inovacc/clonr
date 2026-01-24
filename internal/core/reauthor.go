package core

import (
	"fmt"
	"os"

	git_nerds "github.com/inovacc/git-nerds"
)

// ReauthorOptions contains options for rewriting git author history.
type ReauthorOptions struct {
	// OldEmail is the email address to replace
	OldEmail string
	// NewEmail is the new email address
	NewEmail string
	// NewName is the new author/committer name (optional, keeps existing if empty)
	NewName string
	// RepoPath is the path to the repository (uses current directory if empty)
	RepoPath string
	// AllRefs rewrites all branches and tags when true
	AllRefs bool
}

// ReauthorResult contains the result of a reauthor operation.
type ReauthorResult struct {
	// CommitsRewritten is the number of commits that were rewritten
	CommitsRewritten int
	// TagsRewritten is the list of tags that were rewritten
	TagsRewritten []string
	// BranchesRewritten is the list of branches that were rewritten
	BranchesRewritten []string
}

// Reauthor rewrites git history to change author/committer email and name.
// This delegates to git-nerds library for the actual implementation.
func Reauthor(opts ReauthorOptions) (*ReauthorResult, error) {
	repoPath := opts.RepoPath
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		repoPath = cwd
	}

	// Open repository using git-nerds
	repo, err := git_nerds.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Convert options
	nerdsOpts := git_nerds.ReauthorOptions{
		OldEmail: opts.OldEmail,
		NewEmail: opts.NewEmail,
		NewName:  opts.NewName,
		AllRefs:  opts.AllRefs,
	}

	// Execute reauthor
	nerdsResult, err := repo.Reauthor(nerdsOpts)
	if err != nil {
		return nil, err
	}

	// Convert result
	return &ReauthorResult{
		CommitsRewritten:  nerdsResult.CommitsRewritten,
		TagsRewritten:     nerdsResult.TagsRewritten,
		BranchesRewritten: nerdsResult.BranchesRewritten,
	}, nil
}

// ListAuthors returns a list of unique author emails in the repository.
func ListAuthors(repoPath string) ([]string, error) {
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		repoPath = cwd
	}

	repo, err := git_nerds.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return repo.ListAuthorEmails()
}

// CountCommitsByEmail counts commits by a specific email address.
func CountCommitsByEmail(repoPath, email string) (int, error) {
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("failed to get current directory: %w", err)
		}

		repoPath = cwd
	}

	repo, err := git_nerds.Open(repoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open repository: %w", err)
	}

	return repo.CountCommitsByEmail(email)
}
