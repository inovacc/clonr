package core

import (
	"fmt"

	"github.com/dyammarcano/clonr/internal/database"
)

// ShowConfig displays the current configuration
func ShowConfig() error {
	db := database.GetDB()

	cfg, err := db.GetConfig()
	if err != nil {
		return err
	}

	fmt.Println("Current Configuration:")
	fmt.Println("=====================")
	fmt.Printf("Default Clone Directory: %s\n", cfg.DefaultCloneDir)
	fmt.Printf("Editor:                  %s\n", cfg.Editor)
	fmt.Printf("Terminal:                %s\n", cfg.Terminal)
	fmt.Printf("Monitor Interval:        %d seconds\n", cfg.MonitorInterval)
	fmt.Printf("Server Port:             %d\n", cfg.ServerPort)

	return nil
}
