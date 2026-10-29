-- name: CreateTicket :exec
INSERT INTO tickets (id, title, body, status, position, number, doc_id, project_id, category_id, type_id, developer, tester, reporter_kind, reporter_login, reporter_automation_id, reporter_automation_name, created_at, updated_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTicket :one
-- A developer or tester is a user id, read back with its login, or a login nobody has signed in with, read back as is.
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
WHERE tickets.id = ?;

-- name: ListTicketsByPrefixAndNumber :many
-- A key repeats across workspaces, and inside one when a moved ticket keeps its number, so every match comes back.
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login,
       workspaces.id AS workspace_id, workspaces.slug AS workspace_slug
FROM tickets
JOIN projects ON tickets.project_id = projects.id
LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
JOIN workspaces ON workspaces.id = projects.workspace_id
WHERE projects.prefix = ? AND tickets.number = ?
ORDER BY workspaces.slug, tickets.created_at;

-- name: ListTickets :many
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
ORDER BY tickets.created_at;

-- name: ListTicketsByIDs :many
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
WHERE tickets.id IN (sqlc.slice('ids'));

-- name: ListTicketsByDoc :many
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
WHERE tickets.doc_id = ? ORDER BY tickets.created_at;

-- name: ListTicketsByProject :many
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
WHERE tickets.project_id = ? ORDER BY tickets.created_at;

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

-- name: DeleteTicketPRLink :exec
DELETE FROM ticket_pr_links WHERE ticket_id = ? AND pr_owner = ? AND pr_repo = ? AND pr_number = ?;

-- name: ListTicketPRLinks :many
SELECT pr_owner, pr_repo, pr_number, pr_title, pr_sha, pr_state FROM ticket_pr_links WHERE ticket_id = ? ORDER BY linked_at;

-- name: ListTicketPRLinksBatch :many
SELECT ticket_id, pr_owner, pr_repo, pr_number, pr_title, pr_sha, pr_state FROM ticket_pr_links WHERE ticket_id IN (sqlc.slice('ids')) ORDER BY linked_at;

-- name: SearchTicketPRLinksInWorkspace :many
SELECT l.ticket_id, t.project_id, l.pr_owner, l.pr_repo, l.pr_number, l.pr_title, l.linked_at
FROM ticket_pr_links l JOIN tickets t ON t.id = l.ticket_id JOIN projects p ON p.id = t.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id) AND instr(lower(l.pr_title), sqlc.arg(needle)) > 0
ORDER BY l.linked_at DESC, l.pr_number DESC
LIMIT sqlc.arg(row_limit);

-- name: ListTicketIDsByPR :many
SELECT ticket_id FROM ticket_pr_links WHERE pr_owner = ? AND pr_repo = ? AND pr_number = ?;

-- name: ListTicketOpenPRNumbers :many
SELECT DISTINCT pr_number FROM ticket_pr_links WHERE pr_owner = ? AND pr_repo = ? AND pr_state = 'open' ORDER BY pr_number;

-- name: UpdateTicketPRState :exec
UPDATE ticket_pr_links SET pr_state = ? WHERE pr_owner = ? AND pr_repo = ? AND pr_number = ?;

-- name: SetTicketFinishedAt :execrows
UPDATE tickets SET finished_at = ? WHERE id = ? AND finished_at IS NULL;

-- name: LinkTicketBranch :exec
INSERT INTO ticket_branch_links (ticket_id, branch_owner, branch_repo, branch_name, linked_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(ticket_id, branch_owner, branch_repo, branch_name) DO NOTHING;

-- name: DeleteTicketBranchLink :exec
DELETE FROM ticket_branch_links WHERE ticket_id = ? AND branch_owner = ? AND branch_repo = ? AND branch_name = ?;

-- name: ListTicketBranchLinks :many
SELECT branch_owner, branch_repo, branch_name FROM ticket_branch_links WHERE ticket_id = ? ORDER BY linked_at;

-- name: UpdateTicketType :execrows
UPDATE tickets SET type_id = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTicketDeveloper :execrows
UPDATE tickets SET developer = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTicketTester :execrows
UPDATE tickets SET tester = ?, updated_at = ? WHERE id = ?;

-- name: ClaimTicketDeveloperLogin :exec
UPDATE tickets SET developer = sqlc.arg(user_id) WHERE lower(developer) = lower(sqlc.arg(login));

-- name: ClaimTicketTesterLogin :exec
UPDATE tickets SET tester = sqlc.arg(user_id) WHERE lower(tester) = lower(sqlc.arg(login));

-- name: UpdateTicketTitleBody :execrows
UPDATE tickets SET title = ?, body = ?, updated_at = ? WHERE id = ?;

-- name: UpdateTicketDoc :execrows
UPDATE tickets SET doc_id = ?, updated_at = ? WHERE id = ?;

-- name: MoveTicketCategory :execrows
UPDATE tickets SET category_id = ?, position = ?, updated_at = ? WHERE id = ?;

-- name: AddTicketLabel :execrows
INSERT INTO ticket_labels (ticket_id, label) VALUES (?, ?)
ON CONFLICT(ticket_id, label) DO NOTHING;

-- name: RemoveTicketLabel :execrows
DELETE FROM ticket_labels WHERE ticket_id = ? AND label = ?;

-- name: ListTicketLabels :many
SELECT label FROM ticket_labels WHERE ticket_id = ? ORDER BY label;

-- name: ListTicketProjects :many
SELECT id, project_id FROM tickets WHERE id IN (sqlc.slice('ids'));

-- name: ListTicketLabelsForTickets :many
SELECT ticket_id, label FROM ticket_labels WHERE ticket_id IN (sqlc.slice('ids')) ORDER BY label;

-- name: ListAllTicketLabels :many
SELECT ticket_id, label FROM ticket_labels ORDER BY label;

-- name: ListTicketLabelsByProject :many
SELECT ticket_id, label FROM ticket_labels WHERE ticket_id IN (SELECT id FROM tickets WHERE project_id = ?) ORDER BY label;

-- name: SetLabelColor :exec
INSERT INTO label_colors (project_id, label, color, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(project_id, label) DO UPDATE SET color = excluded.color, updated_at = excluded.updated_at;

-- name: ListLabelColors :many
SELECT label, color FROM label_colors WHERE project_id = ? AND label IN (sqlc.slice('labels'));

-- name: UpdateTicketDueOn :execrows
UPDATE tickets SET due_on = ?, updated_at = ? WHERE id = ?;

-- name: ListTicketsDueBetween :many
-- Due tickets in one workspace's projects, limited to the readable set unless the caller reads them all; the explicit non-empty test is what lets idx_tickets_due_on serve the range.
SELECT sqlc.embed(tickets),
       CAST(COALESCE(d.id, '') AS TEXT) AS developer_id, CAST(COALESCE(d.login, tickets.developer) AS TEXT) AS developer_login,
       CAST(COALESCE(te.id, '') AS TEXT) AS tester_id, CAST(COALESCE(te.login, tickets.tester) AS TEXT) AS tester_login
FROM tickets LEFT JOIN users d ON d.id = tickets.developer LEFT JOIN users te ON te.id = tickets.tester
WHERE tickets.due_on != '' AND tickets.due_on >= sqlc.arg(from_on) AND tickets.due_on <= sqlc.arg(to_on)
  AND tickets.project_id IN (SELECT id FROM projects WHERE workspace_id = sqlc.arg(workspace_id))
  AND (sqlc.arg(all_projects) OR tickets.project_id IN (SELECT value FROM json_each(sqlc.arg(readable))))
ORDER BY tickets.due_on, tickets.id;
