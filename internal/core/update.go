package core

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/inovacc/clonr/internal/client/grpc"
)

// UpdateAllRepos pulls the latest changes for all repositories in the clonr database.
func UpdateAllRepos() {
	client, err := grpc.GetClient()
	if err != nil {
		log.Printf("Failed to connect to server: %v\n", err)
		return
	}

	repos, err := client.GetAllRepos()
	if err != nil {
		log.Printf("Failed to get repositories: %v\n", err)

		return
	}

	for _, repo := range repos {
		_ = UpdateRepo(repo.URL, repo.Path)
	}
}

func UpdateRepo(url, path string) error {
	log.Printf("Updating %s...", path)

	cmd := exec.Command("git", "pull", "origin")
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[pull error] %v: %s\n", err, string(output))

		return err
	}

	log.Printf("[updated] %s\n", output)

	// Update the timestamp in the database
	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := client.UpdateRepoTimestamp(url); err != nil {
		log.Printf("Failed to update timestamp for %s: %v\n", url, err)
	}

	return nil
}
