-- A computer added a day ago that never enrolled (no name, no session, no runner) was an abandoned install (ADR 0213).
DELETE FROM pairing_computers
WHERE name = '' AND bearer_token = '' AND token_expires_at <= 0
  AND created_at < CAST(strftime('%s', 'now') AS INTEGER) - 86400
  AND NOT EXISTS (SELECT 1 FROM runners r WHERE r.computer_id = pairing_computers.id);
