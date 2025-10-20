package server

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/dyammarcano/clonr/internal/core"
	"github.com/dyammarcano/clonr/internal/database"
	"github.com/dyammarcano/clonr/internal/monitor"
	"github.com/spf13/cobra"

	"github.com/gin-gonic/gin"
)

func StartServer(cmd *cobra.Command, args []string) error {
	initDB, err := database.InitDB()
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	r := gin.Default()

	var wg sync.WaitGroup

	r.GET("/repos", func(c *gin.Context) {
		repos, err := initDB.GetAllRepos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, repos)
	})

	r.POST("/repos/update-all", func(c *gin.Context) {
		repos, err := initDB.GetAllRepos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var results = make(map[string]string)

		for _, repo := range repos {
			if err := core.PullRepo(repo.Path); err != nil {
				results[repo.URL] = "error: " + err.Error()
			} else {
				results[repo.URL] = "updated"
			}
		}

		c.JSON(http.StatusOK, results)
	})

	wg.Go(monitor.Monitor(initDB))

	return r.Run(":4000")
}
