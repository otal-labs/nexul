-- The wipe (ADR 0148): every rule on a workspace, a project or a play goes, and on a doc every rule but its creator's
-- own, kept as stored; a rule on a doc that is gone goes with it. Roles and who holds them stay.
DELETE FROM permission_overwrites
WHERE resource_type IN ('workspace', 'project', 'play')
   OR (resource_type = 'doc' AND NOT EXISTS (
       SELECT 1 FROM docs d
       WHERE d.id = permission_overwrites.resource_id
         AND permission_overwrites.target = 'person'
         AND permission_overwrites.user_id = d.created_by));

ALTER TABLE workspace_members DROP COLUMN restricted;

-- An invitation gives a set of roles per workspace and nothing per person: an open link keeps its one role and loses
-- its overrides, its Every project and its Project access.
CREATE TABLE invitation_grants_new (
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    workspace_id  TEXT NOT NULL,
    role_id       TEXT NOT NULL,
    -- Serves ListInvitationGrantsByInvitation.
    PRIMARY KEY (invitation_id, workspace_id, role_id)
);
INSERT INTO invitation_grants_new (invitation_id, workspace_id, role_id)
SELECT invitation_id, workspace_id, role_id FROM invitation_grants;
DROP TABLE invitation_grants;
ALTER TABLE invitation_grants_new RENAME TO invitation_grants;

-- No project lists anyone after the wipe, so a workspace with projects waits for an Owner's one-time access setup; one
-- with none, and every workspace made later, starts set up.
ALTER TABLE workspaces ADD COLUMN access_setup_finished INTEGER NOT NULL DEFAULT 1;
UPDATE workspaces SET access_setup_finished = 0
WHERE EXISTS (SELECT 1 FROM projects p WHERE p.workspace_id = workspaces.id);
