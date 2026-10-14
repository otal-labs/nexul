-- An open code is listed and revoked by this id, so its hash never leaves the server. Codes still open when this runs
-- get a random UUID-shaped one here, so none is left that can't be listed or revoked.
ALTER TABLE runner_enrollment_codes ADD COLUMN id TEXT NOT NULL DEFAULT '';
UPDATE runner_enrollment_codes SET id = lower(
    hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)), 2) || '-' ||
    substr('89ab', abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6)))
WHERE id = '';

-- Serves RevokeRunnerEnrollmentCode and GetRunnerEnrollmentCodeByID.
CREATE UNIQUE INDEX IF NOT EXISTS idx_runner_enrollment_codes_id ON runner_enrollment_codes(id);
