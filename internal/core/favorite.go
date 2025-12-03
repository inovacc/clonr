package core

import "github.com/inovacc/clonr/internal/database"

func SetFavoriteByURL(url string, fav bool) error {
	db := database.GetDB()

	return db.SetFavoriteByURL(url, fav)
}
