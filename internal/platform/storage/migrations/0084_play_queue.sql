-- One row per time an auto play's moment matched a ticket or doc (ADR 0132), kept once decided: it is the person's
-- queue, the target's queued, skipped and didn't-run lines, the daily cap's count, and the once-within limit.
CREATE TABLE IF NOT EXISTS play_queue (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL,
    target_type  TEXT NOT NULL,
    target_id    TEXT NOT NULL,
    play_id      TEXT NOT NULL,
    play_label   TEXT NOT NULL,
    auto_play_id TEXT NOT NULL,
    person_id    TEXT NOT NULL DEFAULT '',
    run_on       TEXT NOT NULL,
    moment       TEXT NOT NULL,
    via          TEXT NOT NULL,
    priority     INTEGER NOT NULL,
    status       TEXT NOT NULL,
    reason       TEXT NOT NULL DEFAULT '',
    trail_id     TEXT NOT NULL DEFAULT '',
    queued_at    INTEGER NOT NULL,
    decided_at   INTEGER,
    not_before   INTEGER NOT NULL
);
-- Serves ListDuePlayQueue, ListPlayQueuePeople and NextPlayQueueNotBefore: waiting rows by person, highest priority then oldest.
CREATE INDEX IF NOT EXISTS idx_play_queue_person ON play_queue(person_id, priority DESC, queued_at, id) WHERE status = 'queued';
-- Serves ListPlayQueueByTarget, PlayQueueQueuedSince and CountPlayQueueStarted: one target's rows by when they were queued.
CREATE INDEX IF NOT EXISTS idx_play_queue_target ON play_queue(target_type, target_id, queued_at, id);
-- Makes a second match while one waits a no-op: at most one waiting row per auto play per target.
CREATE UNIQUE INDEX IF NOT EXISTS idx_play_queue_waiting ON play_queue(auto_play_id, target_id) WHERE status IN ('queued', 'dispatching');

-- A resume restarts a target's daily count of automatic runs from resumed_at.
CREATE TABLE IF NOT EXISTS play_queue_resumes (
    target_type TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    resumed_by  TEXT NOT NULL,
    resumed_at  INTEGER NOT NULL,
    PRIMARY KEY (target_type, target_id)
);
