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
	profiles, err := s.profileService.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		profiles = []model.Profile{}
	}

	activeProfile, _ := s.profileService.GetActiveProfile()

	workspaces, err := s.workspaceService.ListWorkspaces()
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
	profiles, err := s.profileService.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		profiles = []model.Profile{}
	}

	activeProfile, _ := s.profileService.GetActiveProfile()

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
	workspaces, err := s.workspaceService.ListWorkspaces()
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

// handleProfileEditPage renders the edit profile page
func (s *Server) handleProfileEditPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Redirect(w, r, "/profiles", http.StatusFound)
		return
	}

	profile, err := s.profileService.GetProfile(name)
	if err != nil || profile == nil {
		log.Printf("Profile not found: %s", name)
		http.Redirect(w, r, "/profiles", http.StatusFound)
		return
	}

	workspaces, err := s.workspaceService.ListWorkspaces()
	if err != nil {
		log.Printf("Failed to list workspaces: %v", err)
		workspaces = []model.Workspace{}
	}

	data := PageData{
		Title:        "Edit Profile",
		ActivePage:   "profiles",
		Workspaces:   workspaces,
		TPMAvailable: tpm.IsTPMAvailable(),
		Data: ProfileData{
			Profile: profile,
		},
	}

	s.render(w, "profile_edit.html", data)
}

// handleWorkspacesPage renders the workspaces page
func (s *Server) handleWorkspacesPage(w http.ResponseWriter, _ *http.Request) {
	workspaces, err := s.workspaceService.ListWorkspaces()
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
	activeProfile, _ := s.profileService.GetActiveProfile()

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
	profiles, err := s.profileService.ListProfiles()
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

	profile, err := s.profileService.GetProfile(name)
	if err != nil {
		s.jsonError(w, "Profile not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, profile)
}

// handleGetActiveProfile returns the active profile
func (s *Server) handleGetActiveProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := s.profileService.GetActiveProfile()
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
	exists, err := s.profileService.ProfileExists(name)
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
	wsExists, err := s.workspaceService.WorkspaceExists(workspace)
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

		// Create profile with token
		profile, err := s.profileService.CreateProfileWithToken(name, host, username, token, workspace, model.DefaultScopes())
		if err != nil {
			log.Printf("Failed to create profile: %v", err)
			s.jsonError(w, fmt.Sprintf("Failed to create profile: %v", err), http.StatusInternalServerError)
			return
		}

		// Broadcast SSE event
		s.BroadcastEvent(EventProfileCreated, "Profile created", map[string]any{
			"name": profile.Name,
			"user": profile.User,
			"host": profile.Host,
		})

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

// handleUpdateProfile updates an existing profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.jsonError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get existing profile
	profile, err := s.profileService.GetProfile(name)
	if err != nil || profile == nil {
		s.jsonError(w, "Profile not found", http.StatusNotFound)
		return
	}

	// Update workspace if changed
	workspace := strings.TrimSpace(r.FormValue("workspace"))
	if workspace != "" && workspace != profile.Workspace {
		// Verify workspace exists
		wsExists, err := s.workspaceService.WorkspaceExists(workspace)
		if err != nil {
			log.Printf("Failed to check workspace existence: %v", err)
			s.jsonError(w, fmt.Sprintf("Failed to check workspace: %v", err), http.StatusInternalServerError)
			return
		}
		if !wsExists {
			s.jsonError(w, fmt.Sprintf("Workspace %q not found", workspace), http.StatusBadRequest)
			return
		}
		profile.Workspace = workspace
	}

	// Update token if provided
	token := strings.TrimSpace(r.FormValue("token"))
	if token != "" {
		// Validate new token
		ctx := r.Context()
		valid, username, err := validateGitHubToken(ctx, token, profile.Host)
		if err != nil {
			log.Printf("Token validation error: %v", err)
			s.jsonError(w, fmt.Sprintf("Token validation failed: %v", err), http.StatusBadRequest)
			return
		}
		if !valid {
			s.jsonError(w, "Invalid or expired token", http.StatusBadRequest)
			return
		}

		// Encrypt new token
		encryptedToken, tokenStorage, err := s.profileService.EncryptToken(token, name, profile.Host)
		if err != nil {
			log.Printf("Token encryption failed: %v", err)
			s.jsonError(w, fmt.Sprintf("Failed to encrypt token: %v", err), http.StatusInternalServerError)
			return
		}

		profile.EncryptedToken = encryptedToken
		profile.User = username
		profile.TokenStorage = tokenStorage
	}

	// Update last used timestamp
	profile.LastUsedAt = time.Now()

	// Save profile
	if err := s.profileService.SaveProfile(profile); err != nil {
		log.Printf("Failed to save profile: %v", err)
		s.jsonError(w, fmt.Sprintf("Failed to save profile: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventProfileUpdated, "Profile updated", map[string]any{
		"name": profile.Name,
		"user": profile.User,
	})

	// Return success
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/profiles")
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Profile updated successfully",
		Data:    profile,
	})
}

// handleDeleteProfile deletes a profile
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if err := s.profileService.DeleteProfile(name); err != nil {
		s.jsonError(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventProfileDeleted, "Profile deleted", map[string]any{
		"name": name,
	})

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

	if err := s.profileService.SetActiveProfile(name); err != nil {
		s.jsonError(w, "Failed to set active profile", http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventProfileActivated, "Profile activated", map[string]any{
		"name": name,
	})

	// Check if HTMX request - refresh the profile list
	if r.Header.Get("HX-Request") == "true" {
		profiles, _ := s.profileService.ListProfiles()
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
	workspaces, err := s.workspaceService.ListWorkspaces()
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

	// Create workspace
	workspace, err := s.workspaceService.CreateWorkspace(name, path, description)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to create workspace: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventWorkspaceCreated, "Workspace created", map[string]any{
		"name": workspace.Name,
		"path": workspace.Path,
	})

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

	if err := s.workspaceService.DeleteWorkspace(name); err != nil {
		s.jsonError(w, "Failed to delete workspace", http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventWorkspaceDeleted, "Workspace deleted", map[string]any{
		"name": name,
	})

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
		"server_connected":     true, // Always true since we're in the server process
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
