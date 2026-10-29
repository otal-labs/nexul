# How many versions a history keeps

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Should histories be capped so they don't pile up (some docs already have more
than 50 versions), and if so, what does the cap count and what does it never
delete?

## Answer

- Each body history (docs, ticket descriptions, plays, memories,
  automations) keeps its last 10 unnamed versions. Saving an 11th prunes the
  oldest one.
- Named versions (a doc's milestones) never count toward the 10 and are
  never pruned.
- An automation's active and pending versions are never pruned, since they
  are what runs and what waits to run.
- A memory's revert can reach only the versions that are kept.
- A ticket's field changes (status, developer, due date and so on) are a
  separate list with a larger cap of 100. Those rows are about 100 bytes each,
  and their value is in reaching back past the last day of board work.
- Why: a cap keeps the history readable and bounds the table. The space
  itself is small (50 versions of a typical 500-byte doc is about 25 KB),
  and skipping saves that change nothing is the other half of the fix
  ("When editing a doc writes a version").
