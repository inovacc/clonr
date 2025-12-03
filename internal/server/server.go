package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/monitor"

	"github.com/gin-gonic/gin"
)

func StartServer(args []string) error {
	db := database.GetDB()

	cfg, err := db.GetConfig()
	if err != nil {
		return err
	}

	gin.SetMode(gin.ReleaseMode)

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

	port := fmt.Sprintf(":%d", cfg.ServerPort)

	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	fmt.Printf("Listening and serving HTTP on %s\nctrl+c, shutting down server...\n", port)

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Handle Ctrl+C (SIGINT) to quit gracefully with a message
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)

		return err
	}

	return nil
}
