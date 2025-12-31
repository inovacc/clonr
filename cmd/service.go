package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

var (
	serviceStart     bool
	serviceStop      bool
	serviceInstall   bool
	serviceUninstall bool
	serviceStatus    bool
	servicePort      int
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage clonr-server as a system service",
	Long: `Install, uninstall, start, stop, or check the status of clonr-server as a system service.

On Windows, this creates/manages a Windows Service.
On Linux/macOS, this creates/manages a systemd/launchd service.`,
	RunE: runService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().BoolVar(&serviceStart, "start", false, "Start the clonr-server service")
	serviceCmd.Flags().BoolVar(&serviceStop, "stop", false, "Stop the clonr-server service")
	serviceCmd.Flags().BoolVar(&serviceInstall, "install", false, "Install clonr-server as a system service")
	serviceCmd.Flags().BoolVar(&serviceUninstall, "uninstall", false, "Uninstall clonr-server system service")
	serviceCmd.Flags().BoolVar(&serviceStatus, "status", false, "Check clonr-server service status")
	serviceCmd.Flags().IntVarP(&servicePort, "port", "p", 50051, "Port for the server to listen on")
}

// Program implements the service.Interface
type program struct {
	port int
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// This is where the actual service work happens
	// In our case, we just exec clonr-server
	serverPath, err := findServerExecutable()
	if err != nil {
		_ = service.ConsoleLogger.Errorf("Failed to find clonr-server: %v", err)
		return
	}

	args := []string{"start", "--port", fmt.Sprintf("%d", p.port)}
	cmd := exec.Command(serverPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		_ = service.ConsoleLogger.Errorf("Server exited with error: %v", err)
	}
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func runService(cmd *cobra.Command, args []string) error {
	// Count how many flags are set
	flagCount := 0
	if serviceStart {
		flagCount++
	}
	if serviceStop {
		flagCount++
	}
	if serviceInstall {
		flagCount++
	}
	if serviceUninstall {
		flagCount++
	}
	if serviceStatus {
		flagCount++
	}

	if flagCount == 0 {
		return fmt.Errorf("please specify one of: --start, --stop, --install, --uninstall, --status")
	}

	if flagCount > 1 {
		return fmt.Errorf("please specify only one operation at a time")
	}

	// Setup service configuration
	svcConfig := &service.Config{
		Name:        "ClonrServer",
		DisplayName: "Clonr Repository Server",
		Description: "Clonr gRPC server for managing Git repository metadata",
		Arguments:   []string{"start", "--port", fmt.Sprintf("%d", servicePort)},
	}

	prg := &program{port: servicePort}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Handle the requested operation
	switch {
	case serviceInstall:
		return installService(s)
	case serviceUninstall:
		return uninstallService(s)
	case serviceStart:
		return startService(s)
	case serviceStop:
		return stopService(s)
	case serviceStatus:
		return statusService(s)
	}

	return nil
}

func installService(s service.Service) error {
	// First, find the clonr-server executable
	serverPath, err := findServerExecutable()
	if err != nil {
		return fmt.Errorf("cannot find clonr-server executable: %w\n\nPlease ensure clonr-server is installed or in your PATH", err)
	}

	fmt.Printf("Installing clonr-server service...\n")
	fmt.Printf("Server executable: %s\n", serverPath)
	fmt.Printf("Port: %d\n", servicePort)

	err = s.Install()
	if err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("✓ Service installed successfully!")
	fmt.Println("\nTo start the service, run:")
	fmt.Println("  clonr service --start")
	fmt.Println("\nOr use your system's service manager:")
	fmt.Printf("  Windows: sc start ClonrServer\n")
	fmt.Printf("  Linux:   sudo systemctl start clonr-server\n")

	return nil
}

func uninstallService(s service.Service) error {
	fmt.Println("Uninstalling clonr-server service...")

	// Try to stop first
	_ = s.Stop()

	err := s.Uninstall()
	if err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	fmt.Println("✓ Service uninstalled successfully!")
	return nil
}

func startService(s service.Service) error {
	fmt.Println("Starting clonr-server service...")

	err := s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Println("✓ Service started successfully!")
	fmt.Printf("\nServer is running on port %d\n", servicePort)
	fmt.Println("You can now use clonr commands:")
	fmt.Println("  clonr list")
	fmt.Println("  clonr clone https://github.com/user/repo")

	return nil
}

func stopService(s service.Service) error {
	fmt.Println("Stopping clonr-server service...")

	err := s.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Println("✓ Service stopped successfully!")
	return nil
}

func statusService(s service.Service) error {
	status, err := s.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	fmt.Printf("Service Status: ")
	switch status {
	case service.StatusRunning:
		fmt.Println("Running ✓")
	case service.StatusStopped:
		fmt.Println("Stopped")
	case service.StatusUnknown:
		fmt.Println("Unknown")
	default:
		fmt.Printf("%v\n", status)
	}

	return nil
}

// findServerExecutable locates the clonr-server executable
func findServerExecutable() (string, error) {
	// Check common locations
	locations := []string{
		"clonr-server",           // In PATH
		"clonr-server.exe",       // In PATH (Windows)
		"./bin/clonr-server",     // Relative to current dir
		"./bin/clonr-server.exe", // Relative to current dir (Windows)
	}

	// Also check in the same directory as the current executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		locations = append(locations,
			filepath.Join(exeDir, "clonr-server"),
			filepath.Join(exeDir, "clonr-server.exe"),
		)
	}

	// Try each location
	for _, loc := range locations {
		if path, err := exec.LookPath(loc); err == nil {
			// Found it! Return absolute path
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("clonr-server executable not found in PATH or common locations")
}
