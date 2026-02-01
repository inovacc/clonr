package grpc

import (
	"context"
	"net/url"

	"github.com/inovacc/clonr/internal/store"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the ClonrServiceServer interface
type Service struct {
	v1.UnimplementedClonrServiceServer

	db store.Store
}

// NewService creates a new gRPC service instance
func NewService(db store.Store) *Service {
	return &Service{db: db}
}

// Ping verifies database connectivity
func (s *Service) Ping(ctx context.Context, req *v1.Empty) (*v1.Empty, error) {
	if err := s.db.Ping(); err != nil {
		return nil, status.Errorf(codes.Internal, "database ping failed: %v", err)
	}

	return &v1.Empty{}, nil
}

// SaveRepo saves a repository to the database
func (s *Service) SaveRepo(ctx context.Context, req *v1.SaveRepoRequest) (*v1.SaveRepoResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	if req.GetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	u, err := url.Parse(req.GetUrl())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	if err := s.db.SaveRepoWithWorkspace(u, req.GetPath(), req.GetWorkspace()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save repository: %v", err)
	}

	return &v1.SaveRepoResponse{Success: true}, nil
}

// RepoExistsByURL checks if a repository exists by URL
func (s *Service) RepoExistsByURL(ctx context.Context, req *v1.RepoExistsByURLRequest) (*v1.RepoExistsByURLResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	u, err := url.Parse(req.GetUrl())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	exists, err := s.db.RepoExistsByURL(u)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check repository existence: %v", err)
	}

	return &v1.RepoExistsByURLResponse{Exists: exists}, nil
}

// RepoExistsByPath checks if a repository exists by path
func (s *Service) RepoExistsByPath(ctx context.Context, req *v1.RepoExistsByPathRequest) (*v1.RepoExistsByPathResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	exists, err := s.db.RepoExistsByPath(req.GetPath())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check repository existence: %v", err)
	}

	return &v1.RepoExistsByPathResponse{Exists: exists}, nil
}

// InsertRepoIfNotExists inserts a repository if it doesn't already exist
func (s *Service) InsertRepoIfNotExists(ctx context.Context, req *v1.InsertRepoIfNotExistsRequest) (*v1.InsertRepoIfNotExistsResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" && req.GetPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "url or path is required")
	}

	var (
		u   *url.URL
		err error
	)

	if req.GetUrl() != "" {
		u, err = url.Parse(req.GetUrl())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
		}
	}

	if err := s.db.InsertRepoIfNotExists(u, req.GetPath()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert repository: %v", err)
	}

	return &v1.InsertRepoIfNotExistsResponse{Inserted: true}, nil
}

// GetAllRepos retrieves all repositories
func (s *Service) GetAllRepos(ctx context.Context, req *v1.GetAllReposRequest) (*v1.GetAllReposResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	repos, err := s.db.GetAllRepos()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get repositories: %v", err)
	}

	protoRepos := make([]*v1.Repository, len(repos))
	for i, repo := range repos {
		protoRepos[i] = ModelToProtoRepository(&repo)
	}

	return &v1.GetAllReposResponse{Repositories: protoRepos}, nil
}

// GetRepos retrieves repositories with optional filtering
func (s *Service) GetRepos(ctx context.Context, req *v1.GetReposRequest) (*v1.GetReposResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	repos, err := s.db.GetRepos(req.GetWorkspace(), req.GetFavoritesOnly())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get repositories: %v", err)
	}

	protoRepos := make([]*v1.Repository, len(repos))
	for i, repo := range repos {
		protoRepos[i] = ModelToProtoRepository(&repo)
	}

	return &v1.GetReposResponse{Repositories: protoRepos}, nil
}

// SetFavoriteByURL marks or unmarks a repository as favorite
func (s *Service) SetFavoriteByURL(ctx context.Context, req *v1.SetFavoriteRequest) (*v1.SetFavoriteResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	// Validate URL format (per guide - consistency)
	if _, err := url.Parse(req.GetUrl()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	if err := s.db.SetFavoriteByURL(req.GetUrl(), req.GetFavorite()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set favorite: %v", err)
	}

	return &v1.SetFavoriteResponse{Success: true}, nil
}

// UpdateRepoTimestamp updates the timestamp for a repository
func (s *Service) UpdateRepoTimestamp(ctx context.Context, req *v1.UpdateRepoTimestampRequest) (*v1.UpdateRepoTimestampResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	if err := s.db.UpdateRepoTimestamp(req.GetUrl()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update timestamp: %v", err)
	}

	return &v1.UpdateRepoTimestampResponse{Success: true}, nil
}

// RemoveRepoByURL removes a repository by URL
func (s *Service) RemoveRepoByURL(ctx context.Context, req *v1.RemoveRepoByURLRequest) (*v1.RemoveRepoByURLResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	u, err := url.Parse(req.GetUrl())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	if err := s.db.RemoveRepoByURL(u); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove repository: %v", err)
	}

	return &v1.RemoveRepoByURLResponse{Success: true}, nil
}

// GetConfig retrieves the application configuration
func (s *Service) GetConfig(ctx context.Context, req *v1.GetConfigRequest) (*v1.GetConfigResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	cfg, err := s.db.GetConfig()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get configuration: %v", err)
	}

	return &v1.GetConfigResponse{Config: ModelToProtoConfig(cfg)}, nil
}

// SaveConfig saves the application configuration
func (s *Service) SaveConfig(ctx context.Context, req *v1.SaveConfigRequest) (*v1.SaveConfigResponse, error) {
	// Check for context cancellation (per guide)
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetConfig() == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}

	cfg := ProtoToModelConfig(req.GetConfig())
	if err := s.db.SaveConfig(cfg); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save configuration: %v", err)
	}

	return &v1.SaveConfigResponse{Success: true}, nil
}

// SaveProfile saves or updates a profile
func (s *Service) SaveProfile(ctx context.Context, req *v1.SaveProfileRequest) (*v1.SaveProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetProfile() == nil {
		return nil, status.Error(codes.InvalidArgument, "profile is required")
	}

	if req.GetProfile().GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "profile name is required")
	}

	profile := ProtoToModelProfile(req.GetProfile())
	if err := s.db.SaveProfile(profile); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save profile: %v", err)
	}

	return &v1.SaveProfileResponse{Success: true}, nil
}

// GetProfile retrieves a profile by name
func (s *Service) GetProfile(ctx context.Context, req *v1.GetProfileRequest) (*v1.GetProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	profile, err := s.db.GetProfile(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get profile: %v", err)
	}

	if profile == nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	return &v1.GetProfileResponse{Profile: ModelToProtoProfile(profile)}, nil
}

// GetActiveProfile retrieves the currently active profile
func (s *Service) GetActiveProfile(ctx context.Context, _ *v1.GetActiveProfileRequest) (*v1.GetActiveProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	profile, err := s.db.GetActiveProfile()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get active profile: %v", err)
	}

	return &v1.GetActiveProfileResponse{Profile: ModelToProtoProfile(profile)}, nil
}

// SetActiveProfile sets the active profile by name
func (s *Service) SetActiveProfile(ctx context.Context, req *v1.SetActiveProfileRequest) (*v1.SetActiveProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.db.SetActiveProfile(req.GetName()); err != nil {
		if err.Error() == "profile not found" {
			return nil, status.Error(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to set active profile: %v", err)
	}

	return &v1.SetActiveProfileResponse{Success: true}, nil
}

// ListProfiles retrieves all profiles
func (s *Service) ListProfiles(ctx context.Context, _ *v1.ListProfilesRequest) (*v1.ListProfilesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	profiles, err := s.db.ListProfiles()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list profiles: %v", err)
	}

	protoProfiles := make([]*v1.Profile, len(profiles))
	for i, profile := range profiles {
		protoProfiles[i] = ModelToProtoProfile(&profile)
	}

	return &v1.ListProfilesResponse{Profiles: protoProfiles}, nil
}

// DeleteProfile removes a profile by name
func (s *Service) DeleteProfile(ctx context.Context, req *v1.DeleteProfileRequest) (*v1.DeleteProfileResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.db.DeleteProfile(req.GetName()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete profile: %v", err)
	}

	return &v1.DeleteProfileResponse{Success: true}, nil
}

// ProfileExists checks if a profile exists by name
func (s *Service) ProfileExists(ctx context.Context, req *v1.ProfileExistsRequest) (*v1.ProfileExistsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	exists, err := s.db.ProfileExists(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check profile existence: %v", err)
	}

	return &v1.ProfileExistsResponse{Exists: exists}, nil
}

// SaveWorkspace saves or updates a workspace
func (s *Service) SaveWorkspace(ctx context.Context, req *v1.SaveWorkspaceRequest) (*v1.SaveWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetWorkspace() == nil {
		return nil, status.Error(codes.InvalidArgument, "workspace is required")
	}

	if req.GetWorkspace().GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace name is required")
	}

	workspace := ProtoToModelWorkspace(req.GetWorkspace())
	if err := s.db.SaveWorkspace(workspace); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save workspace: %v", err)
	}

	return &v1.SaveWorkspaceResponse{Success: true}, nil
}

// GetWorkspace retrieves a workspace by name
func (s *Service) GetWorkspace(ctx context.Context, req *v1.GetWorkspaceRequest) (*v1.GetWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	workspace, err := s.db.GetWorkspace(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace: %v", err)
	}

	if workspace == nil {
		return nil, status.Error(codes.NotFound, "workspace not found")
	}

	return &v1.GetWorkspaceResponse{Workspace: ModelToProtoWorkspace(workspace)}, nil
}

// GetActiveWorkspace retrieves the currently active workspace
func (s *Service) GetActiveWorkspace(ctx context.Context, _ *v1.GetActiveWorkspaceRequest) (*v1.GetActiveWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	workspace, err := s.db.GetActiveWorkspace()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get active workspace: %v", err)
	}

	return &v1.GetActiveWorkspaceResponse{Workspace: ModelToProtoWorkspace(workspace)}, nil
}

// SetActiveWorkspace sets the active workspace by name
func (s *Service) SetActiveWorkspace(ctx context.Context, req *v1.SetActiveWorkspaceRequest) (*v1.SetActiveWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.db.SetActiveWorkspace(req.GetName()); err != nil {
		if err.Error() == "workspace not found" {
			return nil, status.Error(codes.NotFound, "workspace not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to set active workspace: %v", err)
	}

	return &v1.SetActiveWorkspaceResponse{Success: true}, nil
}

// ListWorkspaces retrieves all workspaces
func (s *Service) ListWorkspaces(ctx context.Context, _ *v1.ListWorkspacesRequest) (*v1.ListWorkspacesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	workspaces, err := s.db.ListWorkspaces()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workspaces: %v", err)
	}

	protoWorkspaces := make([]*v1.Workspace, len(workspaces))
	for i, workspace := range workspaces {
		protoWorkspaces[i] = ModelToProtoWorkspace(&workspace)
	}

	return &v1.ListWorkspacesResponse{Workspaces: protoWorkspaces}, nil
}

// DeleteWorkspace removes a workspace by name
func (s *Service) DeleteWorkspace(ctx context.Context, req *v1.DeleteWorkspaceRequest) (*v1.DeleteWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Check if workspace has repositories
	urls, err := s.db.GetReposByWorkspace(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check workspace repositories: %v", err)
	}

	if len(urls) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "workspace has %d repositories, move them first", len(urls))
	}

	if err := s.db.DeleteWorkspace(req.GetName()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workspace: %v", err)
	}

	return &v1.DeleteWorkspaceResponse{Success: true}, nil
}

// WorkspaceExists checks if a workspace exists by name
func (s *Service) WorkspaceExists(ctx context.Context, req *v1.WorkspaceExistsRequest) (*v1.WorkspaceExistsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	exists, err := s.db.WorkspaceExists(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check workspace existence: %v", err)
	}

	return &v1.WorkspaceExistsResponse{Exists: exists}, nil
}

// GetReposByWorkspace retrieves all repository URLs in a workspace
func (s *Service) GetReposByWorkspace(ctx context.Context, req *v1.GetReposByWorkspaceRequest) (*v1.GetReposByWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetWorkspace() == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace is required")
	}

	urls, err := s.db.GetReposByWorkspace(req.GetWorkspace())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get repositories by workspace: %v", err)
	}

	return &v1.GetReposByWorkspaceResponse{Urls: urls}, nil
}

// UpdateRepoWorkspace updates the workspace for a repository
func (s *Service) UpdateRepoWorkspace(ctx context.Context, req *v1.UpdateRepoWorkspaceRequest) (*v1.UpdateRepoWorkspaceResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	if req.GetWorkspace() == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace is required")
	}

	// Validate URL format
	if _, err := url.Parse(req.GetUrl()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid URL: %v", err)
	}

	if err := s.db.UpdateRepoWorkspace(req.GetUrl(), req.GetWorkspace()); err != nil {
		if err.Error() == "repository not found" {
			return nil, status.Error(codes.NotFound, "repository not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to update repository workspace: %v", err)
	}

	return &v1.UpdateRepoWorkspaceResponse{Success: true}, nil
}
