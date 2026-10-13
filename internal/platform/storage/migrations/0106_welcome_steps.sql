-- The Welcome wizard steps a role gives its holders (ADR 0149): a JSON list of step keys, read whole. Existing roles
-- give none; the Owner role's steps are fixed in code, not stored.
ALTER TABLE roles ADD COLUMN welcome_steps TEXT NOT NULL DEFAULT '[]';

-- One person's record of a Welcome wizard step: when they skipped it, when it was given to them again after their
-- first run, and for the phone when it was first done. Everything else about a step is read from other domains' data.
CREATE TABLE IF NOT EXISTS welcome_step_states (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    step       TEXT NOT NULL,
    skipped_at INTEGER,
    given_at   INTEGER,
    done_at    INTEGER,
    PRIMARY KEY (user_id, step)
);

-- Who made a deploy runner's install command, copied from the code to the runner at enrollment; empty for rows from
-- before, and for the instance runner.
ALTER TABLE runner_enrollment_codes ADD COLUMN enrolled_by TEXT NOT NULL DEFAULT '';
ALTER TABLE runners ADD COLUMN enrolled_by TEXT NOT NULL DEFAULT '';
