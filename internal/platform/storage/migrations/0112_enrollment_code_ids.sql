-- Codes are listed and revoked by id so the hash never leaves the server; open codes get one here.
ALTER TABLE runner_enrollment_codes ADD COLUMN id TEXT NOT NULL DEFAULT '';
UPDATE runner_enrollment_codes SET id = lower(
    hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)), 2) || '-' ||
    substr('89ab', abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6)))
WHERE id = '';

-- Serves RevokeRunnerEnrollmentCode and GetRunnerEnrollmentCodeByID.
CREATE UNIQUE INDEX IF NOT EXISTS idx_runner_enrollment_codes_id ON runner_enrollment_codes(id);
