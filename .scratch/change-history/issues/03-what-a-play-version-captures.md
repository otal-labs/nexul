# What a play version captures

Type: grilling
Status: resolved
Blocked by: 01

## Question

Which of a play's fields does a version hold (instructions, name, trigger,
target, model settings), what writes a new one (every save, or only a change
to the instructions), and do the built-in plays' stored instructions count as
the first version?

## Answer

- A play has the same shape of history as a ticket. Its instructions are a
  body history: each save that changes them writes a version with a text
  diff, and the last 10 are kept. Label, description, enabled, the stage it
  shows at, and its excluded projects are one-line field changes, kept up
  to 100.
- The play editor is a plain form, not live, so every save that changes
  something is one change; the sitting rule does not apply.
- Plays that exist on ship day get their current instructions as a first
  version, marked as of that day. A built-in play nobody has edited is
  marked "Built-in default".
- Each trail records the instructions version its run used, so a trail can
  show "ran with version 7" and diff it against the current instructions.
  If that version has since been pruned, the trail says it is no longer
  kept.
