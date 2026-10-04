-- A body written outside the room before the live-session reset (ADR 0109) left the room's older state, which editors
-- still replay over the body; drop rooms whose newest update predates the doc's last unnamed version row.
DELETE FROM collab_updates WHERE doc_id IN (
    SELECT u.doc_id FROM collab_updates u GROUP BY u.doc_id
    HAVING MAX(u.created_at) <= (SELECT MAX(v.created_at) FROM doc_versions v WHERE v.doc_id = u.doc_id AND v.name = '')
);
