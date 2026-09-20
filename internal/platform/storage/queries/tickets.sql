-- name: CreateTicket :exec
INSERT INTO tickets (id, title, body, status, position, number, doc_id, project_id, category_id, type_id, assignee, created_at, updated_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTicket :one
SELECT * FROM tickets WHERE id = ?;

-- name: GetTicketByPrefixAndNumber :one
SELECT tickets.* FROM tickets JOIN projects ON tickets.project_id = projects.id WHERE projects.prefix = ? AND tickets.number = ?;

-- name: ListTickets :many
SELECT * FROM tickets ORDER BY created_at;

-- name: ListTicketsByDoc :many
SELECT * FROM tickets WHERE doc_id = ? ORDER BY created_at;

-- name: ListTicketsByProject :many
SELECT * FROM tickets WHERE project_id = ? ORDER BY created_at;

-- name: GetTicketStatusAndCategory :one
SELECT status, category_id FROM tickets WHERE id = ?;

-- name: UpdateTicketStatus :execrows
UPDATE tickets SET status = ?, position = ?, updated_at = ? WHERE id = ?;

-- name: SetTicketPosition :execrows
UPDATE tickets SET position = ?, updated_at = ? WHERE id = ?;

-- name: MaxTicketPositionInPair :one
SELECT MAX(position) FROM tickets WHERE status = ? AND category_id IS ?;

-- name: MaxTicketNumberInProject :one
SELECT MAX(number) FROM tickets WHERE project_id IS ?;

-- name: DeleteTicket :execrows
DELETE FROM tickets WHERE id = ?;

-- name: LinkTicketPR :exec
INSERT INTO ticket_pr_links (ticket_id, pr_owner, pr_repo, pr_number, pr_title, pr_sha, pr_state, linked_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(ticket_id, pr_owner, pr_repo, pr_number) DO NOTHING;

-- name: ListTicketPRLinks :many
SELECT pr_owner, pr_repo, pr_number, pr_title, pr_sha, pr_state FROM ticket_pr_links WHERE ticket_id = ? ORDER BY linked_at;

-- name: ListTicketPRLinksBatch :many
SELECT ticket_id, pr_owner, pr_repo, pr_number, pr_title, pr_sha, pr_state FROM ticket_pr_links WHERE ticket_id IN (sqlc.slice('ids')) ORDER BY linked_at;

-- name: ListTicketIDsByPR :many
SELECT ticket_id FROM ticket_pr_links WHERE pr_owner = ? AND pr_repo = ? AND pr_number = ?;

-- name: UpdateTicketPRState :exec
UPDATE ticket_pr_links SET pr_state = ? WHERE pr_owner = ? AND pr_repo = ? AND pr_number = ?;

-- name: SetTicketFinishedAt :execrows
UPDATE tickets SET finished_at = ? WHERE id = ? AND finished_at IS NULL;

-- name: LinkTicketBranch :exec
INSERT INTO ticket_branch_links (ticket_id, branch_owner, branch_repo, branch_name, linked_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(ticket_id, branch_owner, branch_repo, branch_name) DO NOTHING;

-- name: ListTicketBranchLinks :many
SELECT branch_owner, branch_repo, branch_name FROM ticket_branch_links WHERE ticket_id = ? ORDER BY linked_at;

-- name: UpdateTicketType :execrows
UPDATE tickets SET type_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTicketAssignee :execrows
UPDATE tickets SET assignee = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTicketTitleBody :execrows
UPDATE tickets SET title = ?, body = ?, updated_at = ? WHERE id = ?;

-- name: MoveTicketCategory :execrows
UPDATE tickets SET category_id = ?, position = ?, updated_at = ? WHERE id = ?;

-- name: AddTicketLabel :exec
INSERT INTO ticket_labels (ticket_id, label) VALUES (?, ?)
ON CONFLICT(ticket_id, label) DO NOTHING;

-- name: RemoveTicketLabel :exec
DELETE FROM ticket_labels WHERE ticket_id = ? AND label = ?;

-- name: ListTicketLabels :many
SELECT label FROM ticket_labels WHERE ticket_id = ? ORDER BY label;

-- name: ListAllTicketLabels :many
SELECT DISTINCT label FROM ticket_labels ORDER BY label;

-- name: ListTicketLabelsForTickets :many
SELECT ticket_id, label FROM ticket_labels WHERE ticket_id IN (sqlc.slice('ids')) ORDER BY label;

-- name: SetLabelColor :exec
INSERT INTO label_colors (project_id, label, color, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(project_id, label) DO UPDATE SET color = excluded.color, updated_at = excluded.updated_at;

-- name: ListLabelColors :many
SELECT label, color FROM label_colors WHERE project_id = ? AND label IN (sqlc.slice('labels'));
