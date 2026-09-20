---
title: Docs, Tickets, and the Board
description: How documents drive work, how tickets track it, and how the board organizes it.
sidebar:
  order: 10
---

Nexul's loop starts with a document, not a ticket. Docs are the source of truth; tickets are scoped, actionable units of work derived from them.

## Docs

A document belongs to exactly one project, the same rule a ticket follows. Each project's sidebar has two children: **Board** and **Docs**.

- **Rich editor.** The canonical representation is structured rich text, not raw markdown — you get real formatting and styling. Markdown is a conversion surface: it's what LLMs, imports, exports, and integrations read and write, converted to and from the structured document automatically.
- **`@` mentions.** Typing `@` opens an autocomplete of relevant tickets and other documents. Picking one inserts a live reference: a ticket mention renders as a chip showing its id and status, a document mention shows its current title. If you can't access what's tagged, the chip stays visible but inert — you see the title, not the content.
- **Real-time collaboration.** Multiple people can edit the same document at once. Changes appear live, presence shows who's viewing or editing (with cursors and selections), and normal concurrent edits merge automatically. A conflict prompt only appears for edits that genuinely can't be merged. Offline edits sync once you reconnect.
- **Versions.** Lightweight history is kept for recovery, plus deliberate named versions for milestones you want to come back to — not a version for every keystroke.
- **Attachments.** Paste, drop, or use the `/image` slash command to attach files. Everything the doc owns is listed under the body with download and delete; deleting the doc deletes its files too.

## Tickets

Tickets don't have to come from a document — you can create one directly — but a doc can spawn tickets scoped to a specific slice of work.

- **Types.** Owners define ticket types (`bug`, `feature`, `task`, or anything project-specific), each with its own icon and color.
- **Branches and PRs.** From a ticket's Development section you can create a new branch (choosing repository and base branch) or link an existing branch or PR. A branch or PR can link to more than one ticket. Pull requests live here, on the ticket page — there's no separate pull requests page.
- **Finishing.** A ticket is finished once it has at least one linked PR, none of its linked PRs are still open, and at least one was merged. PRs closed without merging don't block or count toward this.
- **Attachments.** Same file list as a document — paste, drop, or `/image` into the description, and the ticket's Attachments list tracks everything added.

## The board

Tickets live on one Kanban board per project — there's no combined all-projects view; visiting the board with no project picked lands you on your last-viewed project.

- **Statuses.** Fully configurable, not hardcoded to open/in-progress/done. Every status belongs to one of five fixed stages, in order: backlog, progress, review, testing, done. A project can skip a stage entirely by having no columns in it. Only the done stage is terminal.
- **Swimlanes.** The board can group tickets into horizontal swimlanes by category — a Discord-style grouping inside a project (for example, a sprint or a `Bugs` section). Dragging a ticket into another swimlane changes its category, not its identity. Uncategorized tickets are allowed.
- **Ticket types and labels.** Filter the board by project, category, label, ticket type, and status. Labels are cross-cutting tags (`bug`, `urgent`) rather than a slicing mechanism — that's what categories are for.
- **Project-scoped settings.** Statuses, ticket types, and label colors are configured per project, under that project's own settings — not shared workspace-wide. Adding or removing a status only affects that project's board.

## Attachments, one mechanism everywhere

Docs, tickets, and chat messages share the same attachment mechanism: a file uploads once to `/api/attachments/<id>`, and whatever it's attached to — a doc body, a ticket description, a chat message — references it by that stable URL rather than storing the bytes inline. Deleting the owning doc or ticket cascades to its files.

Chat has its own page with channels, direct messages, document threads, and
voice channels. See [Chat and voice](/docs/guide/chat-and-voice/). A ticket's
thread stays on the ticket page, where a ticket play's trail appears as well.
