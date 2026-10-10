-- A personal runner's person and computer (ADR 0146); empty means a deploy runner, so no row needs a backfill.
ALTER TABLE runners ADD COLUMN owner_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE runners ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';
-- Serves GetRunnerByComputer: DialComputer finding the runner that reaches a computer.
CREATE INDEX IF NOT EXISTS idx_runners_computer ON runners(computer_id) WHERE computer_id != '';
