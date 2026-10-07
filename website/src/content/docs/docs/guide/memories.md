---
title: Memories
description: Write down what agents should know about a project, and choose which notes every agent turn reads first.
sidebar:
  order: 11
---

A memory is a note for agents, not for people: a coding rule, a library choice, how to run the tests. Requirements, scope, and acceptance criteria go in a doc. Guidance meant to steer the build goes in a memory, even when it's a long report.

Open **Memories** under the project in the sidebar. Every memory belongs to one project and reaches only that project's agent turns. A rule meant for several projects is cloned into each.

## Write a memory

1. Press **New memory** and pick the **Project**.
2. Give it a **Title** and a one-line **When to use**, such as "writing React code". Agents decide whether to read a memory from this line, so make it specific.
3. Switch on **Always included** if every agent turn in the project should read it.
4. Write the body and save.

To edit one, open it, change the title, hint, or body, and press **Save**. A memory isn't edited live like a doc: every save adds a version under **Versions**, and **Revert** on an older version saves it again as the newest, so nothing in the history is lost.

## How agents find memories

An agent turn never carries a memory's text. It names memories for the agent to read:

- **Required** memories (the switch on each row) are named in every agent turn in the project, after the [interview memory](/docs/guide/interview/), which always comes first.
- A play's run dialog lets you add more for that run.
- The agent finds the rest itself by their **When to use** line.
- A plain chat with no ticket or doc names no memories.

Because only names travel, a long memory never stops a turn.

Every computer's [setup](/docs/guide/computer-setup/) installs the nexul-memory skill, which tells agents how to look memories up and when to write or update one. Agents may write memories too; every save keeps its author.

## Main and Footer

The list has two folders. **Main** holds everyday memories. **Footer** holds memories that conclude a play run: the agent reads them once the work is done, for example to decide which column the ticket belongs in now. Move a memory between them with **Move to** in its row menu. A play's run dialog lists footer memories in their own section.

## Clone or delete

**Clone** in the row menu, or **Clone to…** on an open memory, copies it into another project in any workspace you belong to. The copy is independent from then on. **Delete** removes it.

## The decisions log

A project can have a decisions log: a memory listing only the tickets that changed how the project works, such as a new pattern, a dropped library, or a reversed decision. Each entry is at most three lines: the date and the decision, `Why:` in one line, and a link to the ticket. When a later decision reverses an entry, the old one is marked `(superseded by <ticket>)`.

The [decisions check](/docs/guide/plays/#the-decisions-check) writes it as tickets reach Done. You can edit it like any memory. It is never always included; agents look it up when they need the why.

## Templates

Some text starts each workspace or project off: the interview questions, how a ticket mention chip looks, each built-in play's instructions, and each default ticket type's body template. Each has an instance version every new workspace and project starts from. Two more, the Intro and Footer that open and close every agent prompt, live only at the instance.

Edit the instance's in **Settings → Templates**, under Instance settings, with `templates:write`. Each opens in its editor with **Save**, **Reset to default**, and **Clone to…**.

| Template | Where a workspace or project edits it | Follows instance changes? |
|---|---|---|
| Interview | **Configuration → Interview template** | yes, until edited |
| Mention chip | **Configuration → Mention chips** | yes, until edited |
| Play instructions | each built-in play in **Configuration → Plays** | no, copied when the workspace is made |
| Ticket body | a ticket type in project settings | no, copied when the project is made |

A copied template shows whether it matches or differs from the instance's, with **Reset to instance template** when it differs. **Clone to…** copies a template over another of the same kind, at the instance, a workspace, or a project, and asks before overwriting one with its own text.

## Permissions

`memories:read`, `memories:write`, `memories:delete`, and `memories:clone`, each checked on the memory's project. Cloning also needs `memories:write` on the project it lands in. Agents use the same permissions through `memory_list`, `memory_get`, `memory_create`, `memory_update`, and `memory_delete`.
