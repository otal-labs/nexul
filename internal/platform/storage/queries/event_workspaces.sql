-- name: EventWorkspaceOfProject :one
SELECT workspace_id FROM projects WHERE id = ?;

-- name: EventWorkspaceOfTicket :one
SELECT p.workspace_id FROM tickets t JOIN projects p ON p.id = t.project_id WHERE t.id = ?;

-- name: EventWorkspaceOfDoc :one
SELECT p.workspace_id FROM docs d JOIN projects p ON p.id = d.project_id WHERE d.id = ?;

-- name: EventWorkspaceOfConversation :one
SELECT workspace_id FROM conversations WHERE id = ?;

-- name: EventWorkspaceOfPlay :one
SELECT workspace_id FROM plays WHERE id = ?;

-- name: EventWorkspaceOfRepo :one
SELECT p.workspace_id FROM project_repos r JOIN projects p ON p.id = r.project_id WHERE r.owner = ? AND r.name = ?;

-- name: EventWorkspaceOfStack :one
SELECT COALESCE(p.workspace_id, '') FROM stacks s LEFT JOIN projects p ON p.id = s.project_id WHERE s.id = ?;

-- name: EventWorkspaceOfDeploy :one
SELECT COALESCE(p.workspace_id, '') FROM deploys d LEFT JOIN stacks s ON s.id = d.stack_id LEFT JOIN projects p ON p.id = s.project_id WHERE d.id = ?;

-- name: ListWorkspaceIDs :many
SELECT id FROM workspaces ORDER BY created_at, id;
