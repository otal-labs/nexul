-- name: InsertTicketLink :exec
INSERT INTO ticket_links (ticket_id, kind, target_id, created_at) VALUES (?, ?, ?, ?);

-- name: DeleteTicketFoundIn :execrows
DELETE FROM ticket_links WHERE ticket_id = ? AND kind = 'found_in';

-- name: DeleteTicketBlocker :execrows
DELETE FROM ticket_links WHERE ticket_id = ? AND kind = 'blocked_by' AND target_id = ?;

-- name: ListTicketBlockerIDs :many
SELECT target_id FROM ticket_links WHERE ticket_id = ? AND kind = 'blocked_by' AND target_id IS NOT NULL;

-- name: ListTicketLinksFrom :many
SELECT l.kind, l.target_id, t.project_id, t.number, t.title, t.status, p.prefix, s.kind AS stage
FROM ticket_links l
LEFT JOIN tickets t ON t.id = l.target_id
LEFT JOIN projects p ON p.id = t.project_id
LEFT JOIN statuses s ON s.id = t.status
WHERE l.ticket_id = ?
ORDER BY l.created_at, l.target_id;

-- name: ListTicketLinksTo :many
SELECT l.kind, l.ticket_id, t.project_id, t.number, t.title, t.status, p.prefix, s.kind AS stage
FROM ticket_links l
JOIN tickets t ON t.id = l.ticket_id
LEFT JOIN projects p ON p.id = t.project_id
LEFT JOIN statuses s ON s.id = t.status
WHERE l.target_id = ?
ORDER BY l.created_at, l.ticket_id;

-- name: ListUnclearedTicketBlockers :many
SELECT l.ticket_id AS blocked_id, t.id, t.project_id, t.number, t.title, t.status, p.prefix
FROM ticket_links l
JOIN tickets t ON t.id = l.target_id
LEFT JOIN projects p ON p.id = t.project_id
LEFT JOIN statuses s ON s.id = t.status
WHERE l.kind = 'blocked_by' AND COALESCE(s.kind, '') != 'done'
ORDER BY l.ticket_id, l.created_at, t.id;
