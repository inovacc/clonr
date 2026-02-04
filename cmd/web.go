package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/inovacc/clonr/internal/web"
	"github.com/spf13/cobra"
)

var (
	webPort      int
	webNoBrowser bool
)

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Port to run the web server on")
	webCmd.Flags().BoolVar(&webNoBrowser, "no-browser", false, "Don't automatically open the browser")
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the web interface for profile management",
	Long: `Start a local web server that provides a browser-based interface
for managing GitHub profiles, workspaces, and integrations.

The web server runs on localhost only (127.0.0.1) for security.

Examples:
  clonr web                    # Start on default port 8080
  clonr web --port 9000        # Start on custom port
  clonr web --no-browser       # Don't auto-open browser`,
	RunE: runWeb,
}

func runWeb(_ *cobra.Command, _ []string) error {
	config := web.DefaultConfig()
	config.Port = webPort
	config.OpenBrowser = !webNoBrowser

	server, err := web.New(config)
	if err != nil {
		return fmt.Errorf("failed to create web server: %w", err)
	}

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		_, _ = fmt.Fprintln(os.Stdout, "\nShutting down...")
		cancel()
	}()

	_, _ = fmt.Fprintf(os.Stdout, "Starting web server on http://127.0.0.1:%d\n", config.Port)

	if config.OpenBrowser {
		_, _ = fmt.Fprintln(os.Stdout, "Opening browser...")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Open http://127.0.0.1:%d in your browser\n", config.Port)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Press Ctrl+C to stop")

	return server.Start(ctx)
}
