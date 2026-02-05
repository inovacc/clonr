package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store/sqlite/sqlc"
)

// ============================================================================
// Standalone Configuration Operations
// ============================================================================

func (s *Store) GetStandaloneConfig() (*standalone.StandaloneConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetStandaloneConfig(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	var capabilities []string
	if row.Capabilities != nil && *row.Capabilities != "" {
		_ = json.Unmarshal([]byte(*row.Capabilities), &capabilities)
	}

	config := &standalone.StandaloneConfig{
		Enabled:      derefInt64ToBool(row.Enabled),
		IsServer:     derefInt64ToBool(row.IsServer),
		InstanceID:   derefString(row.InstanceID),
		Port:         int(derefInt64(row.Port)),
		APIKeyHash:   row.ApiKeyHash,
		RefreshToken: row.RefreshToken,
		Salt:         row.Salt,
		Capabilities: capabilities,
		CreatedAt:    row.CreatedAt,
		ExpiresAt:    row.ExpiresAt,
	}

	return config, nil
}

func (s *Store) SaveStandaloneConfig(config *standalone.StandaloneConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	capabilitiesJSON, _ := json.Marshal(config.Capabilities)
	capabilitiesStr := string(capabilitiesJSON)

	enabled := int64(0)
	if config.Enabled {
		enabled = 1
	}

	isServer := int64(0)
	if config.IsServer {
		isServer = 1
	}

	return s.queries.UpsertStandaloneConfig(ctx, sqlc.UpsertStandaloneConfigParams{
		Enabled:      ptrInt64(enabled),
		IsServer:     ptrInt64(isServer),
		InstanceID:   ptrString(config.InstanceID),
		Port:         ptrInt64(int64(config.Port)),
		ApiKeyHash:   config.APIKeyHash,
		RefreshToken: config.RefreshToken,
		Salt:         config.Salt,
		Capabilities: &capabilitiesStr,
		ExpiresAt:    config.ExpiresAt,
	})
}

func (s *Store) DeleteStandaloneConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeleteStandaloneConfig(ctx)
}

// ============================================================================
// Standalone Clients Operations (Server Mode)
// ============================================================================

func (s *Store) GetStandaloneClients() ([]*standalone.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	rows, err := s.queries.ListStandaloneClients(ctx)
	if err != nil {
		return nil, err
	}

	clients := make([]*standalone.Client, 0, len(rows))
	for _, row := range rows {
		var connectedAt time.Time
		if row.ConnectedAt != nil {
			connectedAt = *row.ConnectedAt
		}

		client := &standalone.Client{
			ID:          row.ID,
			Name:        row.Name,
			ConnectedAt: connectedAt,
			LastSeen:    row.LastSync,
		}
		clients = append(clients, client)
	}

	return clients, nil
}

func (s *Store) SaveStandaloneClient(client *standalone.Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.InsertStandaloneClient(ctx, sqlc.InsertStandaloneClientParams{
		ID:          client.ID,
		Name:        client.Name,
		MachineInfo: nil, // Client type doesn't have MachineInfo
	})
}

func (s *Store) DeleteStandaloneClient(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeleteStandaloneClient(ctx, id)
}

// ============================================================================
// Standalone Connections Operations (Client Mode)
// ============================================================================

func (s *Store) GetStandaloneConnection(name string) (*standalone.StandaloneConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetStandaloneConnection(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection %q not found", name)
		}

		return nil, err
	}

	return sqlcConnectionToModel(row), nil
}

func (s *Store) ListStandaloneConnections() ([]*standalone.StandaloneConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	rows, err := s.queries.ListStandaloneConnections(ctx)
	if err != nil {
		return nil, err
	}

	connections := make([]*standalone.StandaloneConnection, 0, len(rows))
	for _, row := range rows {
		connections = append(connections, sqlcConnectionToModel(row))
	}

	return connections, nil
}

func (s *Store) SaveStandaloneConnection(conn *standalone.StandaloneConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	syncedItemsJSON, _ := json.Marshal(conn.SyncedItems)
	syncedItemsStr := string(syncedItemsJSON)
	syncStatusStr := conn.SyncStatus

	// Check if exists
	_, err := s.queries.GetStandaloneConnection(ctx, conn.Name)
	if err == sql.ErrNoRows {
		// Insert new
		_, err = s.queries.InsertStandaloneConnection(ctx, sqlc.InsertStandaloneConnectionParams{
			Name:                  conn.Name,
			InstanceID:            ptrString(conn.InstanceID),
			Host:                  conn.Host,
			Port:                  ptrInt64(int64(conn.Port)),
			ApiKeyEncrypted:       conn.APIKeyEncrypted,
			RefreshTokenEncrypted: conn.RefreshTokenEncrypted,
			LocalPasswordHash:     conn.LocalPasswordHash,
			LocalSalt:             conn.LocalSalt,
			SyncStatus:            ptrString(syncStatusStr),
			SyncedItems:           &syncedItemsStr,
			LastSync:              conn.LastSync,
		})

		return err
	} else if err != nil {
		return err
	}

	// Update existing
	return s.queries.UpdateStandaloneConnection(ctx, sqlc.UpdateStandaloneConnectionParams{
		InstanceID:            ptrString(conn.InstanceID),
		Host:                  conn.Host,
		Port:                  ptrInt64(int64(conn.Port)),
		ApiKeyEncrypted:       conn.APIKeyEncrypted,
		RefreshTokenEncrypted: conn.RefreshTokenEncrypted,
		LocalPasswordHash:     conn.LocalPasswordHash,
		LocalSalt:             conn.LocalSalt,
		SyncStatus:            ptrString(syncStatusStr),
		SyncedItems:           &syncedItemsStr,
		LastSync:              conn.LastSync,
		Name:                  conn.Name,
	})
}

func (s *Store) DeleteStandaloneConnection(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeleteStandaloneConnection(ctx, name)
}

// ============================================================================
// Server Encryption Config Operations
// ============================================================================

func (s *Store) GetServerEncryptionConfig() (*standalone.ServerEncryptionConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetServerEncryptionConfig(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	config := &standalone.ServerEncryptionConfig{
		Enabled:      derefInt64ToBool(row.Enabled),
		KeyHash:      row.KeyHash,
		Salt:         row.Salt,
		KeyHint:      derefString(row.KeyHint),
		ConfiguredAt: row.ConfiguredAt,
	}

	return config, nil
}

func (s *Store) SaveServerEncryptionConfig(config *standalone.ServerEncryptionConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	enabled := int64(0)
	if config.Enabled {
		enabled = 1
	}

	return s.queries.UpsertServerEncryptionConfig(ctx, sqlc.UpsertServerEncryptionConfigParams{
		Enabled: ptrInt64(enabled),
		KeyHash: config.KeyHash,
		Salt:    config.Salt,
		KeyHint: ptrString(config.KeyHint),
	})
}

// ============================================================================
// Synced Data Operations
// ============================================================================

func (s *Store) GetSyncedData(connectionName, dataType, name string) (*standalone.SyncedData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetSyncedData(ctx, sqlc.GetSyncedDataParams{
		ConnectionName: connectionName,
		DataType:       dataType,
		Name:           name,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("synced data not found")
		}

		return nil, err
	}

	return sqlcSyncedDataToModel(row), nil
}

func (s *Store) ListSyncedData(connectionName string) ([]*standalone.SyncedData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	rows, err := s.queries.ListSyncedDataByConnection(ctx, connectionName)
	if err != nil {
		return nil, err
	}

	data := make([]*standalone.SyncedData, 0, len(rows))
	for _, row := range rows {
		data = append(data, sqlcSyncedDataToModel(row))
	}

	return data, nil
}

func (s *Store) ListSyncedDataByState(state standalone.SyncState) ([]*standalone.SyncedData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	stateStr := string(state)

	rows, err := s.queries.ListSyncedDataByState(ctx, &stateStr)
	if err != nil {
		return nil, err
	}

	data := make([]*standalone.SyncedData, 0, len(rows))
	for _, row := range rows {
		data = append(data, sqlcSyncedDataToModel(row))
	}

	return data, nil
}

func (s *Store) SaveSyncedData(data *standalone.SyncedData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	stateStr := string(data.State)

	return s.queries.UpsertSyncedData(ctx, sqlc.UpsertSyncedDataParams{
		ID:             data.ID,
		ConnectionName: data.ConnectionName,
		InstanceID:     ptrString(data.InstanceID),
		DataType:       data.DataType,
		Name:           data.Name,
		EncryptedData:  data.EncryptedData,
		Nonce:          data.Nonce,
		State:          &stateStr,
		Checksum:       ptrString(data.Checksum),
	})
}

func (s *Store) DeleteSyncedData(connectionName, dataType, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeleteSyncedData(ctx, sqlc.DeleteSyncedDataParams{
		ConnectionName: connectionName,
		DataType:       dataType,
		Name:           name,
	})
}

// ============================================================================
// Pending Registration Operations (Server Side)
// ============================================================================

func (s *Store) SavePendingRegistration(reg *standalone.ClientRegistration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	machineInfoJSON, _ := json.Marshal(reg.MachineInfo)
	machineInfoStr := string(machineInfoJSON)
	stateStr := string(reg.State)

	return s.queries.InsertPendingRegistration(ctx, sqlc.InsertPendingRegistrationParams{
		ClientID:       reg.ClientID,
		ClientName:     reg.ClientName,
		MachineInfo:    &machineInfoStr,
		State:          &stateStr,
		ChallengeToken: ptrString(reg.ChallengeToken),
		ChallengeAt:    reg.ChallengeAt,
	})
}

func (s *Store) GetPendingRegistration(clientID string) (*standalone.ClientRegistration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetPendingRegistration(ctx, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pending registration not found")
		}

		return nil, err
	}

	var machineInfo standalone.MachineInfo
	if row.MachineInfo != nil && *row.MachineInfo != "" {
		_ = json.Unmarshal([]byte(*row.MachineInfo), &machineInfo)
	}

	reg := &standalone.ClientRegistration{
		ClientID:       row.ClientID,
		ClientName:     row.ClientName,
		MachineInfo:    machineInfo,
		State:          standalone.HandshakeState(derefString(row.State)),
		ChallengeToken: derefString(row.ChallengeToken),
		InitiatedAt:    row.InitiatedAt,
		ChallengeAt:    row.ChallengeAt,
		CompletedAt:    row.CompletedAt,
	}

	return reg, nil
}

func (s *Store) ListPendingRegistrations() ([]*standalone.ClientRegistration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	rows, err := s.queries.ListPendingRegistrations(ctx)
	if err != nil {
		return nil, err
	}

	regs := make([]*standalone.ClientRegistration, 0, len(rows))
	for _, row := range rows {
		var machineInfo standalone.MachineInfo
		if row.MachineInfo != nil && *row.MachineInfo != "" {
			_ = json.Unmarshal([]byte(*row.MachineInfo), &machineInfo)
		}

		reg := &standalone.ClientRegistration{
			ClientID:       row.ClientID,
			ClientName:     row.ClientName,
			MachineInfo:    machineInfo,
			State:          standalone.HandshakeState(derefString(row.State)),
			ChallengeToken: derefString(row.ChallengeToken),
			InitiatedAt:    row.InitiatedAt,
			ChallengeAt:    row.ChallengeAt,
			CompletedAt:    row.CompletedAt,
		}
		regs = append(regs, reg)
	}

	return regs, nil
}

func (s *Store) RemovePendingRegistration(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeletePendingRegistration(ctx, clientID)
}

// ============================================================================
// Registered Clients Operations (Server Side)
// ============================================================================

func (s *Store) SaveRegisteredClient(client *standalone.RegisteredClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	machineInfoJSON, _ := json.Marshal(client.MachineInfo)
	machineInfoStr := string(machineInfoJSON)
	statusStr := client.Status

	return s.queries.InsertRegisteredClient(ctx, sqlc.InsertRegisteredClientParams{
		ClientID:          client.ClientID,
		ClientName:        client.ClientName,
		MachineInfo:       &machineInfoStr,
		EncryptionKeyHash: client.EncryptionKeyHash,
		EncryptionSalt:    client.EncryptionSalt,
		KeyHint:           ptrString(client.KeyHint),
		Status:            &statusStr,
		SyncCount:         ptrInt64(int64(client.SyncCount)),
		LastIp:            ptrString(client.LastIP),
	})
}

func (s *Store) GetRegisteredClient(clientID string) (*standalone.RegisteredClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	row, err := s.queries.GetRegisteredClient(ctx, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("registered client not found")
		}

		return nil, err
	}

	return sqlcRegisteredClientToModel(row), nil
}

func (s *Store) ListRegisteredClients() ([]*standalone.RegisteredClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()

	rows, err := s.queries.ListRegisteredClients(ctx)
	if err != nil {
		return nil, err
	}

	clients := make([]*standalone.RegisteredClient, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, sqlcRegisteredClientToModel(row))
	}

	return clients, nil
}

func (s *Store) DeleteRegisteredClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	return s.queries.DeleteRegisteredClient(ctx, clientID)
}

// ============================================================================
// Helpers
// ============================================================================

func newContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel // Will be collected by GC

	return ctx
}
