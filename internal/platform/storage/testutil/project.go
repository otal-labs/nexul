// Package testutil holds storage fixtures for tests in any package that migrates a real database.
package testutil

import (
	"database/sql"
	"fmt"
)

// GeneralProjectID is the fixture project SeedGeneralProject inserts.
const GeneralProjectID = "project-general"

// GeneralMainFolderID is the default doc folder SeedGeneralProject gives its project.
const GeneralMainFolderID = "folder-general-main"

// SeedGeneralProject inserts a project in the seeded default workspace with statuses open, in_progress, done,
// and closed, ticket types ticket-type-task, -bug, and -feature, its default doc folder GeneralMainFolderID, and one
// always-included memory, for tests that need rows to hang off a project and do not care how it was created.
func SeedGeneralProject(db *sql.DB) error {
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at, icon)
VALUES ('project-general', 'General', '', 0, 'workspace-default', 0, 0, '');
INSERT INTO statuses (id, project_id, name, position, kind, icon, created_at, updated_at) VALUES
    ('open', 'project-general', 'Open', 0, 'backlog', '', 0, 0),
    ('in_progress', 'project-general', 'In progress', 1, 'progress', '', 0, 0),
    ('done', 'project-general', 'Done', 2, 'done', '', 0, 0),
    ('closed', 'project-general', 'Closed', 3, 'done', '', 0, 0);
INSERT INTO ticket_types (id, project_id, name, position, color, created_at, updated_at, body_template) VALUES
    ('ticket-type-task', 'project-general', 'task', 0, '', 0, 0, '## What needs doing' || char(10) || char(10) || char(10) || '## Acceptance criteria' || char(10) || char(10)),
    ('ticket-type-bug', 'project-general', 'bug', 1, '', 0, 0, '## Steps to reproduce' || char(10) || char(10) || char(10) || '## Expected result' || char(10) || char(10) || char(10) || '## Actual result' || char(10) || char(10) || char(10) || '## Provide screenshot' || char(10) || char(10)),
    ('ticket-type-feature', 'project-general', 'feature', 2, '', 0, 0, '## Why' || char(10) || char(10) || char(10) || '## Acceptance criteria' || char(10) || char(10) || char(10) || '## Out of scope' || char(10) || char(10));
INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at)
VALUES ('folder-general-main', 'project-general', 'Main', 1, 0, 0);
INSERT INTO memories (id, workspace_id, project_id, title, when_to_use, body, always_included, version, created_by, created_at, updated_by, updated_at)
VALUES ('memory-project-general-working', 'workspace-default', 'project-general', 'Working in this project',
    'Always included in this project''s turns.', 'Starter context for this project.', 1, 1, '', 0, '', 0);
`)
	if err != nil {
		return fmt.Errorf("seed general project: %w", err)
	}
	return nil
}
