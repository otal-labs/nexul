-- An automation's failed and crashed runs, newest first: the runs list's failures view (ListFailedAutomationRunsByAutomation).
CREATE INDEX IF NOT EXISTS idx_automation_runs_failed ON automation_runs(automation_id, started_at DESC, id DESC) WHERE outcome <> 'success';
