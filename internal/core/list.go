package core

import (
	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/model"
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
