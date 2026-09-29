-- Rows from before 0035 show in no workspace's inbox yet still collapse new ones; scope them by subject or drop them.
UPDATE notifications SET workspace_id = COALESCE(CASE subject_type
    WHEN 'ticket' THEN (SELECT p.workspace_id FROM tickets t JOIN projects p ON p.id = t.project_id WHERE t.id = notifications.subject_id)
    WHEN 'doc' THEN (SELECT p.workspace_id FROM docs d JOIN projects p ON p.id = d.project_id WHERE d.id = notifications.subject_id)
    WHEN 'memory' THEN (SELECT m.workspace_id FROM memories m WHERE m.id = notifications.subject_id)
END, '')
WHERE workspace_id = '';
DELETE FROM notifications WHERE workspace_id = '';
