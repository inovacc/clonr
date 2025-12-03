package server

import (
	"net/http"
	"sync"

	"github.com/dyammarcano/clonr/internal/core"
	"github.com/dyammarcano/clonr/internal/database"
	"github.com/dyammarcano/clonr/internal/monitor"

	"github.com/gin-gonic/gin"
)

func StartServer(args []string) error {
	db := database.GetDB()

	r := gin.Default()

	var wg sync.WaitGroup

	r.GET("/repos", func(c *gin.Context) {
		repos, err := db.GetAllRepos()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, repos)
	})

	r.POST("/repos/update-all", func(c *gin.Context) {
		repos, err := db.GetAllRepos()
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

	wg.Go(monitor.Monitor(db))

	return r.Run(":4000")
}
