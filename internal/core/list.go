package core

import (
	"fmt"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

// ListRepos returns all repositories.
func ListRepos() ([]model.Repository, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return client.GetAllRepos()
}

// ListReposFiltered returns repos optionally filtered by favoritesOnly.
func ListReposFiltered(favoritesOnly bool) ([]model.Repository, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return client.GetRepos("", favoritesOnly)
}

// ListReposFilteredByWorkspace returns repos filtered by workspace.
// Server-side filtering is used for efficiency.
func ListReposFilteredByWorkspace(workspace string, favoritesOnly bool) ([]model.Repository, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return client.GetRepos(workspace, favoritesOnly)
}
