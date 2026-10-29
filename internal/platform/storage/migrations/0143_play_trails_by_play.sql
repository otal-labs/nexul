-- A play's runs across every target, newest first: the play page's runs list and the plays list's health (ListPlayRuns, PlayRunHealth).
CREATE INDEX IF NOT EXISTS idx_play_trails_play ON play_trails(play_id, started_at, id);

-- The queue row that started a run, so a play's runs list names the auto play behind each (ListPlayRunsByIDs).
CREATE INDEX IF NOT EXISTS idx_play_queue_trail ON play_queue(trail_id) WHERE trail_id <> '';
