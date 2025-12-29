package core

import (
	"fmt"

	"github.com/inovacc/clonr/internal/grpcclient"
)

func SetFavoriteByURL(url string, fav bool) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	return client.SetFavoriteByURL(url, fav)
}
