---
title: Memories
description: Write durable Agent context for a project, with versions and explicit permissions.
sidebar:
  order: 11
---

A **Memory** is a note for Agent, not a document for clients or requirements.
It has its own page, storage, search boundary, and permission domain. Memories
are not listed with docs.

## Scope and use

Open **Memories** under the project in the sidebar (`/<workspace>/memories`). The list puts
always-included memories first under **Pinned**; each row's switch turns
**Always included in every turn** on or off at once, and hovering a row shows
**Clone** and a menu with **Delete**. The **New memory** dialog has a **Project**
selector, and every memory belongs to the project picked there.

- A memory reaches Agent turns for its project: a ticket, a doc, or the
  interview.
- A plain chat with no ticket or document carries no memories.
- A rule meant for several projects is cloned into each of them.

Each memory has a **Title**, a one-line **When to use** hint, a rich-text body,
and an **Always included in every turn** switch. Always-included memories are
standing context: an `@Agent` mention sends them in full, and a play run names
them for the Agent to read first. Other memories appear in the turn's index;
the Agent can select one when it needs the body. An `@Agent` mention inlines at
most 20,000 characters of one memory and 60,000 of all of them together,
dropping the last ones past that. A play run has no such limit, because the
Agent reads each memory itself.

The [setup wizard](/docs/guide/computer-setup/) installs the nexul-memory
skill, which carries the same protocol, on every paired computer. It points
agents to `memory_list` and `memory_get`, and tells them when to use
`memory_create` or `memory_update`. It carries a version, so a Nexul release
that changes it gets rewritten on the next setup, or by the skill itself
through the `skill_get` tool.

## The interview

Each project has one **interview memory**: its stack, paradigm, testing
strategy, principles, and vocabulary, written as rules. Open it from
**Interview** under the project in the sidebar (`/<workspace>/projects/<prefix>/interview`).
Until it exists the page offers **Start from the template**, which copies the
workspace's Interview template into a new interview memory. The page's
**Run the interview** button runs the workspace's Interview play instead: an
Agent asks one question at a time and writes the memory for you, and
**Re-run the interview** amends it later. See [Plays](/docs/guide/plays/).

The interview memory reaches every Agent turn in the project: an `@Agent`
mention carries it in full, ahead of the other always-included memories, and
every play names it for the Agent to read before anything else. It has no
always-included switch and cannot be left out of a play run. It is capped at 8,000 characters of markdown; the editor counts against
the cap, and a save over it is refused with the count. It versions and
reverts like any memory. A clone of it is an ordinary memory.

The **Interview template** lives in **Configuration → Interview template**. Out
of the box it has one heading per category: stack and versions,
architecture, error handling and logging, testing, code style, dependency
policy, security and secrets, performance budgets, CI gates, branching and
commits, docs and decision records, UI, and vocabulary. It is markdown under
the same cap. Until a workspace edits it, the workspace shows the instance's
Interview template and follows it as it changes, and a line over the editor
says which: "Following the instance template" or "Edited for this workspace".
**Reset to instance template** drops the workspace's own and follows the
instance's again. A project's interview copies
the template once, so editing the template never changes an existing
interview, and editing an interview never changes the template.

## Templates

Four kinds of text start a workspace or project off, and each has an instance
version every workspace and project starts from:

| Kind | Key | Lives below the instance in | Below the instance |
|---|---|---|---|
| `interview`, the Interview template | none | each workspace | follows the instance until edited |
| `mention_chip`, how a ticket mention chip renders | none | each workspace | follows the instance until edited |
| `play_instructions`, a built-in play's instructions | the play's built-in key | each workspace's play | copied when the workspace is created |
| `ticket_body`, a ticket type's body template | the type's name (`task`, `bug`, `feature`) | each project's type | copied when the project is created |

A template resolves the code default, then the instance's, then the
workspace's or project's. One nobody edited at the instance is the code
default. A copied template never changes when the instance's does: only
workspaces and projects created afterwards start from the new text. Resetting a
workspace's or project's template gives it the instance's current text;
resetting the instance's gives the code default.

**Clone** copies one template's text from the instance, a workspace, or a
project over another, overwriting it: the instance's Interview into a
workspace, one workspace's Fix with AI instructions into another's, a
workspace's chip layout up to the instance, or a project's `Bug` body template
into another project's `bug` type. Plays match by built-in key and ticket types
by name, ignoring case; when the target has no match the clone fails and says
so.

The instance templates are edited in **Settings → Templates**, under Instance
settings, which shows to anyone holding `templates:write`. It lists the nine
templates under Interview, Mention chip, Play instructions, and Ticket bodies,
each marked Default or with who edited it and when. Each opens the same editor
its kind has in a workspace or project, with **Save**, **Reset to default**,
and **Clone to…**. **Clone to…** also sits beside the workspace and project
editors: **Configuration → Interview template**, **Mention chips**, each
built-in play in **Plays**, and a ticket type's menu in project settings. It
asks where to (the instance, a workspace, or a project), offers only the places
you can edit, and asks before overwriting one that has its own text. A built-in
play or a `task`, `bug`, or `feature` type says whether it matches the
instance's template or differs from it, with **Reset to instance template**
when it differs.

Every member reads the instance templates. Writing one needs
`templates:write`, an instance-level permission held in any workspace, which
the Owner holds. Below the instance each place keeps its own permission: the
Interview template takes `memories:write` in the workspace, the chip
`workspaces:write`, a play `plays:write`, and a body template `projects:write`
in the project. A clone needs read where the source lives and write where the
target does.

Over HTTP, `GET /api/templates` lists the instance templates;
`GET`, `PUT`, and `DELETE /api/templates/{kind}` (with `?key=` or a `key`
field) read, replace, and reset one, and `GET` with `?scope=workspace` and
`workspace_id`, or `?scope=project` and `project_id`, reads it below the instance; and `POST /api/templates/clone` and
`POST /api/templates/reset` work at any layer. MCP has `template_get` and
`template_update`, which take a `scope` of `instance`, `workspace`, or
`project`; `template_update` also takes `reset` or `clone_from`. An instance
change publishes `instance_template.updated`.

## The decisions log

Each project can hold one **decisions log**: the tickets that changed how the
project works, a new pattern, a dropped library, a reversed decision, and why.
Routine tickets add nothing, so it never becomes a changelog. An entry is at
most three lines: the date and the decision, `Why:` in one line, and a link to
the ticket, which links its doc and pull request. When a later decision
reverses an entry, the old entry stays and its first line is marked
`(superseded by <ticket>)`, so the log reads as what is true now.

The log is written by the decisions check (see Plays), not by hand, though it
edits, versions, and reverts like any memory. It is created on its first
entry. It is never always included: it sits in the memory index and an Agent
reads it when it needs the why, since the interview already takes the
every-turn slot.

## Edit and version

Open a memory from the list to edit its title, hint, switch, and body. Editing
is explicit: select **Save**. A memory is not live-collaborative like a doc.
Every save appends a version. The version list supports **Revert**, which writes
a new version rather than deleting history.

**Clone to…** copies a memory into another project, in any workspace you
belong to. The copy is independent and can change without changing the source.

## Permissions and API

The permission catalog has four memory actions:
`memories:read`, `memories:write`, `memories:delete`, and
`memories:clone`. Each is checked on the memory's project. Reading the list or a
memory needs `memories:read`. Creating, editing, and reverting need
`memories:write`. Delete and clone use their matching actions, and a clone also
needs `memories:write` on the destination project. Agents can write through the same use-case and MCP
permissions as people.

The HTTP routes are `/api/memories`, `/api/memories/{id}`,
`/api/memories/{id}/versions`, `/api/memories/{id}/revert`,
`/api/memories/{id}/clone`, `/api/memories/interview` (create or return a
project's interview), and `/api/memories/interview-template`. MCP registers
`memory_list`, `memory_get` (with `version` to read an older version),
`memory_create` (with `clone_from_id` to copy a memory, or `kind`
`interview` to create or return a project's interview), `memory_update`
(with `revert_to_version` to restore one), and `memory_delete`; the Interview
template is read and changed with `template_get` and `template_update` (see
[Templates](#templates)). A memory's `kind` is `interview` for the
interview memory, `decisions_log` for the decisions log, and empty otherwise;
`memory.created` and `memory.updated` carry it, and saving the template
publishes `interview_template.updated`. `memory_create` and
`POST /api/memories` take `kind: "decisions_log"` to create a project's log;
a second one is refused, and `memory_update` adds to the existing one.
`memory.created`, `memory.updated`, and `memory.deleted` also reach the browser
live, so the Interview page shows the Agent's writes as they land.
