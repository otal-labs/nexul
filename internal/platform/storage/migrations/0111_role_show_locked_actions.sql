-- Whether a role's holders see an action they can't take dimmed with the reason, instead of not at all (ADR 0152).
-- Existing roles keep it off; a workspace's roles are never touched.
ALTER TABLE roles ADD COLUMN show_locked_actions INTEGER NOT NULL DEFAULT 0;

-- The shipped Template roles that carry it on, matched by their stable key, so an edited copy of one at the instance
-- gets it too. Client stays off, and so does every Template role the instance added itself.
UPDATE instance_templates SET body = json_set(body, '$.show_locked_actions', json('true'))
WHERE kind = 'role' AND key IN ('junior_developer', 'senior_developer', 'devops', 'administrator') AND json_valid(body);
