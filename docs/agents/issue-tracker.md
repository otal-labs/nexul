# Issue tracker: Local Markdown

Issues and specs for this repo live as markdown files in `.scratch/`. They are
**committed to the repo**, not gitignored — the tracker is part of the codebase.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- The spec is `.scratch/<feature-slug>/spec.md`
- Implementation issues are one file per ticket at
  `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01` — never a
  single combined tickets file
- Triage state is a `**Status:**` line near the top of each issue file (see
  [`triage-labels.md`](./triage-labels.md) for the role strings)
- Blocking edges are a `**Blocked by:**` line listing the numbers/titles it
  depends on, or "None — can start immediately"
- Comments and conversation history append to the bottom under a `## Comments`
  heading

## When a skill says "publish to the issue tracker"

Create a new file under `.scratch/<feature-slug>/`, creating the directory if
needed.

## When a skill says "fetch the relevant ticket"

Read the file at the referenced path. The path or issue number is normally
passed directly.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a file with one **child** file per ticket.

- **Map**: `.scratch/<effort>/map.md` — the Notes / Decisions-so-far / Fog body.
- **Child ticket**: `.scratch/<effort>/issues/NN-<slug>.md`, numbered from `01`,
  with the question in the body. A `Type:` line records the ticket type
  (`research`/`prototype`/`grilling`/`task`); a `Status:` line records
  `claimed`/`resolved`.
- **Blocking**: a `Blocked by: NN, NN` line near the top. A ticket is unblocked
  when every file it lists is `resolved`.
- **Frontier**: scan `.scratch/<effort>/issues/` for files that are open,
  unblocked, and unclaimed; first by number wins.
- **Claim**: set `Status: claimed` and save before any work.
- **Resolve**: append the answer under an `## Answer` heading, set
  `Status: resolved`, then append a context pointer to the map's
  Decisions-so-far in `map.md`.

## What lives there now

Only efforts with genuinely open work stay here. An effort whose tickets are
all resolved and whose map or spec says complete gets deleted — the work is in
git history, and anything durable it decided is an ADR.

- `.scratch/latent-bugs/` — small defects seen in passing, one heading each
  in `spec.md`; no tickets, fix and delete the heading.
- `.scratch/t3-orchestrator-v2/` — T3 Code's new orchestrator (protocol 2),
  built and shipped; two tickets are left: 16, the owner's walkthrough on the
  nightly, and 17, the guide pages and glossary terms. Wire research in
  `research/`.
- `.scratch/personal-runners/` — a personal runner on each person's own
  computer: one command adds the computer, Nexul reaches T3 Code through the
  runner's connection instead of a Cloudflare tunnel (ADR 0146), the old
  tunnel and URL pairing retired after existing computers move; later shell
  jobs, container logs and sharing a computer. Spec and tickets 01–20
  written 2026-10-10.
- `.scratch/computer-failover/` — a person's computers in one order, runs going to the
  first ready one, runs at once per computer, and the Computers and Projects tabs
  rebuilt around them (ADRs 0165, 0166). Left: 04, clone automatically, and 05,
  a chosen-computer auto play's ordered list.
- `.scratch/mcp-parity/` — web capabilities no MCP tool reaches yet, one
  bullet each, `needs-triage`.
- `.scratch/pre-release/` — four standing pre-release items, all open.
  Each keeps a **Surface when** list: the conditions that mean "raise this with
  the owner *before* doing the work". Never act on one silently.
- Specs carried over from the dissolved requirement docs, all `needs-triage`
  because the owner has not re-confirmed them since they were written:
  `.scratch/ticket-workflow-depth/` (what is left is an activity timeline and
  custom fields), `.scratch/global-search/` (the command palette covers tickets
  and docs; the all-entity search is not built), `.scratch/branch-from-ticket/`,
  `.scratch/second-git-host/`, `.scratch/integration-store/`, and
  `.scratch/workspace-scoped-automation-surfaces/` (automations are scoped;
  integrations are not). Each is a `spec.md` with no tickets yet;
  triage first, then `/to-tickets`.

## History

On 2026-10-14 the efforts whose tickets had all shipped were removed: LLM
setup, integrations, plays, the phone app, ticket flow, the interview as
questions and clarifying a doc.
Later that day invitations followed, after its walkthrough on a real install:
role descriptions, the invite page before and after sign-in, and existing
accounts turned away.

Before 2026-08-15 this repo tracked work as `docs/{ws,cus,dr}-XX.md` workstream
documents with a dashboard in `orchestrator.md`, driven by three GitHub Actions
workflows. All of it landed (PRs #1–#87) and was removed when the repo moved to
this tracker. It's in git history if you need it. On 2026-09-11 the shipped
efforts were removed from this tracker and the old requirement documents
(`docs/TRD.md`, the per-domain `*Domains.md` files, and their open-question
companions) were dissolved into ADRs under `docs/adr/` and specs here.
