package web

import (
	"io/fs"
	"net/http"
)

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Static files
	staticSubFS, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))

	// Pages
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /profiles", s.handleProfilesPage)
	mux.HandleFunc("GET /profiles/add", s.handleProfileAddPage)
	mux.HandleFunc("GET /workspaces", s.handleWorkspacesPage)
	mux.HandleFunc("GET /slack", s.handleSlackPage)

	// Profile API
	mux.HandleFunc("GET /api/profiles", s.handleListProfiles)
	mux.HandleFunc("GET /api/profiles/active", s.handleGetActiveProfile)
	mux.HandleFunc("GET /api/profiles/{name}", s.handleGetProfile)
	mux.HandleFunc("POST /api/profiles", s.handleCreateProfile)
	mux.HandleFunc("DELETE /api/profiles/{name}", s.handleDeleteProfile)
	mux.HandleFunc("PUT /api/profiles/{name}/active", s.handleSetActiveProfile)

	// Workspace API
	mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	mux.HandleFunc("POST /api/workspaces", s.handleCreateWorkspace)
	mux.HandleFunc("DELETE /api/workspaces/{name}", s.handleDeleteWorkspace)

	// OAuth flows
	mux.HandleFunc("GET /oauth/github/start", s.handleGitHubOAuthStart)
	mux.HandleFunc("GET /oauth/github/status", s.handleGitHubOAuthStatus)
	mux.HandleFunc("GET /oauth/slack/start", s.handleSlackOAuthStart)
	mux.HandleFunc("GET /oauth/slack/callback", s.handleSlackOAuthCallback)

	// Slack API
	mux.HandleFunc("GET /api/slack/status", s.handleSlackStatus)
	mux.HandleFunc("POST /api/slack/add", s.handleSlackAdd)
	mux.HandleFunc("DELETE /api/slack/remove", s.handleSlackRemove)

	// System
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /health", s.handleHealth)
}
