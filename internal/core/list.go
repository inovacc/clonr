package core

import (
	"fmt"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

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

	return client.GetRepos(favoritesOnly)
}
