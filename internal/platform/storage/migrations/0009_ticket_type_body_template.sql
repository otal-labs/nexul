ALTER TABLE ticket_types ADD COLUMN body_template TEXT NOT NULL DEFAULT '';
-- Backfills the seeded default types; the text must match workspace.DefaultTicketTypes, which seeds new projects.
UPDATE ticket_types SET body_template = '## What needs doing


## Acceptance criteria

' WHERE name = 'task' AND body_template = '';
UPDATE ticket_types SET body_template = '## Steps to reproduce


## Expected result


## Actual result


## Provide screenshot

' WHERE name = 'bug' AND body_template = '';
UPDATE ticket_types SET body_template = '## Why


## Acceptance criteria


## Out of scope

' WHERE name = 'feature' AND body_template = '';
