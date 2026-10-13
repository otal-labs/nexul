# 01: How the Repository step's choices and the early Done screens look

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

The Repository step gets two choices beside the repository search,
replacing the muted "Skip for now": "No repository yet" and, once a
repository is picked, "Attach without deploying". How do they look and read?

- Where they sit relative to the search, and how much weight they get
  against "pick a repository to deploy" without competing with it.
- How "Attach without deploying" appears: on the picked row, or as a
  second action once the scan returns.
- What the progress row shows when an early exit jumps from Repository to
  Done.
- The Done screen for each early exit: its line ("<Project> is ready for
  docs and tickets", "<repo> is attached; deploy it whenever it's ready"),
  the interview offer, and the onward links (Docs, Board, Invite people).
- Copy that assumes a repository today: the first-project subtitle "Name
  it, then point Nexul at its repository" and the Repository step's
  description.

Prototype with `design-mode` against the existing wizard kit
(`WizardFooter`, `WizardStepPanel`, `EmptyState`); the owner picks by
looking.

## Answer

Decided in auto design mode, 2026-10-13. The prototype is on the pushed branch `proto/project-paths`, at
`/<ws>/proto/project-paths?scene=search|picked|done-none|done-attached&v=1..3&d=1..3`, and is never merged. The
locked references are in `~/Code/Images/project-paths/`.

**The step ("On the row", 13/15)**
- The search stays the main thing.
- Below the results sits a muted line, "No code to point at?", followed by a medium-weight ink button "No repository
  yet →". The "Don't see a repository?" hint comes after it. The footer keeps only Back; Skip for now leaves this
  step.
- Picking a repository scans it, then opens the row in place:
  - The row gets a `bg-accent` fill and a 2px `brand` left edge.
  - Under the name sits one muted mono line, `main · ✓ compose.yaml · web, worker, postgres`, with the check in
    `text-success`.
  - Right-aligned are a ghost `sm` "Attach without deploying" and a primary `sm` "Deploy it".
  - Picking another row moves it.
  - Picking now waits for a choice, which costs one more click on the deploy path. That is accepted.
- The tests question stays on the step for "No repository yet".

**The progress row after an early exit**
- Service, Reach and Deploy branches are recorded as skipped and drawn as skipped (a dashed ring with a dash), and
  the line fills to Done.
- Repository is drawn skipped after "No repository yet" and done after "Attach".
- Continue setup lands on Done.

**Done ("Next places, marked row")**
- The headline is `text-sm` medium, and verbatim either:
  - "<Project> is ready for docs and tickets."
  - "`<owner/repo>` is attached; deploy it whenever it's ready." (the repository in mono)
- "Left for later" folds deploying into one row: "Deploying · needs a repository · Open Repository", or
  "Deploying · not started · Deploy it".
- The interview offer is unchanged.
- "Where to next" lists three hairline rows (icon, label, muted reason, chevron):
  - Docs, "Write the brief everyone reads first."
  - Board, "Add the first tickets."
  - Invite people, "Bring in the client and the team."
- Every onward row finishes setup on the way out.
- The footer has Back and Finish. Finish stays an outline button while the interview offer is pending.

**Copy**
- The Repository step's description becomes "Pick the repository this project's code lives in. Nexul reads its
  Dockerfile or compose file to deploy it."
- The new-project subtitle becomes "Name the project, then pick its repository if it has one."
- The first-project subtitle becomes "Tickets, docs, and deploys all live in a project. Name it, then pick its
  repository if it has one."

**Runners-up**
- For the step, "Footer" (12/15, `?v=1`): the two choices as ghost buttons where Skip for now was.
- For Done, "Per step" (`?d=1`): each skipped deploy step as its own row.

**Not verified**: the real search, scan and attach (the prototype mocks them), motion (left to the build), and the
first-project Info step.
