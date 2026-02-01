package core

import (
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/model"
)

// ShowConfig displays the current configuration
func ShowConfig() error {
	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := client.GetConfig()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Current Configuration:")
	_, _ = fmt.Fprintln(os.Stdout, "=====================")
	_, _ = fmt.Fprintf(os.Stdout, "Default Clone Directory: %s\n", cfg.DefaultCloneDir)
	_, _ = fmt.Fprintf(os.Stdout, "Editor:                  %s\n", cfg.Editor)
	_, _ = fmt.Fprintf(os.Stdout, "Terminal:                %s\n", cfg.Terminal)
	_, _ = fmt.Fprintf(os.Stdout, "Monitor Interval:        %d seconds\n", cfg.MonitorInterval)
	_, _ = fmt.Fprintf(os.Stdout, "Server Port:             %d\n", cfg.ServerPort)

	return nil
}

// ResetConfig resets the configuration to default values
func ResetConfig() error {
	client, err := grpc.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	defaultCfg := model.DefaultConfig()

	if err := client.SaveConfig(&defaultCfg); err != nil {
		return fmt.Errorf("failed to reset configuration: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "✓ Configuration reset to defaults:")
	_, _ = fmt.Fprintln(os.Stdout, "==================================")
	_, _ = fmt.Fprintf(os.Stdout, "Default Clone Directory: %s\n", defaultCfg.DefaultCloneDir)
	_, _ = fmt.Fprintf(os.Stdout, "Editor:                  %s\n", defaultCfg.Editor)
	_, _ = fmt.Fprintf(os.Stdout, "Terminal:                %s\n", defaultCfg.Terminal)
	_, _ = fmt.Fprintf(os.Stdout, "Monitor Interval:        %d seconds\n", defaultCfg.MonitorInterval)
	_, _ = fmt.Fprintf(os.Stdout, "Server Port:             %d\n", defaultCfg.ServerPort)

	return nil
}
