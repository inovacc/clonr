package sqlite

import (
	"encoding/json"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store/sqlite/sqlc"
)

// Helper functions for dereferencing pointers
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func derefInt64ToBool(i *int64) bool {
	return i != nil && *i == 1
}

// sqlcRepoToModel converts a sqlc Repository to a model.Repository.
func sqlcRepoToModel(row sqlc.Repository) *model.Repository {
	return &model.Repository{
		ID:          uint(row.ID),
		UID:         row.Uid,
		URL:         row.Url,
		Path:        row.Path,
		Workspace:   derefString(row.Workspace),
		Favorite:    derefInt64ToBool(row.Favorite),
		ClonedAt:    row.ClonedAt,
		UpdatedAt:   row.UpdatedAt,
		LastChecked: row.LastChecked,
	}
}

// sqlcProfileToModel converts a sqlc Profile to a model.Profile.
func sqlcProfileToModel(row sqlc.Profile) *model.Profile {
	var scopes []string
	if row.Scopes != nil && *row.Scopes != "" {
		_ = json.Unmarshal([]byte(*row.Scopes), &scopes)
	}

	var notifyChannels []model.NotifyChannel
	if row.NotifyChannels != nil && *row.NotifyChannels != "" {
		_ = json.Unmarshal([]byte(*row.NotifyChannels), &notifyChannels)
	}

	return &model.Profile{
		Name:           row.Name,
		Host:           derefString(row.Host),
		User:           derefString(row.Username),
		TokenStorage:   model.TokenStorage(derefString(row.TokenStorage)),
		Scopes:         scopes,
		Default:        derefInt64ToBool(row.IsDefault),
		EncryptedToken: row.EncryptedToken,
		Workspace:      derefString(row.Workspace),
		NotifyChannels: notifyChannels,
		CreatedAt:      row.CreatedAt,
		LastUsedAt:     row.LastUsedAt,
	}
}

// sqlcWorkspaceToModel converts a sqlc Workspace to a model.Workspace.
func sqlcWorkspaceToModel(row sqlc.Workspace) *model.Workspace {
	return &model.Workspace{
		Name:        row.Name,
		Description: derefString(row.Description),
		Path:        derefString(row.Path),
		Active:      derefInt64ToBool(row.IsActive),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// sqlcConnectionToModel converts a sqlc StandaloneConnection to a standalone.StandaloneConnection.
func sqlcConnectionToModel(row sqlc.StandaloneConnection) *standalone.StandaloneConnection {
	var syncedItems standalone.SyncStats
	if row.SyncedItems != nil && *row.SyncedItems != "" {
		_ = json.Unmarshal([]byte(*row.SyncedItems), &syncedItems)
	}

	return &standalone.StandaloneConnection{
		Name:                  row.Name,
		InstanceID:            derefString(row.InstanceID),
		Host:                  row.Host,
		Port:                  int(derefInt64(row.Port)),
		APIKeyEncrypted:       row.ApiKeyEncrypted,
		RefreshTokenEncrypted: row.RefreshTokenEncrypted,
		LocalPasswordHash:     row.LocalPasswordHash,
		LocalSalt:             row.LocalSalt,
		SyncStatus:            derefString(row.SyncStatus),
		SyncedItems:           syncedItems,
		LastSync:              row.LastSync,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

// sqlcSyncedDataToModel converts a sqlc SyncedDatum to a standalone.SyncedData.
func sqlcSyncedDataToModel(row sqlc.SyncedDatum) *standalone.SyncedData {
	return &standalone.SyncedData{
		ID:             row.ID,
		ConnectionName: row.ConnectionName,
		InstanceID:     derefString(row.InstanceID),
		DataType:       row.DataType,
		Name:           row.Name,
		EncryptedData:  row.EncryptedData,
		Nonce:          row.Nonce,
		State:          standalone.SyncState(derefString(row.State)),
		Checksum:       derefString(row.Checksum),
		SyncedAt:       row.SyncedAt,
		DecryptedAt:    row.DecryptedAt,
	}
}

// sqlcRegisteredClientToModel converts a sqlc RegisteredClient to a standalone.RegisteredClient.
func sqlcRegisteredClientToModel(row sqlc.RegisteredClient) *standalone.RegisteredClient {
	var machineInfo standalone.MachineInfo
	if row.MachineInfo != nil && *row.MachineInfo != "" {
		_ = json.Unmarshal([]byte(*row.MachineInfo), &machineInfo)
	}

	return &standalone.RegisteredClient{
		ClientID:          row.ClientID,
		ClientName:        row.ClientName,
		MachineInfo:       machineInfo,
		EncryptionKeyHash: row.EncryptionKeyHash,
		EncryptionSalt:    row.EncryptionSalt,
		KeyHint:           derefString(row.KeyHint),
		Status:            derefString(row.Status),
		SyncCount:         int(derefInt64(row.SyncCount)),
		LastIP:            derefString(row.LastIp),
		RegisteredAt:      row.RegisteredAt,
		LastSeenAt:        row.LastSeenAt,
	}
}

// sqlcDockerProfileToModel converts a sqlc DockerProfile to a model.DockerProfile.
func sqlcDockerProfileToModel(row sqlc.DockerProfile) *model.DockerProfile {
	return &model.DockerProfile{
		Name:           row.Name,
		Registry:       derefString(row.Registry),
		Username:       row.Username,
		EncryptedToken: row.EncryptedToken,
		TokenStorage:   model.TokenStorage(derefString(row.TokenStorage)),
		CreatedAt:      row.CreatedAt,
		LastUsedAt:     row.LastUsedAt,
	}
}

// sqlcSlackConfigToModel converts a sqlc SlackConfig to a model.SlackConfig.
func sqlcSlackConfigToModel(row sqlc.SlackConfig) *model.SlackConfig {
	var events []model.SlackEventConfig
	if row.Events != nil && *row.Events != "" {
		_ = json.Unmarshal([]byte(*row.Events), &events)
	}

	return &model.SlackConfig{
		ID:                  int(row.ID),
		Enabled:             derefInt64ToBool(row.Enabled),
		WorkspaceID:         derefString(row.WorkspaceID),
		WorkspaceName:       derefString(row.WorkspaceName),
		EncryptedWebhookURL: row.EncryptedWebhookUrl,
		EncryptedBotToken:   row.EncryptedBotToken,
		DefaultChannel:      derefString(row.DefaultChannel),
		BotEnabled:          derefInt64ToBool(row.BotEnabled),
		Events:              events,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}
