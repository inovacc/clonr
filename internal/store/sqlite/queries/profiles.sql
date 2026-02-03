-- name: GetProfile :one
SELECT * FROM profiles WHERE name = ? LIMIT 1;

-- name: GetActiveProfile :one
SELECT * FROM profiles WHERE is_default = 1 LIMIT 1;

-- name: ListProfiles :many
SELECT * FROM profiles ORDER BY name ASC;

-- name: ProfileExists :one
SELECT EXISTS(SELECT 1 FROM profiles WHERE name = ?) AS exists_flag;

-- name: InsertProfile :one
INSERT INTO profiles (
    name, host, username, token_storage, scopes, is_default,
    encrypted_token, workspace, notify_channels, created_at, last_used_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
RETURNING *;

-- name: UpdateProfile :exec
UPDATE profiles SET
    host = ?,
    username = ?,
    token_storage = ?,
    scopes = ?,
    encrypted_token = ?,
    workspace = ?,
    notify_channels = ?
WHERE name = ?;

-- name: UpdateProfileLastUsed :exec
UPDATE profiles SET last_used_at = CURRENT_TIMESTAMP WHERE name = ?;

-- name: SetActiveProfile :exec
UPDATE profiles SET is_default = CASE WHEN name = ? THEN 1 ELSE 0 END;

-- name: ClearActiveProfile :exec
UPDATE profiles SET is_default = 0;

-- name: DeleteProfile :exec
DELETE FROM profiles WHERE name = ?;

-- name: UpdateProfileNotifyChannels :exec
UPDATE profiles SET notify_channels = ? WHERE name = ?;
