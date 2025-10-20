package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dyammarcano/clonr/internal/database"
	"github.com/spf13/cobra"
)

func MapRepos(_ *cobra.Command, args []string) error {
	rootDir := "."

	if len(args) > 0 {
		rootDir = args[0]
	}

	dbConn, err := database.InitDB()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	found := 0
	already := 0

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && info.Name() == ".git" {
			dotGit, err := dotGitCheck(path)
			if err != nil {
				return err
			}

			exists, err := dbConn.RepoExistsByURL(dotGit.URL)
			if err != nil {
				log.Printf("DB check failed for %s: %v\n", dotGit.Path, err)
			} else if exists {
				already++
				log.Printf("Already tracked: %s\n", dotGit.Path)
			} else {
				if err := dbConn.SaveRepo(dotGit.URL, dotGit.Path); err == nil {
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
