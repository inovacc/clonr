-- Docker profile queries

-- name: InsertDockerProfile :execlastid
INSERT INTO docker_profiles (name, registry, username, encrypted_token, token_storage)
VALUES (?, ?, ?, ?, ?);

-- name: GetDockerProfile :one
SELECT
    id,
    name,
    registry,
    username,
    encrypted_token,
    token_storage,
    created_at,
    last_used_at
FROM docker_profiles
WHERE name = ?;

-- name: ListDockerProfiles :many
SELECT
    id,
    name,
    registry,
    username,
    encrypted_token,
    token_storage,
    created_at,
    last_used_at
FROM docker_profiles
ORDER BY name;

-- name: UpdateDockerProfile :exec
UPDATE docker_profiles
SET
    registry = ?,
    username = ?,
    encrypted_token = ?,
    token_storage = ?,
    last_used_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: UpdateDockerProfileLastUsed :exec
UPDATE docker_profiles
SET last_used_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: DeleteDockerProfile :exec
DELETE FROM docker_profiles WHERE name = ?;

-- name: DockerProfileExists :one
SELECT COUNT(*) FROM docker_profiles WHERE name = ?;
