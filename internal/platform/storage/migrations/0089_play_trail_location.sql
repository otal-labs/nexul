-- Where a run started, which its answer, Continue and reattach keep (ADR 0145); empty on older runs, which read the link.
ALTER TABLE play_trails ADD COLUMN harness_project_id TEXT NOT NULL DEFAULT '';
ALTER TABLE play_trails ADD COLUMN worktree INTEGER NOT NULL DEFAULT 0;
