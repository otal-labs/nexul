-- Whether a service's container has a healthcheck, which is what lets a deploy swap it with no downtime; NULL until a
-- runner's observation report says, so a service deployed before this column reads as unknown, not as none.
ALTER TABLE services ADD COLUMN healthcheck INTEGER;
