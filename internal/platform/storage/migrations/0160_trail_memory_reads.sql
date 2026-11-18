-- The memories each play run read with memory_get, one row per run and memory at its first read, written as the step
-- lands (ADR 0196). A deleted memory keeps its rows and the title it had, so a run still says what it read; a deleted
-- run takes its rows with it. The runs already stored are read back from their steps in Go after this script.
CREATE TABLE IF NOT EXISTS trail_memory_reads (
    trail_id  TEXT NOT NULL REFERENCES play_trails(id) ON DELETE CASCADE,
    memory_id TEXT NOT NULL,
    title     TEXT NOT NULL,
    read_at   INTEGER NOT NULL,
    PRIMARY KEY (trail_id, memory_id)
);

-- Serves a memory's last read on the Memories list and page, and memory_list.
CREATE INDEX IF NOT EXISTS idx_trail_memory_reads_memory ON trail_memory_reads(memory_id, read_at);
