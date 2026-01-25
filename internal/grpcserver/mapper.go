package grpcserver

import (
	"github.com/inovacc/clonr/internal/model"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
		Favorite:    protoRepo.GetFavorite(),
		ClonedAt:    protoRepo.GetClonedAt().AsTime(),
		UpdatedAt:   protoRepo.GetUpdatedAt().AsTime(),
		LastChecked: protoRepo.GetLastChecked().AsTime(),
	}
}

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
		Active:         profile.Active,
		EncryptedToken: profile.EncryptedToken,
		CreatedAt:      timestamppb.New(profile.CreatedAt),
		LastUsedAt:     timestamppb.New(profile.LastUsedAt),
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
		Active:         protoProfile.GetActive(),
		EncryptedToken: protoProfile.GetEncryptedToken(),
		CreatedAt:      protoProfile.GetCreatedAt().AsTime(),
		LastUsedAt:     protoProfile.GetLastUsedAt().AsTime(),
	}
}
