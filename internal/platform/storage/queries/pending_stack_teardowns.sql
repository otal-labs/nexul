-- name: SavePendingStackTeardown :exec
INSERT INTO pending_stack_teardowns (machine, slug, strategy, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(machine, slug) DO UPDATE SET strategy = excluded.strategy, created_at = excluded.created_at;

-- name: ListPendingStackTeardowns :many
SELECT slug, strategy FROM pending_stack_teardowns WHERE machine = ? ORDER BY created_at, slug;

-- name: DeletePendingStackTeardown :exec
DELETE FROM pending_stack_teardowns WHERE machine = ? AND slug = ?;
