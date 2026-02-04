package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/core"
)

//go:embed templates/*.html templates/partials/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Config holds the web server configuration
type Config struct {
	Port        int
	Host        string
	OpenBrowser bool
}

// DefaultConfig returns the default web server configuration
func DefaultConfig() Config {
	return Config{
		Port:        8080,
		Host:        "127.0.0.1",
		OpenBrowser: true,
	}
}

// Server represents the web server
type Server struct {
	httpServer *http.Server
	grpcClient *grpc.Client
	pm         *core.ProfileManager
	config     Config
	templates  *template.Template
}

// New creates a new web server
func New(config Config) (*Server, error) {
	// Connect to gRPC server
	grpcClient, err := grpc.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Create profile manager
	pm, err := core.NewProfileManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create profile manager: %w", err)
	}

	// Parse templates
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		grpcClient: grpcClient,
		pm:         pm,
		config:     config,
		templates:  tmpl,
	}, nil
}

// parseTemplates parses all embedded HTML templates
func parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("Jan 02, 2006 15:04")
		},
		"formatTimeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			d := time.Since(t)
			switch {
			case d < time.Minute:
				return "just now"
			case d < time.Hour:
				m := int(d.Minutes())
				if m == 1 {
					return "1 minute ago"
				}
				return fmt.Sprintf("%d minutes ago", m)
			case d < 24*time.Hour:
				h := int(d.Hours())
				if h == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", h)
			case d < 7*24*time.Hour:
				days := int(d.Hours() / 24)
				if days == 1 {
					return "1 day ago"
				}
				return fmt.Sprintf("%d days ago", days)
			default:
				return t.Format("Jan 02, 2006")
			}
		},
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			if maxLen <= 3 {
				return s[:maxLen]
			}
			return s[:maxLen-3] + "..."
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse all templates from embedded FS
	tmpl, err := tmpl.ParseFS(templatesFS, "templates/*.html", "templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

// Start starts the web server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.loggingMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Open browser if configured
	if s.config.OpenBrowser {
		url := fmt.Sprintf("http://%s", addr)
		go func() {
			// Small delay to ensure server is ready
			time.Sleep(100 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("Failed to open browser: %v", err)
				log.Printf("Open manually: %s", url)
			}
		}()
	}

	log.Printf("Web server starting on http://%s", addr)

	// Start server
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	return s.Shutdown(context.Background()) //nolint:contextcheck // parent context cancelled, use background for shutdown
}

// Shutdown gracefully shuts down the web server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Println("Shutting down web server...")
	return s.httpServer.Shutdown(shutdownCtx)
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// render renders a template with the given data
func (s *Server) render(w http.ResponseWriter, templateName string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders a partial template (for HTMX)
func (s *Server) renderPartial(w http.ResponseWriter, templateName string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
