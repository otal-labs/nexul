-- A ticket's developer and tester hold user ids; a login no account holds stays as typed, for its person to claim on first sign-in.
UPDATE tickets SET developer = (
    SELECT id FROM users WHERE lower(login) = lower(tickets.developer) ORDER BY login = tickets.developer DESC, rowid LIMIT 1)
WHERE developer != '' AND developer NOT IN (SELECT id FROM users)
  AND EXISTS (SELECT 1 FROM users WHERE lower(login) = lower(tickets.developer));

UPDATE tickets SET tester = (
    SELECT id FROM users WHERE lower(login) = lower(tickets.tester) ORDER BY login = tickets.tester DESC, rowid LIMIT 1)
WHERE tester != '' AND tester NOT IN (SELECT id FROM users)
  AND EXISTS (SELECT 1 FROM users WHERE lower(login) = lower(tickets.tester));
