---
title: Memories
description: Write durable Agent context for a project, with versions and explicit permissions.
sidebar:
  order: 11
---

A **Memory** is a note for Agent, not a document for clients or requirements.
It has its own page, storage, search boundary, and permission domain. Memories
are not listed with docs.

The choice follows purpose, not format. Requirements, product scope, acceptance
criteria, and business processes are docs. Technical research, architecture
guidance, coding standards, and library recommendations meant to guide the
build are memories, even as a long report with citations. A deliverable with
both is split into a doc and a memory that link to each other.

## Scope and use

Open **Memories** under the project in the sidebar (`/<workspace>/memories`). The list has two
folders, **Main** and **Footer**, with required memories first in each. Each row's switch marks
the memory **Required** at once, and hovering a row shows a menu with **Move to**, **Clone**, and
**Delete**. The **New memory** dialog has a **Project**
selector, and every memory belongs to the project picked there.

- A memory reaches Agent turns for its project: a ticket, a doc, or the
  interview.
- A plain chat with no ticket or document carries no memories.
- A rule meant for several projects is cloned into each of them.

Each memory has a **Title**, a one-line **When to use** hint, a rich-text body,
and the **Required** switch on its row in the list. Required memories are
standing context: every Agent turn in the project names them, by title and id,
for the Agent to read with `memory_get` before it starts, and a follow-up in
the same conversation does not name them again. No turn carries a memory's
body, so memory size never limits a turn. The Agent finds the other memories
with `memory_list` when their when-to-use fits the task.

A memory in the **Footer** folder concludes a play run; move one there with the
row's **Move to**. A play's run dialog lists footer memories in their own
**Footer** section at the bottom. A run names the picked ones, and any required
one, last, after its instructions, and the Agent reads them once the work is
done to conclude the run: for example, which column the ticket now belongs in.
The interview and the decisions log always stay in Main.

The [setup wizard](/docs/guide/computer-setup/) installs the nexul-memory
skill, which carries the fuller protocol, on every paired computer. It points
agents to `memory_list` and `memory_get`, and tells them when to use
`memory_create` or `memory_update`. It carries a version, so a Nexul release
that changes it gets rewritten on the next setup, or by the skill itself
through the `skill_get` tool.

## The interview

Each project has one **interview memory**: its stack, paradigm, testing
strategy, principles, and vocabulary, written as rules. Open it from
**Interview** under the project in the sidebar (`/<workspace>/projects/<prefix>/interview`).
The page lists the Interview template's questions, one open at a time: each
**Next** or **Skip** saves, and any answered or skipped question opens again on
a click. Once every question is answered or skipped, **Done** runs the
workspace's Interview play: an Agent asks follow-ups about what your answers
and the code leave open, then writes the memory shown beside the questions;
**Regenerate** runs it again later. Follow-ups the Agent asked appear under the
questions, one section per round. See [Plays](/docs/guide/plays/).

The interview memory reaches every Agent turn in the project: every mention and
every play names it, ahead of the other always-included memories, for the Agent
to read before anything else. It has no
always-included switch and cannot be left out of a play run. It is capped at 8,000 characters of markdown; the editor counts against
the cap, and a save over it is refused with the count. It versions and
reverts like any memory. A clone of it is an ordinary memory.

The **Interview template** lives in **Configuration → Interview template**. It
is the list of questions the interview asks, in markdown: a `##` heading per
question, the text under it as a hint, then `- ` bullets for single-choice
options or `- [ ]` bullets for multi-select ones, with the text after `: `
describing an option. A question with no bullets takes free text. Out of the
box it asks twelve questions only a person can answer: languages and
frameworks, how the code is organised, how errors travel, when tests are
written and which a change needs, style rules, dependencies, secrets, how a
change reaches the main branch, where decisions are written down, the user
interface, and the project's own words. A save is refused, naming the line,
when text comes before the first question, a heading is empty, a question is
asked twice, or there are none. The template holds up to 32,000 characters,
and the editor shows how many questions it has. Until a workspace edits it, the workspace shows the instance's
Interview template and follows it as it changes, and a line over the editor
says which: "Following the instance template" or "Edited for this workspace".
**Reset to instance template** drops the workspace's own and follows the
instance's again. The template is never copied into an interview memory, which
starts empty, so editing one never changes the other.

## Templates

Four kinds of text start a workspace or project off, and each has an instance
version every workspace and project starts from. A fifth, `agent_prompt`, lives
only at the instance:

| Kind | Key | Lives below the instance in | Below the instance |
|---|---|---|---|
| `interview`, the Interview template | none | each workspace | follows the instance until edited |
| `mention_chip`, how a ticket mention chip renders | none | each workspace | follows the instance until edited |
| `play_instructions`, a built-in play's instructions | the play's built-in key | each workspace's play | copied when the workspace is created |
| `ticket_body`, a ticket type's body template | the type's name (`task`, `bug`, `feature`) | each project's type | copied when the project is created |
| `agent_prompt`, the Intro and Footer every full Agent prompt opens and closes with | `intro` or `footer` | nowhere | the instance's text is used for every turn |

The Intro tells the Agent who it is and to check it can reach Nexul; the Footer
holds the standing rules for ticket bodies and memories. An emptied one leaves
its part out of the prompt, and it cannot be cloned anywhere, because there is
no layer below the instance to clone it to.

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
settings, which shows to anyone holding `templates:write`. It lists the eleven
templates under Interview, Mention chip, Play instructions, Ticket bodies, and
Agent prompt,
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
entry. It is never always included: an Agent finds it with `memory_list` and
reads it when it needs the why, since the interview already takes the
every-turn slot.

## Edit and version

Open a memory from the list to edit its title, hint, and body. Editing
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

The interview's answers are stored per project, one per question, apart from
the interview memory: deleting the memory keeps them, deleting the project
removes them. `GET /api/memories/interview-answers?project_id=` lists them,
`PUT /api/memories/interview-answers` saves one, and
`POST /api/memories/interview-answers/skip` and `/clear` skip or clear one;
reading needs `memories:read` and answering `memories:write`. `memory_get` on
the interview memory returns them as `answers`, and `memory_update` takes
`answers` to save or skip the template's questions. Each change publishes
`interview_answer.saved` or `interview_answer.cleared`, naming the question and
its author but never the answer, and reaches the project's readers live.

An interview can also be pointed at sources: a file or folder in the project's
checkout, a doc, a memory, another project in the workspace, or pasted text.
Each has a stance, follow or question. A follow source is what drafting reads
to propose answers; a question source is only asked about. A project holds at
most 50 sources, pasted text up to 32,000 characters, and a path is relative to
the checkout. Adding a doc, memory, or project also takes read access on it,
and someone who cannot read one sees only its kind, flagged not visible. A
deleted one stays listed, flagged gone. Drafts are proposed answers kept apart
from the answers, one per question, and a draft equal to the stored answer is
dropped. `GET`, `POST /api/memories/interview-sources`, and
`PATCH` and `DELETE /api/memories/interview-sources/{id}` list, add, change,
and remove sources; `GET /api/memories/interview-drafts` and
`DELETE /api/memories/interview-drafts/{id}` list and dismiss drafts. Reading
needs `memories:read` and changing `memories:write`. `memory_get` on the
interview memory returns `sources`, pasted text included, and `drafts`, and
`memory_update` takes `add_sources`, `update_sources`, `remove_sources`,
`drafts`, and `dismiss_drafts`. Changes publish `interview_source.added`,
`.changed`, `.removed`, `interview_draft.saved`, and `.dismissed`, never a
source's content, and reach the project's readers live.

On the Interview page the sources sit above the questions, each with its
stance and a remove control, and "Add source" takes any kind; a dropped text or
markdown file becomes pasted text named after the file. "Draft answers" shows
once a follow source exists and carries a warning dot when a source was added,
or a doc, memory, or pasted text changed, since the last drafting run started.
A drafted question opens with the draft picked and where it came from, so Next
confirms it. A draft on an answered question shows as a suggested change:
Accept saves it as the answer, Dismiss deletes the draft.
