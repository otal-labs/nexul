-- Each list tool reads one page in SQL (ADR 0140); each index below walks one list's order to its page instead of sorting
-- every row, and ends in id, the tiebreaker that keeps an offset from skipping or repeating rows that share a second.

-- dead_letter_list, newest first.
CREATE INDEX IF NOT EXISTS idx_dead_letters_created ON dead_letters(created_at, id);
