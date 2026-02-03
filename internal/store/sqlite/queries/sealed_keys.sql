-- Sealed key queries

-- name: GetSealedKey :one
SELECT
    sealed_data,
    version,
    key_type,
    metadata,
    created_at,
    rotated_at,
    last_accessed
FROM sealed_keys
WHERE id = 1;

-- name: InsertSealedKey :exec
INSERT INTO sealed_keys (id, sealed_data, version, key_type, metadata, created_at, rotated_at, last_accessed)
VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
    sealed_data = excluded.sealed_data,
    version = excluded.version,
    key_type = excluded.key_type,
    metadata = excluded.metadata,
    rotated_at = CASE WHEN sealed_keys.sealed_data != excluded.sealed_data THEN CURRENT_TIMESTAMP ELSE sealed_keys.rotated_at END,
    last_accessed = CURRENT_TIMESTAMP;

-- name: UpdateSealedKeyAccess :exec
UPDATE sealed_keys
SET last_accessed = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: DeleteSealedKey :exec
DELETE FROM sealed_keys WHERE id = 1;

-- name: SealedKeyExists :one
SELECT COUNT(*) FROM sealed_keys WHERE id = 1;
