package service

import "github.com/inovacc/clonr/internal/database"

func Service() error {
	db := database.GetDB()

	cfg, err := db.GetConfig()
	if err != nil {
		return err
	}

	return nil
}
