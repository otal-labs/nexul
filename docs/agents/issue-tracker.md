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
- `.scratch/llm-setup/` — wayfinder map complete 2026-09-24 (decisions
  01–16, ADRs 0062–0065) and sliced into implementation tickets 17–38:
  reaching a remote harness through a per-computer tunnel, the MCP-only
  setup gate and its wizard, and the guided lifecycle (ticket people and
  templates, found-in and blocked-by, the testing step, the interview, the
  decisions check). Research findings in `research/`.
- `.scratch/integrations/` — three tracks: Cloudflare deploy→domain,
  branch-driven deployments, LiveKit voice channels. Implemented; tickets 04
  and 11 await the owner's reaction to the built UI. Research findings in
  `research/`.
- `.scratch/plays/` — plays: a pre-configured Agent turn fired from a
  ticket or doc page ("Fix with AI"), selectable memories with their own
  permission, doc threads, and persisted trails. Wayfinder map complete
  2026-09-17 and sliced; tickets 13–33 shipped, including the walkthrough
  on a paired harness (28) and the five follow-ups it produced (29–33).
  Delete the directory once the owner has reacted to the live UI. Research
  findings in `research/`.
- `.scratch/instance-upgrade/` — the in-place instance upgrade through a
  helper container run by the instance runner. Built; the directory goes once
  the owner has run a real upgrade against public packages.
- `.scratch/bots/` — bots: a named poster in any conversation that outside
  systems drive through a Discord-compatible webhook URL. Wayfinder map
  charted 2026-09-16, tickets 01 to 03 resolved, parked 2026-09-20 until the
  repository migration lands; resumes at ticket 04. Research findings in
  `research/`.
- `.scratch/pre-release/` — four standing pre-release items, all open.
  Each keeps a **Surface when** list: the conditions that mean "raise this with
  the owner *before* doing the work". Never act on one silently.
- Specs carried over from the dissolved requirement docs, all `needs-triage`
  because the owner has not re-confirmed them since they were written:
  `.scratch/ticket-workflow-depth/` (blocked tickets, type templates, comments
  and activity), `.scratch/global-search/`, `.scratch/branch-from-ticket/`,
  `.scratch/per-runner-credentials/`, `.scratch/second-git-host/`,
  `.scratch/integration-store/`, `.scratch/workspace-scoped-automation-surfaces/`.
  Each is a `spec.md` with no tickets yet;
  triage first, then `/to-tickets`.

## History

Before 2026-08-15 this repo tracked work as `docs/{ws,cus,dr}-XX.md` workstream
documents with a dashboard in `orchestrator.md`, driven by three GitHub Actions
workflows. All of it landed (PRs #1–#87) and was removed when the repo moved to
this tracker. It's in git history if you need it. On 2026-09-11 the shipped
efforts were removed from this tracker and the old requirement documents
(`docs/TRD.md`, the per-domain `*Domains.md` files, and their open-question
companions) were dissolved into ADRs under `docs/adr/` and specs here.
