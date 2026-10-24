-- name: MeetingUpsert :exec
INSERT INTO meetings (
    id, workspace_id, organiser_id, title, notes, starts_at, ends_at, timezone, voice_channel_id, link, repeats,
    repeat_until, remind_minutes, series_id, replaces_at, cancelled_at, next_remind_at, next_occurrence, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    title = excluded.title, notes = excluded.notes, starts_at = excluded.starts_at, ends_at = excluded.ends_at,
    timezone = excluded.timezone, voice_channel_id = excluded.voice_channel_id, link = excluded.link,
    repeats = excluded.repeats, repeat_until = excluded.repeat_until, remind_minutes = excluded.remind_minutes,
    series_id = excluded.series_id, replaces_at = excluded.replaces_at, cancelled_at = excluded.cancelled_at,
    next_remind_at = excluded.next_remind_at, next_occurrence = excluded.next_occurrence, updated_at = excluded.updated_at;

-- name: MeetingGet :one
SELECT * FROM meetings WHERE id = ?;

-- name: MeetingsByIDs :many
SELECT * FROM meetings WHERE id IN (SELECT value FROM json_each(sqlc.arg(ids)));

-- name: MeetingListVisible :many
SELECT * FROM meetings
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (organiser_id = sqlc.arg(user_id) OR id IN (SELECT meeting_id FROM meeting_invitees WHERE user_id = sqlc.arg(user_id)))
  AND starts_at < sqlc.arg(to_at)
  AND (ends_at > sqlc.arg(from_at)
    OR (repeats != 'none' AND (repeat_until IS NULL OR repeat_until + (ends_at - starts_at) > sqlc.arg(from_at))))
ORDER BY starts_at, id;

-- name: MeetingReplacements :many
SELECT * FROM meetings
WHERE series_id != '' AND series_id IN (SELECT value FROM json_each(sqlc.arg(series_ids)))
ORDER BY replaces_at, id;

-- name: MeetingInviteesOf :many
SELECT * FROM meeting_invitees WHERE meeting_id IN (SELECT value FROM json_each(sqlc.arg(ids))) ORDER BY meeting_id, user_id;

-- name: MeetingInviteeInsert :exec
INSERT INTO meeting_invitees (meeting_id, user_id, reply, replied_at) VALUES (?, ?, ?, ?)
ON CONFLICT(meeting_id, user_id) DO NOTHING;

-- name: MeetingInviteesDeleteExcept :exec
DELETE FROM meeting_invitees
WHERE meeting_id = sqlc.arg(meeting_id) AND user_id NOT IN (SELECT value FROM json_each(sqlc.arg(user_ids)));

-- name: MeetingReplySave :execrows
UPDATE meeting_invitees SET reply = ?, replied_at = ? WHERE meeting_id = ? AND user_id = ?;

-- name: MeetingNextReminder :one
SELECT MIN(next_remind_at) FROM meetings WHERE next_remind_at IS NOT NULL;

-- name: MeetingRemindersDue :many
SELECT * FROM meetings WHERE next_remind_at IS NOT NULL AND next_remind_at <= ? ORDER BY next_remind_at, id;

-- name: MeetingMarkReminded :execrows
UPDATE meetings SET next_remind_at = ?, next_occurrence = ? WHERE id = ? AND next_remind_at = ?;

-- name: MeetingDueTickets :many
SELECT t.id, t.project_id, p.workspace_id, p.prefix, t.number, t.title, t.due_on, u.id AS developer_id, u.timezone
FROM tickets t
JOIN projects p ON p.id = t.project_id
JOIN users u ON u.login = t.developer
LEFT JOIN statuses s ON s.id = t.status
LEFT JOIN ticket_due_reminders r ON r.ticket_id = t.id
WHERE t.due_on != '' AND t.due_on >= sqlc.arg(from_date) AND t.due_on <= sqlc.arg(to_date)
  AND t.developer != '' AND COALESCE(s.kind, '') != 'done'
  AND (r.due_on IS NULL OR r.due_on != t.due_on)
ORDER BY t.due_on, t.id;

-- name: MeetingTicketReminded :exec
INSERT INTO ticket_due_reminders (ticket_id, due_on, sent_at) VALUES (?, ?, ?)
ON CONFLICT(ticket_id) DO UPDATE SET due_on = excluded.due_on, sent_at = excluded.sent_at;
