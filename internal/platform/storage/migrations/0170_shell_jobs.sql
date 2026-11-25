-- The owner's switch for commands from Nexul on a computer; the computer's own opt-in lives in its runner's unit.
ALTER TABLE pairing_computers ADD COLUMN shell_enabled INTEGER NOT NULL DEFAULT 1;

-- One command run on a computer through its runner. The command, who ran it, when and how it ended are kept forever,
-- past the computer and the people; the output is emptied 30 days after the job ends.
CREATE TABLE IF NOT EXISTS runner_shell_jobs (
    id               TEXT PRIMARY KEY,
    computer_id      TEXT NOT NULL,
    owner_id         TEXT NOT NULL,
    requester_id     TEXT NOT NULL,
    adapter          TEXT NOT NULL,
    shell            TEXT NOT NULL DEFAULT '',
    command          TEXT NOT NULL,
    cwd              TEXT NOT NULL DEFAULT '',
    timeout_seconds  INTEGER NOT NULL,
    status           TEXT NOT NULL,
    exit_code        INTEGER,
    error            TEXT NOT NULL DEFAULT '',
    output           TEXT NOT NULL DEFAULT '',
    output_truncated INTEGER NOT NULL DEFAULT 0,
    output_purged    INTEGER NOT NULL DEFAULT 0,
    created_at       INTEGER NOT NULL,
    started_at       INTEGER,
    ended_at         INTEGER
);

-- A computer's command history, newest first, paged by offset (ListShellJobs).
CREATE INDEX IF NOT EXISTS idx_runner_shell_jobs_computer ON runner_shell_jobs(computer_id, created_at DESC, id);

-- The daily purge's scan: ended jobs still holding output (PurgeShellJobOutput).
CREATE INDEX IF NOT EXISTS idx_runner_shell_jobs_output ON runner_shell_jobs(ended_at) WHERE output != '';
