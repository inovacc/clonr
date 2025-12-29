package grpcserver

import (
	"context"
	"net/url"

	"github.com/inovacc/clonr/internal/database"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the ClonrServiceServer interface
type Service struct {
	v1.UnimplementedClonrServiceServer

	db database.Store
}

// NewService creates a new gRPC service instance
func NewService(db database.Store) *Service {
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

	if err := s.db.SaveRepo(u, req.GetPath()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save repository: %v", err)
	}

	return &v1.SaveRepoResponse{Success: true}, nil
}

// RepoExistsByURL checks if a repository exists by URL
func (s *Service) RepoExistsByURL(ctx context.Context, req *v1.RepoExistsByURLRequest) (*v1.RepoExistsByURLResponse, error) {
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
	repos, err := s.db.GetRepos(req.GetFavoritesOnly())
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
	if req.GetUrl() == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	if err := s.db.SetFavoriteByURL(req.GetUrl(), req.GetFavorite()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set favorite: %v", err)
	}

	return &v1.SetFavoriteResponse{Success: true}, nil
}

// UpdateRepoTimestamp updates the timestamp for a repository
func (s *Service) UpdateRepoTimestamp(ctx context.Context, req *v1.UpdateRepoTimestampRequest) (*v1.UpdateRepoTimestampResponse, error) {
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
	cfg, err := s.db.GetConfig()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get configuration: %v", err)
	}

	return &v1.GetConfigResponse{Config: ModelToProtoConfig(cfg)}, nil
}

// SaveConfig saves the application configuration
func (s *Service) SaveConfig(ctx context.Context, req *v1.SaveConfigRequest) (*v1.SaveConfigResponse, error) {
	if req.GetConfig() == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}

	cfg := ProtoToModelConfig(req.GetConfig())
	if err := s.db.SaveConfig(cfg); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save configuration: %v", err)
	}

	return &v1.SaveConfigResponse{Success: true}, nil
}
