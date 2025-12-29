package core

import (
	"fmt"
	"net/url"

	"github.com/inovacc/clonr/internal/grpcclient"
)

func RemoveRepo(urlStr string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	return client.RemoveRepoByURL(u)
}
