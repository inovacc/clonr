package core

import (
	"log"
	"os"
	"path/filepath"

	"github.com/inovacc/clonr/internal/database"
)

func MapRepos(args []string) error {
	rootDir := "."

	if len(args) > 0 {
		rootDir = args[0]
	}

	db := database.GetDB()

	found := 0
	already := 0

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			dotGit, err := dotGitCheck(path)
			if err != nil {
				return err
			}

			exists, err := db.RepoExistsByURL(dotGit.URL)
			switch {
			case err != nil:
				log.Printf("DB check failed for %s: %v\n", dotGit.Path, err)

				return nil
			case exists:
				already++

				log.Printf("Already tracked: %s\n", dotGit.Path)

				return nil
			default:
				if err := db.SaveRepo(dotGit.URL, dotGit.Path); err == nil {
					log.Printf("Added: %s\n", dotGit.Path)

					found++
				} else {
					log.Printf("Failed to add %s: %v\n", dotGit.Path, err)
				}
			}

			// Don't recurse into .git
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		log.Printf("Error searching for git repos: %v\n", err)
	}

	log.Printf("%d new repositories mapped. %d already tracked.\n", found, already)

	return nil
}
