# 10 — The testing step

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

After docs, code, and review, the work should be deployed somewhere testers
can reach it — the owner wants Nexul to help everyone, not just developers.
Branch deploy rules with preview deployments already exist; what does the
testing step add on top?

- Who tests: manual testers, QA, the agent itself, or all three? What does a
  non-developer need to see (a URL, the ticket, a checklist?) to test a
  ticket?
- e2e strategy questions belong to the step-0 interview (same repo or a
  separate e2e repo, coverage) — this ticket decides what Nexul *does* with
  those answers during the lifecycle: does a ticket get a "testing" signal,
  a deploy link on the ticket, an assigned tester?
- How a failed test feeds ticket 09's bug flow, and how a passed test moves
  the ticket toward done.

## Comments

Facts and decisions to start from (2026-09-23): the board already has a
fixed `testing` stage between review and done. The bugs grilling settled
that a bug found before done moves the card back to progress, a bug found
after done is a new ticket with a required "found in" link, and agents may
file bugs as QA — reported as "Nexul · for <person>".

## Answer

Grilled with the owner 2026-09-24.

**What a tester gets.** A ticket in a testing-stage column shows a **Test
this** panel, written for non-developers: where to test (the live URL), what
to check (the ticket's **acceptance criteria** — every feature and task
template now carries that section, see the rename below), and **Pass** /
**Fail**.

- **Which URL — never production** (owner, 2026-09-24: testers do not test
  on prod). The panel offers only a deployment that does not touch
  production's services: the branch's own preview deployment, or failing
  that the shared test environment the branch deploys to (labelled "shared,
  may include other changes"). A deployment counts as production-touching
  when it is the default branch's own, or when its row shares the default
  branch's network with no overrides; those are never offered. With nothing
  safe, the panel says there is no test environment and nudges to add a
  deploy branch on its own network.
- **Pass** moves the card to the first done-stage column and records who
  tested it.
- **Fail** opens a dialog with the bug template (steps, expected, actual,
  screenshot), posts the result to the ticket's own thread, and moves the
  card back to progress — the before-done rule from the bugs ticket; the
  fixing agent reads the thread.

**Three people on a ticket.** *Assignee* is renamed **Developer**, and
**Tester** and **Reporter** join it. Developer and Tester are one person
each, optional; Reporter is set at creation and never edited (agent-filed
tickets show "Nexul · for <person>"). Cards show the developer, or the tester
in testing columns. Anyone who can see a ticket may still press Pass or
Fail; the Tester field gives QA people a "waiting for me to test" view.

**Bugs out of the create dialog.** The normal create dialog offers feature
and task only. Bugs are born from **Report a bug** on a ticket, from
**Fail**, or from a board-level **Report a bug** that allows ticking "origin
unknown" — the agent is then told the origin is unknown rather than handed a
guessed link. The MCP create tool keeps accepting bugs on the same terms.

**Template rename.** Feature's "What done looks like" and task's "Done
when" both become **Acceptance criteria**, the thing a tester tests against.

**Test with AI.** A seeded ticket play shown only in the testing stage. The
agent follows the interview memory's testing strategy: it checks the live
URL against the acceptance criteria and runs the project's tests, then
presses Pass or Fail exactly as a person would. Where the interview calls for
an automated end-to-end suite, it also adds or extends a test covering the
ticket's acceptance criteria.

**A tests repository.** A project may attach a second repository marked
**tests**, never deployed, so the one-deployable-repository rule holds. The
project wizard's repository step asks whether tests live in the same
repository or a separate one, and the answer pre-fills the interview.

**Deploy branches step in the project wizard.** Previews per branch were the
missing piece of the existing wizard (project, repository, service, env,
reach, done). A new step lists branch rows, each showing the URL it will
serve:

- The first row is seeded with the repository's default branch and the
  service's hostname.
- Wildcard rows such as `feature/*` → `*.example.com` or `staging/*`,
  each with a live example (`feature/security-test` →
  `security-test.example.com`). Branch names are made URL-safe: dots,
  slashes, and capitals become dashes and lowercase, trimmed to hostname
  limits (`feature/dot.test` → `dot-test.example.com`).
- **Each row picks its network explicitly** (owner: staging wants the
  production network, feature branches the QA one). The picker lists the
  machine's networks with what runs on each ("qa_default — postgres,
  redis"), defaulting to the default branch's network.
- **The network usually picks the database**: when the app reaches its
  database by container name, the chosen network decides which database
  answers. For a database outside docker (managed, raw address) that does
  not work, so each row also has an optional **Overrides** list under
  Advanced (key = value, e.g. `DATABASE_URL`) that replaces the default
  branch's values for that branch only — the per-branch overrides ADR 0036
  anticipated.
- A row sharing the default branch's network with no overrides shows a
  warning: "Uses production's services, including its database — testers
  are never sent here." Such a row still deploys; it just never becomes a
  test target.
- Skippable, like every wizard step.
