-- An auto play no longer names computers: it runs as its picker and follows their order (ADR 0197, ADR 0211). A shared
-- computer it named that its picker still holds Run agents on joins the picker's order after their own computers.
INSERT INTO pairing_order_shares (user_id, computer_id, position, created_at)
SELECT DISTINCT a.run_as, j.value,
    (SELECT COALESCE(MAX(own.position), -1) + 1 FROM pairing_computers own WHERE own.user_id = a.run_as),
    CAST(strftime('%s', 'now') AS INTEGER)
FROM auto_plays a
JOIN json_each(a.computer_ids) j
JOIN pairing_computers c ON c.id = j.value AND c.user_id != a.run_as
JOIN permission_overwrites po ON po.resource_type = 'computer' AND po.resource_id = c.id AND po.target = 'person'
    AND po.user_id = a.run_as
WHERE a.run_on = 'computer' AND a.run_as != ''
  AND EXISTS (SELECT 1 FROM json_each(po.allow) WHERE value = 'computers:run_agents')
ON CONFLICT(user_id, computer_id) DO NOTHING;

UPDATE auto_plays SET run_on = 'person' WHERE run_on = 'computer';
UPDATE play_queue SET run_on = 'person' WHERE run_on = 'computer';

ALTER TABLE auto_plays DROP COLUMN computer_ids;
ALTER TABLE auto_plays DROP COLUMN computer_id;
ALTER TABLE play_queue DROP COLUMN computer_ids;
ALTER TABLE play_queue DROP COLUMN computer_id;
