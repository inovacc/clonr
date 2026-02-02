package grpc

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	v1 "github.com/inovacc/clonr/internal/api/v1"
	"github.com/inovacc/clonr/internal/mapper"
	"github.com/inovacc/clonr/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

var (
	once      sync.Once
	client    *Client
	errClient error
)

// Client wraps the gRPC client and provides methods matching the store.Store interface
type Client struct {
	conn    *grpc.ClientConn
	service v1.ClonrServiceClient
	timeout time.Duration
}

// GetClient returns the singleton gRPC client instance
func GetClient() (*Client, error) {
	once.Do(lazyLoad)

	if errClient != nil {
		return nil, errClient
	}

	return client, nil
}

func lazyLoad() {
	addr := discoverServerAddress()

	// Use grpc.NewClient (v1.78.0+) instead of deprecated DialContext
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errClient = fmt.Errorf("failed to create gRPC client: %w", err)
		return
	}

	// Perform health check to trigger connection
	healthClient := healthpb.NewHealthClient(conn)

	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	resp, err := healthClient.Check(healthCtx, &healthpb.HealthCheckRequest{})
	if err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		_ = conn.Close()

		// Server not running - try to start one automatically
		if startErr := startOnDemandServer(defaultServerPort); startErr != nil {
			errClient = fmt.Errorf("failed to start on-demand server: %w", startErr)
			return
		}

		// Wait for the server to be ready
		addr = fmt.Sprintf("localhost:%d", defaultServerPort)
		if waitErr := waitForServer(addr); waitErr != nil {
			errClient = fmt.Errorf("server started but not ready: %w", waitErr)
			return
		}

		// Reconnect to the now-running server
		conn, err = grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			errClient = fmt.Errorf("failed to connect to started server: %w", err)
			return
		}
	}

	client = &Client{
		conn:    conn,
		service: v1.NewClonrServiceClient(conn),
		timeout: 30 * time.Second,
	}
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// Ping verifies the connection to the server
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.service.Ping(ctx, &v1.Empty{})

	return handleGRPCError(err)
}

// SaveRepo saves a repository to the database via gRPC
func (c *Client) SaveRepo(u *url.URL, path string) error {
	return c.SaveRepoWithWorkspace(u, path, "")
}

// SaveRepoWithWorkspace saves a repository with workspace to the database via gRPC
func (c *Client) SaveRepoWithWorkspace(u *url.URL, path string, workspace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SaveRepo(ctx, &v1.SaveRepoRequest{
		Url:       u.String(),
		Path:      path,
		Workspace: workspace,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// RepoExistsByURL checks if a repository exists by URL
func (c *Client) RepoExistsByURL(u *url.URL) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.RepoExistsByURL(ctx, &v1.RepoExistsByURLRequest{
		Url: u.String(),
	})
	if err != nil {
		return false, handleGRPCError(err)
	}

	return resp.GetExists(), nil
}

// RepoExistsByPath checks if a repository exists by path
func (c *Client) RepoExistsByPath(path string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.RepoExistsByPath(ctx, &v1.RepoExistsByPathRequest{
		Path: path,
	})
	if err != nil {
		return false, handleGRPCError(err)
	}

	return resp.GetExists(), nil
}

// InsertRepoIfNotExists inserts a repository if it doesn't exist
func (c *Client) InsertRepoIfNotExists(u *url.URL, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	urlStr := ""
	if u != nil {
		urlStr = u.String()
	}

	resp, err := c.service.InsertRepoIfNotExists(ctx, &v1.InsertRepoIfNotExistsRequest{
		Url:  urlStr,
		Path: path,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetInserted() {
		return fmt.Errorf("repository already exists")
	}

	return nil
}

// GetAllRepos retrieves all repositories
func (c *Client) GetAllRepos() ([]model.Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetAllRepos(ctx, &v1.GetAllReposRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	repos := make([]model.Repository, len(resp.GetRepositories()))
	for i, pr := range resp.GetRepositories() {
		repos[i] = mapper.ProtoToModelRepository(pr)
	}

	return repos, nil
}

// GetRepos retrieves repositories with optional filtering
func (c *Client) GetRepos(workspace string, favoritesOnly bool) ([]model.Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetRepos(ctx, &v1.GetReposRequest{
		Workspace:     workspace,
		FavoritesOnly: favoritesOnly,
	})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	repos := make([]model.Repository, len(resp.GetRepositories()))
	for i, pr := range resp.GetRepositories() {
		repos[i] = mapper.ProtoToModelRepository(pr)
	}

	return repos, nil
}

// SetFavoriteByURL marks or unmarks a repository as favorite
func (c *Client) SetFavoriteByURL(urlStr string, fav bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SetFavoriteByURL(ctx, &v1.SetFavoriteRequest{
		Url:      urlStr,
		Favorite: fav,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// UpdateRepoTimestamp updates the timestamp for a repository
func (c *Client) UpdateRepoTimestamp(urlStr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.UpdateRepoTimestamp(ctx, &v1.UpdateRepoTimestampRequest{
		Url: urlStr,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// RemoveRepoByURL removes a repository by URL
func (c *Client) RemoveRepoByURL(u *url.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.RemoveRepoByURL(ctx, &v1.RemoveRepoByURLRequest{
		Url: u.String(),
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// GetConfig retrieves the application configuration
func (c *Client) GetConfig() (*model.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetConfig(ctx, &v1.GetConfigRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	if resp.GetConfig() == nil {
		return nil, fmt.Errorf("no configuration returned")
	}

	return mapper.ProtoToModelConfig(resp.GetConfig()), nil
}

// SaveConfig saves the application configuration
func (c *Client) SaveConfig(cfg *model.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SaveConfig(ctx, &v1.SaveConfigRequest{
		Config: mapper.ModelToProtoConfig(cfg),
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// handleGRPCError converts gRPC errors to user-friendly messages
func handleGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("unknown error: %w", err)
	}

	//nolint:exhaustive // default case handles remaining codes
	switch st.Code() {
	case codes.InvalidArgument:
		return fmt.Errorf("invalid input: %s", st.Message())
	case codes.AlreadyExists:
		return fmt.Errorf("already exists: %s", st.Message())
	case codes.NotFound:
		return fmt.Errorf("not found: %s", st.Message())
	case codes.Unavailable:
		return fmt.Errorf("server unavailable - is clonr-server running?\nStart it with: clonr-server start")
	case codes.DeadlineExceeded:
		return fmt.Errorf("request timeout: %s", st.Message())
	case codes.Canceled:
		return fmt.Errorf("request canceled: %s", st.Message())
	default:
		return fmt.Errorf("server error: %s", st.Message())
	}
}

// SaveProfile saves or updates a profile via gRPC
func (c *Client) SaveProfile(profile *model.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SaveProfile(ctx, &v1.SaveProfileRequest{
		Profile: mapper.ModelToProtoProfile(profile),
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// GetProfile retrieves a profile by name
func (c *Client) GetProfile(name string) (*model.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetProfile(ctx, &v1.GetProfileRequest{
		Name: name,
	})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	return mapper.ProtoToModelProfile(resp.GetProfile()), nil
}

// GetActiveProfile retrieves the currently active profile
func (c *Client) GetActiveProfile() (*model.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetActiveProfile(ctx, &v1.GetActiveProfileRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	return mapper.ProtoToModelProfile(resp.GetProfile()), nil
}

// SetActiveProfile sets the active profile by name
func (c *Client) SetActiveProfile(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SetActiveProfile(ctx, &v1.SetActiveProfileRequest{
		Name: name,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// ListProfiles retrieves all profiles
func (c *Client) ListProfiles() ([]model.Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.ListProfiles(ctx, &v1.ListProfilesRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	profiles := make([]model.Profile, len(resp.GetProfiles()))
	for i, pr := range resp.GetProfiles() {
		profiles[i] = *mapper.ProtoToModelProfile(pr)
	}

	return profiles, nil
}

// DeleteProfile removes a profile by name
func (c *Client) DeleteProfile(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.DeleteProfile(ctx, &v1.DeleteProfileRequest{
		Name: name,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// ProfileExists checks if a profile exists by name
func (c *Client) ProfileExists(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.ProfileExists(ctx, &v1.ProfileExistsRequest{
		Name: name,
	})
	if err != nil {
		return false, handleGRPCError(err)
	}

	return resp.GetExists(), nil
}

// SaveWorkspace saves or updates a workspace via gRPC
func (c *Client) SaveWorkspace(workspace *model.Workspace) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SaveWorkspace(ctx, &v1.SaveWorkspaceRequest{
		Workspace: mapper.ModelToProtoWorkspace(workspace),
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// GetWorkspace retrieves a workspace by name
func (c *Client) GetWorkspace(name string) (*model.Workspace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetWorkspace(ctx, &v1.GetWorkspaceRequest{
		Name: name,
	})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	return mapper.ProtoToModelWorkspace(resp.GetWorkspace()), nil
}

// GetActiveWorkspace retrieves the currently active workspace
func (c *Client) GetActiveWorkspace() (*model.Workspace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetActiveWorkspace(ctx, &v1.GetActiveWorkspaceRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	return mapper.ProtoToModelWorkspace(resp.GetWorkspace()), nil
}

// SetActiveWorkspace sets the active workspace by name
func (c *Client) SetActiveWorkspace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.SetActiveWorkspace(ctx, &v1.SetActiveWorkspaceRequest{
		Name: name,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// ListWorkspaces retrieves all workspaces
func (c *Client) ListWorkspaces() ([]model.Workspace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.ListWorkspaces(ctx, &v1.ListWorkspacesRequest{})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	workspaces := make([]model.Workspace, len(resp.GetWorkspaces()))
	for i, pw := range resp.GetWorkspaces() {
		workspaces[i] = *mapper.ProtoToModelWorkspace(pw)
	}

	return workspaces, nil
}

// DeleteWorkspace removes a workspace by name
func (c *Client) DeleteWorkspace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.DeleteWorkspace(ctx, &v1.DeleteWorkspaceRequest{
		Name: name,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

// WorkspaceExists checks if a workspace exists by name
func (c *Client) WorkspaceExists(name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.WorkspaceExists(ctx, &v1.WorkspaceExistsRequest{
		Name: name,
	})
	if err != nil {
		return false, handleGRPCError(err)
	}

	return resp.GetExists(), nil
}

// GetReposByWorkspace retrieves all repository URLs in a workspace
func (c *Client) GetReposByWorkspace(workspace string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.GetReposByWorkspace(ctx, &v1.GetReposByWorkspaceRequest{
		Workspace: workspace,
	})
	if err != nil {
		return nil, handleGRPCError(err)
	}

	return resp.GetUrls(), nil
}

// UpdateRepoWorkspace updates the workspace for a repository
func (c *Client) UpdateRepoWorkspace(urlStr string, workspace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.service.UpdateRepoWorkspace(ctx, &v1.UpdateRepoWorkspaceRequest{
		Url:       urlStr,
		Workspace: workspace,
	})
	if err != nil {
		return handleGRPCError(err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("operation failed")
	}

	return nil
}
