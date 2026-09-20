# Domain Docs

How the engineering skills should consume this repo's documentation when
exploring the codebase.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root — the ubiquitous language, and nothing
  else. Single-context repo; there is no `CONTEXT-MAP.md`.
- **`docs/adr/`** — the decision log. One file per decision,
  `NNNN-slug.md`, recording what was decided and the trade-off behind it. Read
  the ones touching your area before changing behaviour there.
- **`.scratch/<effort>/spec.md`** — the requirements for the work in front of
  you. The tracker holds the specs; there is no separate requirements document.

If a file doesn't exist, **proceed silently**. Don't flag its absence or suggest
creating it upfront. `/domain-modeling` creates them lazily when terms or
decisions actually get resolved.

## File structure

```
/
├── CONTEXT.md              ← ubiquitous language (glossary only)
├── docs/
│   ├── adr/                ← every durable decision, NNNN-slug.md
│   └── agents/             ← this file and its siblings
├── .scratch/               ← the issue tracker: specs and tickets
└── internal/<domain>/      ← the code
```

## Use the glossary's vocabulary

When your output names a domain concept — an issue title, a refactor proposal, a
hypothesis, a test name — use the term as defined in `CONTEXT.md`, and respect
its `_Avoid_` lists.

If the concept you need isn't in the glossary yet, that's a signal: either
you're inventing language the project doesn't use (reconsider), or there's a
real gap (note it for `/domain-modeling`).

## Flag conflicts, don't override

If your output contradicts an ADR or a tracker spec, surface it explicitly:

> _Contradicts ADR 0003 — but worth reopening because…_

The hard rules in `AGENTS.md` win over everything, including these docs. If a
practice doc disagrees with a hard rule, the hard rule is right.
