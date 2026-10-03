-- Where a person's new harness threads start: '' or 'folder' is the T3 project's own folder, 'worktree' a fresh git
-- worktree of it. On a project link '' falls through to the person's pairing defaults.
ALTER TABLE pairing_user_defaults ADD COLUMN start_in TEXT NOT NULL DEFAULT '';
ALTER TABLE pairing_project_links ADD COLUMN start_in TEXT NOT NULL DEFAULT '';
