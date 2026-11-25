-- A machine's switch for commands from Nexul on its runners, off until someone with Run commands on it turns it on; each
-- runner's own opt-in is its install's --allow-shell, which Nexul can never turn on.
ALTER TABLE machines ADD COLUMN shell_enabled INTEGER NOT NULL DEFAULT 0;

-- A command on a machine names the runner it ran on and the machine, where a computer's names the computer and owner.
ALTER TABLE runner_shell_jobs ADD COLUMN runner_id TEXT NOT NULL DEFAULT '';
ALTER TABLE runner_shell_jobs ADD COLUMN machine_id TEXT NOT NULL DEFAULT '';

-- A machine's command history, newest first, paged by offset (ListMachineShellJobs).
CREATE INDEX IF NOT EXISTS idx_runner_shell_jobs_machine ON runner_shell_jobs(machine_id, created_at DESC, id) WHERE machine_id != '';
