-- A schedule auto play fires on a cron schedule and sweeps the tickets its conditions hold for (ADR 0179); schedule is
-- its cron, timezone, stop time and cap as JSON, NULL for every other moment.
ALTER TABLE auto_plays ADD COLUMN schedule TEXT;
-- One at a time: a run waits while another holding the same lock_key goes, the auto play itself or a named resource.
ALTER TABLE auto_plays ADD COLUMN one_at_a_time INTEGER NOT NULL DEFAULT 0;
ALTER TABLE auto_plays ADD COLUMN resource TEXT NOT NULL DEFAULT '';

-- A sweep's queued runs name it, go in its order within a priority, and start nothing after start_by.
ALTER TABLE play_queue ADD COLUMN sweep_id TEXT NOT NULL DEFAULT '';
ALTER TABLE play_queue ADD COLUMN sweep_position INTEGER NOT NULL DEFAULT 0;
ALTER TABLE play_queue ADD COLUMN start_by INTEGER;
ALTER TABLE play_queue ADD COLUMN lock_key TEXT NOT NULL DEFAULT '';

-- The person queue reads in sweep order within one second's matches.
DROP INDEX IF EXISTS idx_play_queue_person;
-- Serves ListDuePlayQueue, ListPlayQueuePeople and NextPlayQueueNotBefore: waiting rows by person, highest priority, oldest, sweep order.
CREATE INDEX IF NOT EXISTS idx_play_queue_person ON play_queue(person_id, priority DESC, queued_at, sweep_position, id) WHERE status = 'queued';
-- Serves PlayQueueLockHeld: the rows that may hold a lock.
CREATE INDEX IF NOT EXISTS idx_play_queue_lock ON play_queue(lock_key, status) WHERE lock_key != '' AND status IN ('dispatching', 'started');
-- Serves ListPlayQueueBySweep and PlayQueueSweepBusy: one sweep's rows in its order.
CREATE INDEX IF NOT EXISTS idx_play_queue_sweep ON play_queue(sweep_id, sweep_position) WHERE sweep_id != '';
-- Serves NextPlayQueueStartBy: the earliest window still to close.
CREATE INDEX IF NOT EXISTS idx_play_queue_start_by ON play_queue(start_by) WHERE status = 'queued' AND start_by IS NOT NULL;

-- One row per firing of a schedule auto play; its id is derived from the auto play and the minute it fired, so a firing
-- happens once. finished_at and counts are set once nothing of it waits or works, which sends the digest.
CREATE TABLE IF NOT EXISTS auto_play_sweeps (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    auto_play_id TEXT NOT NULL,
    play_id      TEXT NOT NULL,
    play_label   TEXT NOT NULL,
    fired_at     INTEGER NOT NULL,
    start_by     INTEGER,
    matched      INTEGER NOT NULL,
    over_cap     INTEGER NOT NULL,
    counts       TEXT NOT NULL DEFAULT '{}',
    finished_at  INTEGER
);
-- Serves ListLastAutoPlaySweeps: an auto play's latest sweep.
CREATE INDEX IF NOT EXISTS idx_auto_play_sweeps_auto_play ON auto_play_sweeps(auto_play_id, fired_at DESC, id);
-- Serves ListOpenAutoPlaySweeps: the sweeps still going.
CREATE INDEX IF NOT EXISTS idx_auto_play_sweeps_open ON auto_play_sweeps(fired_at, id) WHERE finished_at IS NULL;
-- Serves ListEnabledScheduleAutoPlays: every switched-on schedule, for the scheduler.
CREATE INDEX IF NOT EXISTS idx_auto_plays_schedule ON auto_plays(created_at, id) WHERE enabled = 1 AND moment = 'schedule';
