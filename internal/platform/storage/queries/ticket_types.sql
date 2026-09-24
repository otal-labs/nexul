-- name: CreateTicketType :exec
INSERT INTO ticket_types (id, project_id, name, position, color, body_template, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTicketType :one
SELECT * FROM ticket_types WHERE id = ?;

-- name: ListTicketTypesByProject :many
SELECT * FROM ticket_types WHERE project_id = ? ORDER BY position, id;

-- name: UpdateTicketTypeMeta :execrows
UPDATE ticket_types SET name = ?, color = ?, body_template = ?, updated_at = ? WHERE id = ?;

-- name: DeleteTicketType :execrows
DELETE FROM ticket_types WHERE id = ?;

-- name: ReorderTicketTypePosition :execrows
UPDATE ticket_types SET position = ?, updated_at = ? WHERE id = ? AND project_id = ?;

-- name: CountTicketTypeTickets :one
SELECT COUNT(*) FROM tickets WHERE type_id = ?;
