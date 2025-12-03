package core

import (
	"log"
	"os/exec"

	"github.com/inovacc/clonr/internal/database"
)

// UpdateAllRepos pulls the latest changes for all repositories in the clonr database.
func UpdateAllRepos() {
	db := database.GetDB()

	repos, err := db.GetAllRepos()
	if err != nil {
		log.Printf("Failed to get repositories: %v\n", err)

		return
	}

	for _, repo := range repos {
		_ = UpdateRepo(repo.Path)
	}
}

func UpdateRepo(path string) error {
	log.Printf("Updating %s...", path)

	cmd := exec.Command("git", "pull", "origin")
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[pull error] %v: %s\n", err, string(output))

		return err
	}

	log.Printf("[updated] %s\n", output)

	return nil
}
