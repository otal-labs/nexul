-- A note's markdown (ADR 0108) is searchable as its ticket; rows are keyed by attachment id because VACUUM may renumber a TEXT-keyed table's rowids.
CREATE VIRTUAL TABLE IF NOT EXISTS ticket_notes_fts USING fts5(ticket_id UNINDEXED, attachment_id UNINDEXED, body);

-- CreateNote inserts the file before the message, so the file is there when the message links it.
CREATE TRIGGER IF NOT EXISTS ticket_notes_fts_message_ai AFTER INSERT ON messages WHEN new.attachment_id IS NOT NULL AND new.deleted_at IS NULL BEGIN
    INSERT INTO ticket_notes_fts(ticket_id, attachment_id, body)
    SELECT c.ticket_id, a.id, CAST(a.data AS TEXT) FROM attachments a JOIN conversations c ON c.id = new.conversation_id
    WHERE a.id = new.attachment_id AND c.ticket_id IS NOT NULL;
END;

-- Replacing a note's file overwrites its bytes in place.
CREATE TRIGGER IF NOT EXISTS ticket_notes_fts_file_au AFTER UPDATE OF data ON attachments WHEN new.conversation_id IS NOT NULL BEGIN
    UPDATE ticket_notes_fts SET body = CAST(new.data AS TEXT) WHERE attachment_id = new.id;
END;

-- Deleting a note deletes its file in the same transaction as the message (ADR 0108), and a deleted ticket cascades to its thread's files.
CREATE TRIGGER IF NOT EXISTS ticket_notes_fts_file_ad AFTER DELETE ON attachments WHEN old.conversation_id IS NOT NULL BEGIN
    DELETE FROM ticket_notes_fts WHERE attachment_id = old.id;
END;

INSERT INTO ticket_notes_fts(ticket_id, attachment_id, body)
SELECT c.ticket_id, a.id, CAST(a.data AS TEXT)
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
JOIN attachments a ON a.id = m.attachment_id
WHERE m.deleted_at IS NULL AND c.ticket_id IS NOT NULL
  AND a.id NOT IN (SELECT attachment_id FROM ticket_notes_fts);
