---
title: Documentation and ADRs
description: How CONTEXT.md, the ADRs, the tracker, and the practices files fit together.
sidebar:
  order: 5
---

Nexul treats its own documentation as source of truth, the same
principle the product applies to its users' docs. Three places carry that
weight, each with a distinct job, and nothing is written twice.

## CONTEXT.md — the vocabulary

`CONTEXT.md` is a glossary and nothing else — no specs, no implementation
detail. It defines the product's ubiquitous language (what a "Workspace" is,
what a "Runner" is versus an "Agent", what a "Stack" is). Read it before
anything else; when a term is ambiguous, this is where it's resolved, and
its `_Avoid_` lists name the words that used to mean something different.

## `docs/adr/` — the decision log

Every durable decision lives here as one file, numbered sequentially
(`NNNN-slug.md`). An ADR records *that* a decision was made and *why* — the
context, what was chosen, and what was rejected. Together they are the
answer to "why is it built this way?", and they are the first thing to read
before changing behaviour in an area you don't know.

Record one only when all three are true:

- The decision is **hard to reverse**.
- It is **surprising without context** — a future reader would ask "why not
  the obvious thing?"
- It is **the result of a real trade-off**, not an arbitrary choice.

If it's easy to reverse, skip the ADR — it isn't earning its place.

### Adding one

1. Pick the next sequential number and a short slug:
   `docs/adr/NNNN-slug.md`.
2. State the decision, the trade-off, and what was rejected and why. See
   [`0009-sqlc-generates-the-storage-queries.md`](https://github.com/otal-labs/nexul/blob/master/docs/adr/0009-sqlc-generates-the-storage-queries.md)
   for a worked example — a paragraph is enough. The format, from
   [`docs/adr/README.md`](https://github.com/otal-labs/nexul/blob/master/docs/adr/README.md):

   > An ADR can be a single paragraph — the value is recording *that* a
   > decision was made and *why*, not filling in sections.

3. If the new decision overrides an older one, say so in it explicitly
   rather than quietly leaving both standing.

## `.scratch/` — the requirements

Requirements are not a standing reference document; they belong to the work
that needs them. Each effort gets a directory — `.scratch/<effort-slug>/` —
holding a `spec.md`, numbered tickets under `issues/`, and a `map.md` when
the effort started from open questions. When an effort ships, its directory
is deleted: the code is the implementation, the ADRs are the reasoning, and
git history has the rest.

## `practices/` — how code is written

How code is written, as opposed to what was decided, lives in
[`practices/`](https://github.com/otal-labs/nexul/tree/master/practices): one
file per language or surface (`go.md`, `react-guide.md`, `testing.md`,
`architecture.md`, `design-language.md`, `borrowed-practices.md`). `AGENTS.md`
routes every task to the file it needs, and a change is reviewed against
those files. The [Coding Standards](/docs/contributing/coding-standards/)
page is a digest of them.
