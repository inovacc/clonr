-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE name = ? LIMIT 1;

-- name: GetActiveWorkspace :one
SELECT * FROM workspaces WHERE is_active = 1 LIMIT 1;

-- name: ListWorkspaces :many
SELECT * FROM workspaces ORDER BY name ASC;

-- name: WorkspaceExists :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE name = ?) AS exists_flag;

-- name: InsertWorkspace :one
INSERT INTO workspaces (name, description, path, is_active, created_at, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateWorkspace :exec
UPDATE workspaces SET
    description = ?,
    path = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: SetActiveWorkspace :exec
UPDATE workspaces SET is_active = CASE WHEN name = ? THEN 1 ELSE 0 END;

-- name: ClearActiveWorkspace :exec
UPDATE workspaces SET is_active = 0;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE name = ?;
