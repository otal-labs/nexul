-- ADR 0152; existing workspace roles keep it off.
ALTER TABLE roles ADD COLUMN show_locked_actions INTEGER NOT NULL DEFAULT 0;

-- Edited copies of the four shipped template roles that carry it on; Client and instance-added templates stay off.
UPDATE instance_templates SET body = json_set(body, '$.show_locked_actions', json('true'))
WHERE kind = 'role' AND key IN ('junior_developer', 'senior_developer', 'devops', 'administrator') AND json_valid(body);
