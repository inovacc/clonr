-- Synced Data
-- name: GetSyncedData :one
SELECT * FROM synced_data
WHERE connection_name = ? AND data_type = ? AND name = ?
LIMIT 1;

-- name: ListSyncedDataByConnection :many
SELECT * FROM synced_data
WHERE connection_name = ?
ORDER BY synced_at DESC;

-- name: ListSyncedDataByState :many
SELECT * FROM synced_data
WHERE state = ?
ORDER BY synced_at DESC;

-- name: ListSyncedDataByConnectionAndType :many
SELECT * FROM synced_data
WHERE connection_name = ? AND data_type = ?
ORDER BY synced_at DESC;

-- name: InsertSyncedData :one
INSERT INTO synced_data (
    id, connection_name, instance_id, data_type, name,
    encrypted_data, nonce, state, checksum, synced_at, decrypted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
RETURNING *;

-- name: UpsertSyncedData :exec
INSERT INTO synced_data (
    id, connection_name, instance_id, data_type, name,
    encrypted_data, nonce, state, checksum, synced_at, decrypted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
ON CONFLICT(connection_name, data_type, name) DO UPDATE SET
    instance_id = excluded.instance_id,
    encrypted_data = excluded.encrypted_data,
    nonce = excluded.nonce,
    state = excluded.state,
    checksum = excluded.checksum,
    synced_at = CURRENT_TIMESTAMP;

-- name: UpdateSyncedDataState :exec
UPDATE synced_data SET
    state = @state,
    decrypted_at = CASE WHEN @state = 'decrypted' THEN CURRENT_TIMESTAMP ELSE decrypted_at END
WHERE connection_name = @connection_name AND data_type = @data_type AND name = @name;

-- name: DeleteSyncedData :exec
DELETE FROM synced_data
WHERE connection_name = ? AND data_type = ? AND name = ?;

-- name: DeleteSyncedDataByConnection :exec
DELETE FROM synced_data WHERE connection_name = ?;

-- Pending Registrations (server side)
-- name: GetPendingRegistration :one
SELECT * FROM pending_registrations WHERE client_id = ? LIMIT 1;

-- name: ListPendingRegistrations :many
SELECT * FROM pending_registrations ORDER BY initiated_at DESC;

-- name: InsertPendingRegistration :exec
INSERT INTO pending_registrations (
    client_id, client_name, machine_info, state,
    challenge_token, challenge_at, initiated_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
ON CONFLICT(client_id) DO UPDATE SET
    client_name = excluded.client_name,
    machine_info = excluded.machine_info,
    state = excluded.state,
    challenge_token = excluded.challenge_token,
    challenge_at = excluded.challenge_at;

-- name: UpdatePendingRegistrationState :exec
UPDATE pending_registrations SET
    state = @state,
    completed_at = CASE WHEN @state IN ('completed', 'rejected') THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE client_id = @client_id;

-- name: DeletePendingRegistration :exec
DELETE FROM pending_registrations WHERE client_id = ?;

-- Registered Clients (server side)
-- name: GetRegisteredClient :one
SELECT * FROM registered_clients WHERE client_id = ? LIMIT 1;

-- name: ListRegisteredClients :many
SELECT * FROM registered_clients ORDER BY registered_at DESC;

-- name: ListRegisteredClientsByStatus :many
SELECT * FROM registered_clients WHERE status = ? ORDER BY registered_at DESC;

-- name: InsertRegisteredClient :exec
INSERT INTO registered_clients (
    client_id, client_name, machine_info, encryption_key_hash, encryption_salt,
    key_hint, status, sync_count, last_ip, registered_at, last_seen_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
ON CONFLICT(client_id) DO UPDATE SET
    client_name = excluded.client_name,
    machine_info = excluded.machine_info,
    encryption_key_hash = excluded.encryption_key_hash,
    encryption_salt = excluded.encryption_salt,
    key_hint = excluded.key_hint,
    status = excluded.status;

-- name: UpdateRegisteredClientLastSeen :exec
UPDATE registered_clients SET
    last_seen_at = CURRENT_TIMESTAMP,
    last_ip = ?,
    sync_count = sync_count + 1
WHERE client_id = ?;

-- name: UpdateRegisteredClientStatus :exec
UPDATE registered_clients SET status = ? WHERE client_id = ?;

-- name: DeleteRegisteredClient :exec
DELETE FROM registered_clients WHERE client_id = ?;
