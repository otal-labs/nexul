-- A personal runner's code is bound to its person and computer (ADR 0146); empty means a deploy runner's code.
ALTER TABLE runner_enrollment_codes ADD COLUMN owner_user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE runner_enrollment_codes ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';

-- One runner per computer: of two codes minted for one computer, the second to enroll is refused.
DROP INDEX IF EXISTS idx_runners_computer;
-- Serves GetRunnerByComputer: DialComputer and the pairing seam finding the runner that reaches a computer.
CREATE UNIQUE INDEX IF NOT EXISTS idx_runners_computer ON runners(computer_id) WHERE computer_id != '';
