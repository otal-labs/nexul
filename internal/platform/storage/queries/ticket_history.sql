-- name: InsertTicketDescriptionVersion :exec
INSERT INTO ticket_description_versions (ticket_id, version, body, editor_ids, via, words_added, words_removed, restored_from, created_at)
VALUES (
    sqlc.arg(for_ticket),
    (SELECT COALESCE(MAX(m.version), 0) + 1 FROM ticket_description_versions m WHERE m.ticket_id = sqlc.arg(for_ticket)),
    sqlc.arg(body), sqlc.arg(editor_ids), sqlc.arg(via), sqlc.arg(words_added), sqlc.arg(words_removed), sqlc.arg(restored_from), sqlc.arg(created_at)
);

-- name: TicketDescriptionMatchesLatest :one
SELECT EXISTS(
    SELECT 1 FROM ticket_description_versions v
    WHERE v.ticket_id = sqlc.arg(for_ticket) AND v.body = sqlc.arg(want_body)
      AND v.version = (SELECT MAX(l.version) FROM ticket_description_versions l WHERE l.ticket_id = sqlc.arg(for_ticket))
);

-- name: PruneTicketDescriptionVersions :exec
DELETE FROM ticket_description_versions
WHERE ticket_description_versions.ticket_id = sqlc.arg(for_ticket)
  AND ticket_description_versions.version NOT IN (
    SELECT k.version FROM ticket_description_versions k WHERE k.ticket_id = sqlc.arg(for_ticket) ORDER BY k.version DESC LIMIT sqlc.arg(keep)
  );

-- name: ListTicketDescriptionVersionSummaries :many
SELECT version, editor_ids, via, words_added, words_removed, restored_from, created_at FROM ticket_description_versions WHERE ticket_id = ? ORDER BY version DESC;

-- name: ListTicketDescriptionVersionNumbers :many
SELECT version FROM ticket_description_versions WHERE ticket_id = ? ORDER BY version;

-- name: SetTicketDescriptionVersionSize :exec
UPDATE ticket_description_versions SET words_added = ?, words_removed = ? WHERE ticket_id = ? AND version = ?;

-- name: GetTicketDescriptionVersion :one
SELECT ticket_id, version, body, editor_ids, via, created_at FROM ticket_description_versions WHERE ticket_id = ? AND version = ?;

-- name: InsertTicketChange :exec
INSERT INTO ticket_changes (id, ticket_id, field, from_value, to_value, actor, created_at) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: PruneTicketChanges :exec
DELETE FROM ticket_changes
WHERE ticket_changes.ticket_id = sqlc.arg(for_ticket)
  AND ticket_changes.id NOT IN (
    SELECT k.id FROM ticket_changes k WHERE k.ticket_id = sqlc.arg(for_ticket) ORDER BY k.created_at DESC, k.id DESC LIMIT sqlc.arg(keep)
  );

-- name: ListTicketChanges :many
SELECT id, field, from_value, to_value, actor, created_at FROM ticket_changes WHERE ticket_id = ? ORDER BY created_at DESC, id DESC;
