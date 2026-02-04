package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/store"
)

var (
	// ErrWorkspaceNotFound is returned when a workspace doesn't exist
	ErrWorkspaceNotFound = errors.New("workspace not found")

	// ErrWorkspaceExists is returned when trying to create a workspace that already exists
	ErrWorkspaceExists = errors.New("workspace already exists")
)

// WorkspaceService provides workspace operations with direct database access.
type WorkspaceService struct {
	store store.Store
}

// NewWorkspaceService creates a new WorkspaceService with direct database access.
func NewWorkspaceService(s store.Store) *WorkspaceService {
	return &WorkspaceService{store: s}
}

// GetWorkspace retrieves a workspace by name.
func (ws *WorkspaceService) GetWorkspace(name string) (*model.Workspace, error) {
	workspace, err := ws.store.GetWorkspace(name)
	if err != nil {
		return nil, err
	}

	if workspace == nil {
		return nil, ErrWorkspaceNotFound
	}

	return workspace, nil
}

// GetActiveWorkspace retrieves the currently active workspace.
func (ws *WorkspaceService) GetActiveWorkspace() (*model.Workspace, error) {
	return ws.store.GetActiveWorkspace()
}

// SetActiveWorkspace sets the active workspace.
func (ws *WorkspaceService) SetActiveWorkspace(name string) error {
	exists, err := ws.store.WorkspaceExists(name)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !exists {
		return ErrWorkspaceNotFound
	}

	return ws.store.SetActiveWorkspace(name)
}

// ListWorkspaces returns all workspaces.
func (ws *WorkspaceService) ListWorkspaces() ([]model.Workspace, error) {
	return ws.store.ListWorkspaces()
}

// SaveWorkspace saves a workspace to the database.
func (ws *WorkspaceService) SaveWorkspace(workspace *model.Workspace) error {
	return ws.store.SaveWorkspace(workspace)
}

// DeleteWorkspace removes a workspace.
func (ws *WorkspaceService) DeleteWorkspace(name string) error {
	exists, err := ws.store.WorkspaceExists(name)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !exists {
		return ErrWorkspaceNotFound
	}

	return ws.store.DeleteWorkspace(name)
}

// WorkspaceExists checks if a workspace exists.
func (ws *WorkspaceService) WorkspaceExists(name string) (bool, error) {
	return ws.store.WorkspaceExists(name)
}

// CreateWorkspace creates a new workspace.
func (ws *WorkspaceService) CreateWorkspace(name, path, description string) (*model.Workspace, error) {
	exists, err := ws.store.WorkspaceExists(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if exists {
		return nil, ErrWorkspaceExists
	}

	// Check if this is the first workspace
	workspaces, err := ws.store.ListWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	isFirst := len(workspaces) == 0

	workspace := &model.Workspace{
		Name:        name,
		Description: description,
		Path:        path,
		Active:      isFirst,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ws.store.SaveWorkspace(workspace); err != nil {
		return nil, fmt.Errorf("failed to save workspace: %w", err)
	}

	return workspace, nil
}

// GetReposByWorkspace returns repository URLs for a workspace.
func (ws *WorkspaceService) GetReposByWorkspace(workspace string) ([]string, error) {
	return ws.store.GetReposByWorkspace(workspace)
}

// UpdateRepoWorkspace updates a repository's workspace.
func (ws *WorkspaceService) UpdateRepoWorkspace(urlStr string, workspace string) error {
	return ws.store.UpdateRepoWorkspace(urlStr, workspace)
}
