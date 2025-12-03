package core

import (
	"github.com/dyammarcano/clonr/internal/database"
	"github.com/dyammarcano/clonr/internal/model"
)

func ListRepos() ([]model.Repository, error) {
	db := database.GetDB()

	return db.GetAllRepos()
}

// ListReposFiltered returns repos optionally filtered by favoritesOnly.
func ListReposFiltered(favoritesOnly bool) ([]model.Repository, error) {
	db := database.GetDB()

	return db.GetRepos(favoritesOnly)
}
