package grpc

import (
	v1 "github.com/inovacc/clonr/internal/api/v1"
	"github.com/inovacc/clonr/internal/mapper"
	"github.com/inovacc/clonr/internal/model"
)

// Re-export mapper functions for backward compatibility within the grpc package.
// These delegate to the shared mapper package to avoid code duplication.

// ModelToProtoRepository converts a model.Repository to a proto Repository
func ModelToProtoRepository(repo *model.Repository) *v1.Repository {
	return mapper.ModelToProtoRepository(repo)
}

// ProtoToModelRepository converts a proto Repository to a model.Repository
func ProtoToModelRepository(protoRepo *v1.Repository) model.Repository {
	return mapper.ProtoToModelRepository(protoRepo)
}

// ModelToProtoConfig converts a model.Config to a proto Config
func ModelToProtoConfig(cfg *model.Config) *v1.Config {
	return mapper.ModelToProtoConfig(cfg)
}

// ProtoToModelConfig converts a proto Config to a model.Config
func ProtoToModelConfig(protoCfg *v1.Config) *model.Config {
	return mapper.ProtoToModelConfig(protoCfg)
}

// ModelToProtoProfile converts a model.Profile to a proto Profile
func ModelToProtoProfile(profile *model.Profile) *v1.Profile {
	return mapper.ModelToProtoProfile(profile)
}

// ProtoToModelProfile converts a proto Profile to a model.Profile
func ProtoToModelProfile(protoProfile *v1.Profile) *model.Profile {
	return mapper.ProtoToModelProfile(protoProfile)
}

// ModelToProtoWorkspace converts a model.Workspace to a proto Workspace
func ModelToProtoWorkspace(workspace *model.Workspace) *v1.Workspace {
	return mapper.ModelToProtoWorkspace(workspace)
}

// ProtoToModelWorkspace converts a proto Workspace to a model.Workspace
func ProtoToModelWorkspace(protoWorkspace *v1.Workspace) *model.Workspace {
	return mapper.ProtoToModelWorkspace(protoWorkspace)
}

// ModelToProtoDockerProfile converts a model.DockerProfile to a proto DockerProfile
func ModelToProtoDockerProfile(profile *model.DockerProfile) *v1.DockerProfile {
	return mapper.ModelToProtoDockerProfile(profile)
}

// ProtoToModelDockerProfile converts a proto DockerProfile to a model.DockerProfile
func ProtoToModelDockerProfile(protoProfile *v1.DockerProfile) *model.DockerProfile {
	return mapper.ProtoToModelDockerProfile(protoProfile)
}
