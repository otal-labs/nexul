# A pressed play asks where to run instead of falling back

A person with no project link for a project ran every play there on their pairing defaults: their default computer
and fallback T3 project (ADR 0102). The fallback T3 project is one checkout, so a play pressed in any unlinked project
ran in that checkout, and nothing in the run dialog said so. The owner kept finding runs in the wrong checkout.

Decision: a play a person starts never falls back. The first run in a project they have not linked asks where (a
computer and a T3 project, with their defaults filled in as the suggestion), and the answer is saved as their project
link for that project. Later runs use the link without asking. Runs nobody presses keep the fallback, because nobody
is there to answer.

- **Who asks.** `Runner.Run` (the run dialog, `play_run`, and Continue on a gone thread's run that recorded no
  location) resolves through `pairing.ResolvePersonRun`. Without a link and without a picked T3 project it refuses with
  `needs_location`, an `ErrInvalid` whose details carry the reason, and keeps no failed trail: being asked is not a
  failed run. A picked computer that is not the linked one needs its T3 project too, so a pick can never land in a
  fallback project either.
- **Who falls back.** Auto plays from the queue, the decisions check when a card enters done, and the decisions
  check's Run again resolve through the starter's link, else their defaults, as before. The decisions check works
  through MCP, not in a checkout, and its Run again has no dialog to ask in.
- **What is saved.** The computer and T3 project, once that computer lists the T3 project; one it does not list is
  refused, and one it cannot be asked about (offline) is refused rather than saved unchecked. The link keeps its own
  model on the same computer, or takes the person's default model on their default computer; the model picked in the
  dialog is for that run only, as its pill says. The link's start-in is kept. The run dialog's **Change** saves the
  same way, so a wrong link is fixed from the run. The Projects tab still edits and clears links; clearing one brings
  the question back.
- **Launch.** The validated computer, T3 project, provider, model, options and start-in travel with the turn, and the
  trail records them (migration 0089). Editing or clearing settings after validation cannot redirect that run, nor
  its follow-up turns: an answer, Continue, and following it again after a restart or on news in T3 Code all go where
  the run started, never where the link points now. A trail from before the migration has no recorded T3 project and
  follows the link as before.
- **Thread.** A new run reuses the target's thread unless that thread is in another T3 project, which starts a fresh
  one, after a Change or a failed one alike; picking the same location again keeps the thread. An answer always goes
  to the thread that asked, whichever T3 project it is in, so a pending question is never left behind.
- **Started again.** Continue on a run whose thread is gone starts the play again where that run ran, rechecking that
  computer and its setup without reading or saving the link. A run with no recorded location asks where, like a
  press, and the web opens the run dialog on **Where to run** with the message filled in. The old trail says the
  play started again only once it has, and otherwise why it did not.
- **Web.** The run dialog shows **Where to run**: two pickers on a first run, one line with **Change** after. The
  harness pill keeps provider and model for the one run (ADR 0058); the computer moved into **Where to run**. Only an
  unlinked project turns an offline or expired computer into the question; a linked one keeps the play disabled
  with that reason.
- **MCP.** `play_run` takes `t3_project_id` beside `computer_id` and saves both as the caller's link, checked as
  above, and its refusal says to ask the user and which tool lists the choices; so does `trail_update`'s continue.
  `computer_list` with one computer's id lists that computer's T3 projects. No new tool.
- **Events.** None. A link is one person's own setting, changed only by them, and their own client refetches it after
  the run that saved it (ADR 0102).

The trade-off: one extra confirm per person per project, and an integration that starts plays through the HTTP
gateway now gets `needs_location` until its user has a link there. `@Agent` mentions still use the defaults in an
unlinked project; a mention has no dialog to ask in.

Rejected: asking only when the fallback T3 project looks wrong for the project, which needs a guess at what "wrong"
means; asking on every run, which turns the common case into two clicks; and saving the pick without running, which
makes the first run a settings detour.

Amends ADR 0102 for plays a person starts, and ADR 0058's run dialog, whose computer pick moved into the location.
Decided 2026-10-10.
