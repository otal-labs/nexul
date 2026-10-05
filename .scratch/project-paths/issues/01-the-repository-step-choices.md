# 01: How the Repository step's choices and the early Done screens look

Type: prototype
Status: open
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
