---
title: Memories
description: Write durable Agent context at workspace or project scope, with versions and explicit permissions.
sidebar:
  order: 11
---

A **Memory** is a note for Agent, not a document for clients or requirements.
It has its own page, storage, search boundary, and permission domain. Memories
are not listed with docs.

## Scope and use

Open **Memories** at `/memories`. The **New memory** dialog has a **Project**
selector. Its `Workspace` option creates a workspace-scoped memory. Choosing a
project creates a project-scoped memory.

- A workspace memory reaches every Agent turn in that workspace, including a
  plain chat with no ticket or document.
- A project memory reaches turns for that project.
- A project's list shows workspace memories first, then its own memories.

Each memory has a **Title**, a one-line **When to use** hint, a rich-text body,
and an **Always included in every turn** switch. Always-included memories are
sent in full as standing context. Other memories appear in the turn's index;
the Agent can select one when it needs the body. A single full memory is capped
at 20,000 characters, and selected memories in a play run are capped at 60,000
characters together.

The memory skill card in **Settings → T3 pairing** contains the same protocol
for a user's local T3 sessions. It points agents to `memory_list` and
`memory_get`, and tells them when to use `memory_create` or `memory_update`.

## The interview

Each project has one **interview memory**: its stack, paradigm, testing
strategy, principles, and vocabulary, written as rules. Open it from
**Interview** under the project in the sidebar (`/projects/<prefix>/interview`).
Until it exists the page offers **Start from the template**, which copies the
workspace's Interview template into a new interview memory.

The interview memory is sent in full with every Agent turn in the project,
every play and every `@Agent` mention, ahead of the other always-included
memories. It has no always-included switch and cannot be left out of a play
run. It is capped at 8,000 characters of markdown; the editor counts against
the cap, and a save over it is refused with the count. It versions and
reverts like any memory. A clone of it is an ordinary memory.

The **Interview template** lives in **Settings → Interview template**. A new
workspace starts with one heading per category: stack and versions,
architecture, error handling and logging, testing, code style, dependency
policy, security and secrets, performance budgets, CI gates, branching and
commits, docs and decision records, UI, and vocabulary. It is markdown under
the same cap, and **Reset to default** restores the seeded categories. A
project's interview copies the template once, so editing the template never
changes an existing interview, and editing an interview never changes the
template.

## Edit and version

Open a memory from the list to edit its title, hint, switch, and body. Editing
is explicit: select **Save**. A memory is not live-collaborative like a doc.
Every save appends a version. The version list supports **Revert**, which writes
a new version rather than deleting history.

**Clone to…** copies a memory into another project or directly into a workspace.
The copy is independent and can change without changing the source.

## Permissions and API

The permission catalog has four memory actions:
`memories:read`, `memories:write`, `memories:delete`, and
`memories:clone`. Reading the list or a memory needs `memories:read`.
Creating, editing, and reverting need `memories:write`. Delete and clone use
their matching actions. Agents can write through the same use-case and MCP
permissions as people.

The HTTP routes are `/api/memories`, `/api/memories/{id}`,
`/api/memories/{id}/versions`, `/api/memories/{id}/revert`,
`/api/memories/{id}/clone`, `/api/memories/interview` (create or return a
project's interview), and `/api/memories/interview-template`. MCP registers
`memory_list`, `memory_get`, `memory_create`, `memory_update`,
`memory_delete`, `memory_list_versions`, `memory_revert`, `memory_clone`,
`memory_create_interview`, `interview_template_get`, and
`interview_template_update`. A memory's `kind` is `interview` for the
interview memory and empty otherwise; `memory.created` and `memory.updated`
carry it, and saving the template publishes `interview_template.updated`.
