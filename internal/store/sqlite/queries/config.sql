-- name: GetConfig :one
SELECT * FROM config WHERE id = 1;

-- name: UpdateConfig :exec
UPDATE config SET
    default_clone_dir = ?,
    editor = ?,
    terminal = ?,
    monitor_interval = ?,
    server_port = ?,
    custom_editors = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: UpdateConfigEditor :exec
UPDATE config SET editor = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- name: UpdateConfigCloneDir :exec
UPDATE config SET default_clone_dir = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- name: UpdateConfigServerPort :exec
UPDATE config SET server_port = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
