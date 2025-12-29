package core

import (
	"fmt"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

// ShowConfig displays the current configuration
func ShowConfig() error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := client.GetConfig()
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

// ResetConfig resets the configuration to default values
func ResetConfig() error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	defaultCfg := model.DefaultConfig()

	if err := client.SaveConfig(&defaultCfg); err != nil {
		return fmt.Errorf("failed to reset configuration: %w", err)
	}

	fmt.Println("✓ Configuration reset to defaults:")
	fmt.Println("==================================")
	fmt.Printf("Default Clone Directory: %s\n", defaultCfg.DefaultCloneDir)
	fmt.Printf("Editor:                  %s\n", defaultCfg.Editor)
	fmt.Printf("Terminal:                %s\n", defaultCfg.Terminal)
	fmt.Printf("Monitor Interval:        %d seconds\n", defaultCfg.MonitorInterval)
	fmt.Printf("Server Port:             %d\n", defaultCfg.ServerPort)

	return nil
}
