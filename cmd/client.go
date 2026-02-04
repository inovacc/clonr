package cmd

import (
	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/model"
)

// ClientInterface defines the methods used by cmd package from the gRPC client.
// This interface allows for mocking in tests.
type ClientInterface interface {
	// Profile methods
	SetActiveProfile(name string) error
	GetProfile(name string) (*model.Profile, error)
	GetActiveProfile() (*model.Profile, error)
	ListProfiles() ([]model.Profile, error)

	// Workspace methods
	ListWorkspaces() ([]model.Workspace, error)
	GetActiveWorkspace() (*model.Workspace, error)
	SaveWorkspace(workspace *model.Workspace) error

	// Config methods
	GetConfig() (*model.Config, error)
}

// clientFactory is the function used to get the gRPC client.
// It can be overridden in tests to return a mock client.
var clientFactory = func() (ClientInterface, error) {
	return grpc.GetClient()
}

// getClient returns a ClientInterface instance.
// In production, this returns the real gRPC client.
// In tests, clientFactory can be replaced to return a mock.
func getClient() (ClientInterface, error) {
	return clientFactory()
}
