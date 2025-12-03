package core

import (
	"net/url"

	"github.com/inovacc/clonr/internal/database"
)

func RemoveRepo(urlStr string) error {
	db := database.GetDB()

	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	return db.RemoveRepoByURL(u)
}
