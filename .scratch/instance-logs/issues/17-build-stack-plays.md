# 17: Build stack plays, running on a chosen computer, and "Investigate a crash"

Type: task
Status: resolved
Blocked by: 14

## What to build

From ticket 07: a third play type, stack, with the moments `stack.service_crashed` and
`stack.service_recovered` (and `stack.error_rate_crossed` from ticket 18); conditions on stack,
project and service; Run on a chosen computer for every play type, acting as the person who
picked it, who must own the computer or hold `computers:run_agents` on its share; "didn't run"
with a reason if that right goes; the shipped "Investigate a crash" play and its auto play, off
until a computer is set.

## Surfaces

- MCP: `play_create` and `play_update` take the stack type, its moments and the computer.
- UI: the play editor's type, moments and computer picker.
- Docs: an ADR succeeding 0132 for the third play type; the plays guide.

## Built

- Migration 0121 and ADR 0156 (0132 superseded in part). A stack run's target is the stack, with
  one active run and one daily cap per stack; it reports in a new per-stack conversation and its
  trail is started by the person who picked the computer.
- `run_as` is whoever saved the auto play with a new computer; the right (owner or
  `computers:run_agents`) is checked at pick, at match and at the front of the queue.
- Shared computers show "can't run here yet" in the picker, like every other picker: pairing
  still resolves computers by owner, so a run on a shared one couldn't start. Only the picker
  changes once it can.
- "Investigate a crash" ships off until a computer is picked; existing workspaces get it from
  the migration.
