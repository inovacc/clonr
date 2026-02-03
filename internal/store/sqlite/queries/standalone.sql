-- Standalone Config
-- name: GetStandaloneConfig :one
SELECT * FROM standalone_config WHERE id = 1;

-- name: UpsertStandaloneConfig :exec
INSERT INTO standalone_config (id, enabled, is_server, instance_id, port, api_key_hash, refresh_token, salt, capabilities, created_at, expires_at)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
ON CONFLICT(id) DO UPDATE SET
    enabled = excluded.enabled,
    is_server = excluded.is_server,
    instance_id = excluded.instance_id,
    port = excluded.port,
    api_key_hash = excluded.api_key_hash,
    refresh_token = excluded.refresh_token,
    salt = excluded.salt,
    capabilities = excluded.capabilities,
    expires_at = excluded.expires_at;

-- name: DeleteStandaloneConfig :exec
DELETE FROM standalone_config WHERE id = 1;

-- Standalone Clients (server mode)
-- name: GetStandaloneClient :one
SELECT * FROM standalone_clients WHERE id = ? LIMIT 1;

-- name: ListStandaloneClients :many
SELECT * FROM standalone_clients ORDER BY connected_at DESC;

-- name: InsertStandaloneClient :exec
INSERT INTO standalone_clients (id, name, machine_info, connected_at, last_sync)
VALUES (?, ?, ?, CURRENT_TIMESTAMP, NULL)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    machine_info = excluded.machine_info,
    last_sync = excluded.last_sync;

-- name: UpdateStandaloneClientLastSync :exec
UPDATE standalone_clients SET last_sync = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteStandaloneClient :exec
DELETE FROM standalone_clients WHERE id = ?;

-- Standalone Connections (client mode)
-- name: GetStandaloneConnection :one
SELECT * FROM standalone_connections WHERE name = ? LIMIT 1;

-- name: ListStandaloneConnections :many
SELECT * FROM standalone_connections ORDER BY name ASC;

-- name: InsertStandaloneConnection :one
INSERT INTO standalone_connections (
    name, instance_id, host, port, api_key_encrypted, refresh_token_encrypted,
    local_password_hash, local_salt, sync_status, synced_items, last_sync, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateStandaloneConnection :exec
UPDATE standalone_connections SET
    instance_id = ?,
    host = ?,
    port = ?,
    api_key_encrypted = ?,
    refresh_token_encrypted = ?,
    local_password_hash = ?,
    local_salt = ?,
    sync_status = ?,
    synced_items = ?,
    last_sync = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: UpdateStandaloneConnectionStatus :exec
UPDATE standalone_connections SET
    sync_status = ?,
    last_sync = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: DeleteStandaloneConnection :exec
DELETE FROM standalone_connections WHERE name = ?;

-- Server Encryption Config
-- name: GetServerEncryptionConfig :one
SELECT * FROM server_encryption_config WHERE id = 1;

-- name: UpsertServerEncryptionConfig :exec
INSERT INTO server_encryption_config (id, enabled, key_hash, salt, key_hint, configured_at)
VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    enabled = excluded.enabled,
    key_hash = excluded.key_hash,
    salt = excluded.salt,
    key_hint = excluded.key_hint,
    configured_at = excluded.configured_at;

-- name: DeleteServerEncryptionConfig :exec
DELETE FROM server_encryption_config WHERE id = 1;
