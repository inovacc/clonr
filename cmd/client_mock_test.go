package cmd

import (
	"github.com/inovacc/clonr/internal/model"
)

// MockClient is a mock implementation of ClientInterface for testing.
type MockClient struct {
	// Profile data
	Profiles      []model.Profile
	ActiveProfile *model.Profile

	// Workspace data
	Workspaces      []model.Workspace
	ActiveWorkspace *model.Workspace

	// Config data
	Config *model.Config

	// Error injection
	SetActiveProfileErr   error
	GetProfileErr         error
	GetActiveProfileErr   error
	ListProfilesErr       error
	ListWorkspacesErr     error
	GetActiveWorkspaceErr error
	SaveWorkspaceErr      error
	GetConfigErr          error

	// Call tracking
	SetActiveProfileCalled bool
	SetActiveProfileName   string
	GetProfileCalled       bool
	GetProfileName         string
	SaveWorkspaceCalled    bool
	SavedWorkspace         *model.Workspace
}

// NewMockClient creates a new MockClient with default values.
func NewMockClient() *MockClient {
	return &MockClient{
		Profiles:   []model.Profile{},
		Workspaces: []model.Workspace{},
		Config: &model.Config{
			DefaultCloneDir: "/tmp/clonr",
		},
	}
}

// SetActiveProfile implements ClientInterface.
func (m *MockClient) SetActiveProfile(name string) error {
	m.SetActiveProfileCalled = true

	m.SetActiveProfileName = name
	if m.SetActiveProfileErr != nil {
		return m.SetActiveProfileErr
	}
	// Find and set the profile as active
	for i := range m.Profiles {
		if m.Profiles[i].Name == name {
			m.ActiveProfile = &m.Profiles[i]
			return nil
		}
	}

	return nil
}

// GetProfile implements ClientInterface.
func (m *MockClient) GetProfile(name string) (*model.Profile, error) {
	m.GetProfileCalled = true

	m.GetProfileName = name
	if m.GetProfileErr != nil {
		return nil, m.GetProfileErr
	}

	for i := range m.Profiles {
		if m.Profiles[i].Name == name {
			return &m.Profiles[i], nil
		}
	}

	return nil, nil
}

// GetActiveProfile implements ClientInterface.
func (m *MockClient) GetActiveProfile() (*model.Profile, error) {
	if m.GetActiveProfileErr != nil {
		return nil, m.GetActiveProfileErr
	}

	return m.ActiveProfile, nil
}

// ListProfiles implements ClientInterface.
func (m *MockClient) ListProfiles() ([]model.Profile, error) {
	if m.ListProfilesErr != nil {
		return nil, m.ListProfilesErr
	}

	return m.Profiles, nil
}

// ListWorkspaces implements ClientInterface.
func (m *MockClient) ListWorkspaces() ([]model.Workspace, error) {
	if m.ListWorkspacesErr != nil {
		return nil, m.ListWorkspacesErr
	}

	return m.Workspaces, nil
}

// GetActiveWorkspace implements ClientInterface.
func (m *MockClient) GetActiveWorkspace() (*model.Workspace, error) {
	if m.GetActiveWorkspaceErr != nil {
		return nil, m.GetActiveWorkspaceErr
	}

	return m.ActiveWorkspace, nil
}

// SaveWorkspace implements ClientInterface.
func (m *MockClient) SaveWorkspace(workspace *model.Workspace) error {
	m.SaveWorkspaceCalled = true

	m.SavedWorkspace = workspace
	if m.SaveWorkspaceErr != nil {
		return m.SaveWorkspaceErr
	}

	m.Workspaces = append(m.Workspaces, *workspace)

	return nil
}

// GetConfig implements ClientInterface.
func (m *MockClient) GetConfig() (*model.Config, error) {
	if m.GetConfigErr != nil {
		return nil, m.GetConfigErr
	}

	return m.Config, nil
}

// withMockClient sets up the clientFactory to return the mock client,
// and returns a cleanup function to restore the original factory.
func withMockClient(mock *MockClient) func() {
	original := clientFactory
	clientFactory = func() (ClientInterface, error) {
		return mock, nil
	}

	return func() {
		clientFactory = original
	}
}

// withMockClientError sets up the clientFactory to return an error.
func withMockClientError(err error) func() {
	original := clientFactory
	clientFactory = func() (ClientInterface, error) {
		return nil, err
	}

	return func() {
		clientFactory = original
	}
}
