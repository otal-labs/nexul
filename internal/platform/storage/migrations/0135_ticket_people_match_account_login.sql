-- A ticket's developer and tester are stored as their account spells its login, since the clients compare them exactly.
UPDATE tickets SET developer = (SELECT login FROM users WHERE lower(login) = lower(tickets.developer))
WHERE developer NOT IN (SELECT login FROM users)
  AND (SELECT COUNT(*) FROM users WHERE lower(login) = lower(tickets.developer)) = 1;

UPDATE tickets SET tester = (SELECT login FROM users WHERE lower(login) = lower(tickets.tester))
WHERE tester NOT IN (SELECT login FROM users)
  AND (SELECT COUNT(*) FROM users WHERE lower(login) = lower(tickets.tester)) = 1;
