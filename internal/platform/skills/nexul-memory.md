---
name: nexul-memory
description: Use Nexul's memories — durable notes for agents, each in one project, own entity from docs — before and during any task.
---

# Nexul memory protocol

Memories are durable, human-editable notes for agents, not people: their own
entity with a table, page, and permission, never a doc. Each belongs to one
project and reaches only that project's turns; a chat with no ticket or doc
has none. A turn never carries a memory's body. It names the memories to read
first, by name and id: the project's always-included ones in every turn, the
interview memory leading, plus the ones picked for a play run. This skill
applies the same protocol in your own local T3 sessions.

## Read

1. When the turn names memories to read first, read each with `memory_get`
   before anything else and follow them as standing rules.
2. For the rest, call `memory_list` for the current project.
3. Pick the memories whose when-to-use line matches what you're about to
   do — don't fetch everything, only what's relevant.
4. Fetch a chosen memory's full content with the `memory_get` MCP tool,
   passing its id.

## Write

Save a memory when:

- You learn a durable fact worth remembering for next time (a coding
  convention, a gotcha, a standing preference) — use your own judgment.
- A user says something like "@Agent remember X" — always save when asked,
  even if it seems minor.

Create a new memory with `memory_create`, passing the `project_id` of the
project it belongs to. A fact that holds for several projects is saved in
each, or copied with `clone_from_id`. Update an existing one with
`memory_update` if one already covers the same ground. Humans can always
edit memories directly on the Memories page — that's the safety valve for a
wrong or stale memory.

## Doc or memory

Route by purpose, never by format or length. Ask who the content is for and
what it is meant to do:

- A doc (`doc_create`) is for people deciding what to build: requirements,
  product scope, acceptance criteria, business processes, and
  project-management deliverables such as a plan or a status report.
- A memory (`memory_create`) is for agents building it: reusable technical
  research, architecture guidance, coding standards, dependency and library
  recommendations, and practices within a library.

A long technical report with citations is still a memory when its purpose is
to guide implementation, such as research into a C++ architecture, which
libraries to use, and how to use them. A short list of what a client's
checkout must do is still a doc. A deliverable that holds both is split: the
requirements go in a doc, the technical guidance in a memory, and each body
links the other: the doc names the memory's title and id, the memory names the
doc's title and id.

- A destination the person named, a doc or a doc folder, wins over this rule.
- Give a technical memory a when-to-use line narrow enough that it is read
  only when relevant, and link it from where the work starts: the ticket's
  thread with a note, or the project's rules when it applies to most work.
- Saving keeps meaning: a recommendation stays a recommendation, not a
  decision.
- Scope follows the request. A research request ends with the findings
  saved; an implementation request includes the dependency and schema choices
  it needs, with no separate approval round.

## Curate

Keep the set small and specific. Prefer updating an existing memory over
creating a near-duplicate. A when-to-use line should be short enough to
scan in a list: "use this if you are writing React code", not a paragraph.

## Staying current

This file's `metadata.version` names the version you have. Once per session,
when Nexul's MCP tools are connected, call `skill_get` with `name`
`nexul-memory`. If the version it returns differs from this file's, write the
`content` it returns over this file at every path it lists, then follow the new
text instead of this one.
