package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/inovacc/clonr/internal/application"
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
	Short: "Manage clonr server as a system service",
	Long: `Install, uninstall, start, stop, or check the status of clonr server as a system service.

On Windows, this creates/manages a Windows Service.
On Linux/macOS, this creates/manages a systemd/launchd service.`,
	RunE: runService,
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().BoolVar(&serviceStart, "start", false, "Start the clonr server service")
	serviceCmd.Flags().BoolVar(&serviceStop, "stop", false, "Stop the clonr server service")
	serviceCmd.Flags().BoolVar(&serviceInstall, "install", false, "Install clonr server as a system service")
	serviceCmd.Flags().BoolVar(&serviceUninstall, "uninstall", false, "Uninstall clonr server system service")
	serviceCmd.Flags().BoolVar(&serviceStatus, "status", false, "Check clonr server service status")
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
	// Execute: clonr server start --port <port>
	clonrPath, err := findClonrExecutable()
	if err != nil {
		_ = service.ConsoleLogger.Errorf("Failed to find clonr: %v", err)
		return
	}

	args := []string{"server", "start", "--port", fmt.Sprintf("%d", p.port), "--idle-timeout", "0"}
	cmd := exec.Command(clonrPath, args...)
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

func runService(_ *cobra.Command, _ []string) error {
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
		Arguments:   []string{"server", "start", "--port", fmt.Sprintf("%d", servicePort), "--idle-timeout", "0"},
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
	// First, find the clonr executable
	clonrPath, err := findClonrExecutable()
	if err != nil {
		return fmt.Errorf("cannot find clonr executable: %w\n\nPlease ensure clonr is installed or in your PATH", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Installing clonr server service...\n")
	_, _ = fmt.Fprintf(os.Stdout, "Executable: %s\n", clonrPath)
	_, _ = fmt.Fprintf(os.Stdout, "Command: clonr server start --port %d\n", servicePort)

	err = s.Install()
	if err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "✓ Service installed successfully!")
	_, _ = fmt.Fprintln(os.Stdout, "\nTo start the service, run:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr service --start")
	_, _ = fmt.Fprintln(os.Stdout, "\nOr use your system's service manager:")
	_, _ = fmt.Fprintf(os.Stdout, "  Windows: sc start ClonrServer\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Linux:   sudo systemctl start clonr\n")

	return nil
}

func uninstallService(s service.Service) error {
	_, _ = fmt.Fprintln(os.Stdout, "Uninstalling clonr server service...")

	// Try to stop first
	_ = s.Stop()

	err := s.Uninstall()
	if err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "✓ Service uninstalled successfully!")

	return nil
}

func startService(s service.Service) error {
	_, _ = fmt.Fprintln(os.Stdout, "Starting clonr server service...")

	err := s.Start()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "✓ Service started successfully!")
	_, _ = fmt.Fprintf(os.Stdout, "\nServer is running on port %d\n", servicePort)
	_, _ = fmt.Fprintln(os.Stdout, "You can now use clonr commands:")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr list")
	_, _ = fmt.Fprintln(os.Stdout, "  clonr clone https://github.com/user/repo")

	return nil
}

func stopService(s service.Service) error {
	_, _ = fmt.Fprintln(os.Stdout, "Stopping clonr-server service...")

	err := s.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "✓ Service stopped successfully!")

	return nil
}

func statusService(s service.Service) error {
	status, err := s.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Service Status: ")

	switch status {
	case service.StatusRunning:
		_, _ = fmt.Fprintln(os.Stdout, "Running ✓")
	case service.StatusStopped:
		_, _ = fmt.Fprintln(os.Stdout, "Stopped")
	case service.StatusUnknown:
		_, _ = fmt.Fprintln(os.Stdout, "Unknown")
	default:
		_, _ = fmt.Fprintf(os.Stdout, "%v\n", status)
	}

	return nil
}

// findClonrExecutable locates the clonr executable
func findClonrExecutable() (string, error) {
	exeName := application.AppExeName
	exeNameWin := application.AppExeNameWindows

	// Check common locations
	locations := []string{
		exeName,               // In PATH
		exeNameWin,            // In PATH (Windows)
		"./bin/" + exeName,    // Relative to current dir
		"./bin/" + exeNameWin, // Relative to current dir (Windows)
	}

	// Also check in the same directory as the current executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		locations = append(locations,
			filepath.Join(exeDir, exeName),
			filepath.Join(exeDir, exeNameWin),
		)
		// If already running clonr, use it directly
		if filepath.Base(exePath) == exeName || filepath.Base(exePath) == exeNameWin {
			absPath, _ := filepath.Abs(exePath)
			return absPath, nil
		}
	}

	// Try each location
	for _, loc := range locations {
		if path, err := exec.LookPath(loc); err == nil {
			// Found it! Return absolute path
			absPath, _ := filepath.Abs(path)
			return absPath, nil
		}
	}

	return "", fmt.Errorf("%s executable not found in PATH or common locations", application.AppName)
}
