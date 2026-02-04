package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
)

// PageData holds common data for page templates
type PageData struct {
	Title         string
	ActivePage    string
	Profiles      []model.Profile
	Workspaces    []model.Workspace
	ActiveProfile *model.Profile
	TPMAvailable  bool
	Error         string
	Success       string
	Data          any
}

// ProfileData holds profile-specific template data
type ProfileData struct {
	Profile   *model.Profile
	Token     string
	Validated bool
	Valid     bool
}

// APIResponse is a generic API response
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleIndex renders the main dashboard
func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	profiles, err := s.pm.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		profiles = []model.Profile{}
	}

	activeProfile, _ := s.pm.GetActiveProfile()

	workspaces, err := s.grpcClient.ListWorkspaces() //nolint:contextcheck // client manages its own timeout
	if err != nil {
		log.Printf("Failed to list workspaces: %v", err)
		workspaces = []model.Workspace{}
	}

	data := PageData{
		Title:         "Dashboard",
		ActivePage:    "dashboard",
		Profiles:      profiles,
		Workspaces:    workspaces,
		ActiveProfile: activeProfile,
		TPMAvailable:  tpm.IsTPMAvailable(),
	}

	s.render(w, "index.html", data)
}

// handleProfilesPage renders the profiles page
func (s *Server) handleProfilesPage(w http.ResponseWriter, _ *http.Request) {
	profiles, err := s.pm.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		profiles = []model.Profile{}
	}

	activeProfile, _ := s.pm.GetActiveProfile()

	data := PageData{
		Title:         "Profiles",
		ActivePage:    "profiles",
		Profiles:      profiles,
		ActiveProfile: activeProfile,
		TPMAvailable:  tpm.IsTPMAvailable(),
	}

	s.render(w, "profiles.html", data)
}

// handleProfileAddPage renders the add profile page
func (s *Server) handleProfileAddPage(w http.ResponseWriter, _ *http.Request) {
	workspaces, err := s.grpcClient.ListWorkspaces() //nolint:contextcheck // client manages its own timeout
	if err != nil {
		log.Printf("Failed to list workspaces: %v", err)
		workspaces = []model.Workspace{}
	}

	data := PageData{
		Title:        "Add Profile",
		ActivePage:   "profiles",
		Workspaces:   workspaces,
		TPMAvailable: tpm.IsTPMAvailable(),
	}

	s.render(w, "profile_add.html", data)
}

// handleWorkspacesPage renders the workspaces page
func (s *Server) handleWorkspacesPage(w http.ResponseWriter, _ *http.Request) {
	workspaces, err := s.grpcClient.ListWorkspaces() //nolint:contextcheck // client manages its own timeout
	if err != nil {
		log.Printf("Failed to list workspaces: %v", err)
		workspaces = []model.Workspace{}
	}

	data := PageData{
		Title:        "Workspaces",
		ActivePage:   "workspaces",
		Workspaces:   workspaces,
		TPMAvailable: tpm.IsTPMAvailable(),
	}

	s.render(w, "workspaces.html", data)
}

// handleSlackPage renders the Slack integration page
func (s *Server) handleSlackPage(w http.ResponseWriter, _ *http.Request) {
	activeProfile, _ := s.pm.GetActiveProfile()

	data := PageData{
		Title:         "Slack Integration",
		ActivePage:    "slack",
		ActiveProfile: activeProfile,
		TPMAvailable:  tpm.IsTPMAvailable(),
	}

	s.render(w, "slack.html", data)
}

// handleListProfiles returns all profiles as JSON or HTML partial
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.pm.ListProfiles()
	if err != nil {
		s.jsonError(w, "Failed to list profiles", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "profile_list.html", profiles)
		return
	}

	s.jsonResponse(w, profiles)
}

// handleGetProfile returns a single profile
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	profile, err := s.pm.GetProfile(name)
	if err != nil {
		s.jsonError(w, "Profile not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, profile)
}

// handleGetActiveProfile returns the active profile
func (s *Server) handleGetActiveProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := s.pm.GetActiveProfile()
	if err != nil {
		s.jsonError(w, "No active profile", http.StatusNotFound)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "profile_row.html", profile)
		return
	}

	s.jsonResponse(w, profile)
}

// handleCreateProfile creates a new profile with a token
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.jsonError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	token := strings.TrimSpace(r.FormValue("token"))
	workspace := strings.TrimSpace(r.FormValue("workspace"))
	host := strings.TrimSpace(r.FormValue("host"))

	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if workspace == "" {
		s.jsonError(w, "Workspace required", http.StatusBadRequest)
		return
	}

	if host == "" {
		host = "github.com"
	}

	// Check if profile exists
	exists, err := s.grpcClient.ProfileExists(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		log.Printf("Failed to check profile existence for %q: %v", name, err)
		s.jsonError(w, fmt.Sprintf("Failed to check profile existence: %v", err), http.StatusInternalServerError)
		return
	}
	if exists {
		s.jsonError(w, "Profile already exists", http.StatusConflict)
		return
	}

	// Check workspace exists
	wsExists, err := s.grpcClient.WorkspaceExists(workspace) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		log.Printf("Failed to check workspace existence for %q: %v", workspace, err)
		s.jsonError(w, fmt.Sprintf("Failed to check workspace existence: %v", err), http.StatusInternalServerError)
		return
	}
	if !wsExists {
		s.jsonError(w, fmt.Sprintf("Workspace %q not found. Create it first.", workspace), http.StatusBadRequest)
		return
	}

	// If token provided, validate and create profile
	if token != "" {
		// Validate token
		ctx := r.Context()
		valid, username, err := validateGitHubToken(ctx, token, host)
		if err != nil {
			log.Printf("Token validation error for host %q: %v", host, err)
			s.jsonError(w, fmt.Sprintf("Token validation failed: %v", err), http.StatusBadRequest)
			return
		}
		if !valid {
			s.jsonError(w, "Invalid or expired token. Please check your Personal Access Token.", http.StatusBadRequest)
			return
		}
		log.Printf("Token validated successfully for user %q on host %q", username, host)

		// Encrypt token
		encryptedToken, err := tpm.EncryptToken(token, name, host)
		if err != nil {
			log.Printf("Token encryption failed: %v", err)
			s.jsonError(w, fmt.Sprintf("Failed to encrypt token: %v", err), http.StatusInternalServerError)
			return
		}

		tokenStorage := model.TokenStorageEncrypted
		if tpm.IsDataOpen(encryptedToken) {
			tokenStorage = model.TokenStorageOpen
		}

		// Check if first profile
		profiles, _ := s.pm.ListProfiles()
		isFirst := len(profiles) == 0

		// Create profile
		profile := &model.Profile{
			Name:           name,
			Host:           host,
			User:           username,
			TokenStorage:   tokenStorage,
			Scopes:         model.DefaultScopes(),
			Default:        isFirst,
			EncryptedToken: encryptedToken,
			CreatedAt:      time.Now(),
			LastUsedAt:     time.Now(),
			Workspace:      workspace,
		}

		if err := s.grpcClient.SaveProfile(profile); err != nil { //nolint:contextcheck // client manages its own timeout
			log.Printf("Failed to save profile: %v", err)
			s.jsonError(w, fmt.Sprintf("Failed to save profile: %v", err), http.StatusInternalServerError)
			return
		}

		// Return success
		if r.Header.Get("HX-Request") == "true" {
			// Redirect to profiles page
			w.Header().Set("HX-Redirect", "/profiles")
			w.WriteHeader(http.StatusOK)
			return
		}

		s.jsonResponse(w, APIResponse{
			Success: true,
			Message: "Profile created successfully",
			Data:    profile,
		})
		return
	}

	// No token - need to start OAuth flow
	s.jsonError(w, "Token required (OAuth flow not implemented for web)", http.StatusBadRequest)
}

// handleDeleteProfile deletes a profile
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if err := s.pm.DeleteProfile(name); err != nil {
		s.jsonError(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return empty response to remove the row
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Profile deleted",
	})
}

// handleSetActiveProfile sets a profile as active
func (s *Server) handleSetActiveProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if err := s.pm.SetActiveProfile(name); err != nil {
		s.jsonError(w, "Failed to set active profile", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request - refresh the profile list
	if r.Header.Get("HX-Request") == "true" {
		profiles, _ := s.pm.ListProfiles()
		s.renderPartial(w, "profile_list.html", profiles)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Active profile set",
	})
}

// handleListWorkspaces returns all workspaces
func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := s.grpcClient.ListWorkspaces() //nolint:contextcheck // client manages its own timeout
	if err != nil {
		s.jsonError(w, "Failed to list workspaces", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "workspace_list.html", workspaces)
		return
	}

	s.jsonResponse(w, workspaces)
}

// handleCreateWorkspace creates a new workspace
func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.jsonError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	path := strings.TrimSpace(r.FormValue("path"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		s.jsonError(w, "Workspace name required", http.StatusBadRequest)
		return
	}

	if path == "" {
		s.jsonError(w, "Workspace path required", http.StatusBadRequest)
		return
	}

	// Check if workspace exists
	exists, err := s.grpcClient.WorkspaceExists(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		s.jsonError(w, "Failed to check workspace existence", http.StatusInternalServerError)
		return
	}
	if exists {
		s.jsonError(w, "Workspace already exists", http.StatusConflict)
		return
	}

	// Check if first workspace
	workspaces, _ := s.grpcClient.ListWorkspaces() //nolint:contextcheck // client manages its own timeout
	isFirst := len(workspaces) == 0

	workspace := &model.Workspace{
		Name:        name,
		Description: description,
		Path:        path,
		Active:      isFirst,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.grpcClient.SaveWorkspace(workspace); err != nil { //nolint:contextcheck // client manages its own timeout
		s.jsonError(w, "Failed to save workspace", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/workspaces")
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Workspace created",
		Data:    workspace,
	})
}

// handleDeleteWorkspace deletes a workspace
func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Workspace name required", http.StatusBadRequest)
		return
	}

	if err := s.grpcClient.DeleteWorkspace(name); err != nil { //nolint:contextcheck // client manages its own timeout
		s.jsonError(w, "Failed to delete workspace", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Workspace deleted",
	})
}

// handleStatus returns system status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"tpm_available":        tpm.IsTPMAvailable(),
		"encryption_available": tpm.IsEncryptionAvailable(),
		"server_connected":     s.grpcClient.Ping() == nil, //nolint:contextcheck // client manages its own timeout
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "status.html", status)
		return
	}

	s.jsonResponse(w, status)
}

// handleHealth returns health check status
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// jsonResponse writes a JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

// jsonError writes a JSON error response
func (s *Server) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	}); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}
