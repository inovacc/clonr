// Package mapper provides conversion functions between model and proto types.
// This package is shared between server and client to avoid code duplication.
package mapper

import (
	v1 "github.com/inovacc/clonr/internal/api/v1"
	"github.com/inovacc/clonr/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Repository conversions

// ModelToProtoRepository converts a model.Repository to a proto Repository
func ModelToProtoRepository(repo *model.Repository) *v1.Repository {
	if repo == nil {
		return nil
	}

	return &v1.Repository{
		Id:          uint32(repo.ID),
		Uid:         repo.UID,
		Url:         repo.URL,
		Path:        repo.Path,
		Workspace:   repo.Workspace,
		Favorite:    repo.Favorite,
		ClonedAt:    timestamppb.New(repo.ClonedAt),
		UpdatedAt:   timestamppb.New(repo.UpdatedAt),
		LastChecked: timestamppb.New(repo.LastChecked),
	}
}

// ProtoToModelRepository converts a proto Repository to a model.Repository
func ProtoToModelRepository(protoRepo *v1.Repository) model.Repository {
	if protoRepo == nil {
		return model.Repository{}
	}

	return model.Repository{
		ID:          uint(protoRepo.GetId()),
		UID:         protoRepo.GetUid(),
		URL:         protoRepo.GetUrl(),
		Path:        protoRepo.GetPath(),
		Workspace:   protoRepo.GetWorkspace(),
		Favorite:    protoRepo.GetFavorite(),
		ClonedAt:    protoRepo.GetClonedAt().AsTime(),
		UpdatedAt:   protoRepo.GetUpdatedAt().AsTime(),
		LastChecked: protoRepo.GetLastChecked().AsTime(),
	}
}

// Config conversions

// ModelToProtoConfig converts a model.Config to a proto Config
func ModelToProtoConfig(cfg *model.Config) *v1.Config {
	if cfg == nil {
		return nil
	}

	return &v1.Config{
		DefaultCloneDir: cfg.DefaultCloneDir,
		Editor:          cfg.Editor,
		Terminal:        cfg.Terminal,
		MonitorInterval: int32(cfg.MonitorInterval),
		ServerPort:      int32(cfg.ServerPort),
	}
}

// ProtoToModelConfig converts a proto Config to a model.Config
func ProtoToModelConfig(protoCfg *v1.Config) *model.Config {
	if protoCfg == nil {
		return nil
	}

	return &model.Config{
		DefaultCloneDir: protoCfg.GetDefaultCloneDir(),
		Editor:          protoCfg.GetEditor(),
		Terminal:        protoCfg.GetTerminal(),
		MonitorInterval: int(protoCfg.GetMonitorInterval()),
		ServerPort:      int(protoCfg.GetServerPort()),
	}
}

// Profile conversions

// ModelToProtoProfile converts a model.Profile to a proto Profile
func ModelToProtoProfile(profile *model.Profile) *v1.Profile {
	if profile == nil {
		return nil
	}

	return &v1.Profile{
		Name:           profile.Name,
		Host:           profile.Host,
		User:           profile.User,
		TokenStorage:   string(profile.TokenStorage),
		Scopes:         profile.Scopes,
		Active:         profile.Default, // Map Default to Active for proto compatibility
		EncryptedToken: profile.EncryptedToken,
		CreatedAt:      timestamppb.New(profile.CreatedAt),
		LastUsedAt:     timestamppb.New(profile.LastUsedAt),
		Workspace:      profile.Workspace,
	}
}

// ProtoToModelProfile converts a proto Profile to a model.Profile
func ProtoToModelProfile(protoProfile *v1.Profile) *model.Profile {
	if protoProfile == nil {
		return nil
	}

	return &model.Profile{
		Name:           protoProfile.GetName(),
		Host:           protoProfile.GetHost(),
		User:           protoProfile.GetUser(),
		TokenStorage:   model.TokenStorage(protoProfile.GetTokenStorage()),
		Scopes:         protoProfile.GetScopes(),
		Default:        protoProfile.GetActive(), // Map Active to Default from proto
		EncryptedToken: protoProfile.GetEncryptedToken(),
		CreatedAt:      protoProfile.GetCreatedAt().AsTime(),
		LastUsedAt:     protoProfile.GetLastUsedAt().AsTime(),
		Workspace:      protoProfile.GetWorkspace(),
	}
}

// Workspace conversions

// ModelToProtoWorkspace converts a model.Workspace to a proto Workspace
func ModelToProtoWorkspace(workspace *model.Workspace) *v1.Workspace {
	if workspace == nil {
		return nil
	}

	return &v1.Workspace{
		Name:        workspace.Name,
		Description: workspace.Description,
		Path:        workspace.Path,
		Active:      workspace.Active,
		CreatedAt:   timestamppb.New(workspace.CreatedAt),
		UpdatedAt:   timestamppb.New(workspace.UpdatedAt),
	}
}

// ProtoToModelWorkspace converts a proto Workspace to a model.Workspace
func ProtoToModelWorkspace(protoWorkspace *v1.Workspace) *model.Workspace {
	if protoWorkspace == nil {
		return nil
	}

	return &model.Workspace{
		Name:        protoWorkspace.GetName(),
		Description: protoWorkspace.GetDescription(),
		Path:        protoWorkspace.GetPath(),
		Active:      protoWorkspace.GetActive(),
		CreatedAt:   protoWorkspace.GetCreatedAt().AsTime(),
		UpdatedAt:   protoWorkspace.GetUpdatedAt().AsTime(),
	}
}

// Docker Profile conversions

// ModelToProtoDockerProfile converts a model.DockerProfile to a proto DockerProfile
func ModelToProtoDockerProfile(profile *model.DockerProfile) *v1.DockerProfile {
	if profile == nil {
		return nil
	}

	return &v1.DockerProfile{
		Name:           profile.Name,
		Registry:       profile.Registry,
		Username:       profile.Username,
		EncryptedToken: profile.EncryptedToken,
		TokenStorage:   string(profile.TokenStorage),
		CreatedAt:      timestamppb.New(profile.CreatedAt),
		LastUsedAt:     timestamppb.New(profile.LastUsedAt),
	}
}

// ProtoToModelDockerProfile converts a proto DockerProfile to a model.DockerProfile
func ProtoToModelDockerProfile(protoProfile *v1.DockerProfile) *model.DockerProfile {
	if protoProfile == nil {
		return nil
	}

	return &model.DockerProfile{
		Name:           protoProfile.GetName(),
		Registry:       protoProfile.GetRegistry(),
		Username:       protoProfile.GetUsername(),
		EncryptedToken: protoProfile.GetEncryptedToken(),
		TokenStorage:   model.TokenStorage(protoProfile.GetTokenStorage()),
		CreatedAt:      protoProfile.GetCreatedAt().AsTime(),
		LastUsedAt:     protoProfile.GetLastUsedAt().AsTime(),
	}
}
