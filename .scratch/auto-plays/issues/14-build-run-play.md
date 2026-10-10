# 14: Build runPlay in the SDK and the automations host

Type: task
Status: open
Blocked by: 04, 10

## Question

Build what 04 decided: `runPlay` in the SDK with the play-name union, the
host and server side of the call through the queue, `createMockContext`
support, SDK tests, and the SDK docs page.

## Notes

- Includes the unique play names from 04's answer: the rename migration
  for duplicates, the unique index, and the taken-name error on create and
  rename in the dialog and MCP.
- `play_queue` has `auto_play_id` but no origin columns; `runPlay` needs
  `origin_kind`/`origin_id` (or a nullable `automation_id`) in a new
  migration, and the re-check and the ticket lines must handle a run with
  no auto play.
