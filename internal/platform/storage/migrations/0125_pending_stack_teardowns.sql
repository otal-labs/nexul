-- A deleted stack's teardown waiting for its machine's runner to connect; the row goes once the runner has been sent it.
CREATE TABLE IF NOT EXISTS pending_stack_teardowns (
    machine    TEXT NOT NULL,
    slug       TEXT NOT NULL,
    strategy   TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (machine, slug)
) WITHOUT ROWID;
