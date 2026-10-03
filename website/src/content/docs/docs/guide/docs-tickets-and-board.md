---
title: Docs, Tickets, and the Board
description: How documents drive work, how tickets track it, and how the board organizes it.
sidebar:
  order: 10
---

Nexul's loop starts with a document, not a ticket. Docs are the source of truth; tickets are scoped, actionable units of work derived from them.

## Docs

A document belongs to exactly one project, the same rule a ticket follows. Each project's sidebar includes **Board**, **Interview**, **Docs**, **Memories**, and **Settings**. **Docs** lists the project's documents you can open, grouped by folder, beside the one that's open. Within a folder, docs sort newest first by when each was created; a toggle in the list header switches to last edited, and your choice is remembered. Pin a doc from its row menu to lift it into a Pinned group at the top, newest pin first; pins are kept in your browser, so they are yours alone and don't follow you to another one. Its search filters the list by title and first line, showing the matches inside their folders.

- **Folders.** Every doc lives in exactly one folder of its project, one level deep. Each project has a default folder, **Main**, which holds every doc from before folders existed and every new doc unless you start it from another folder's **+**. **New folder** in the list header adds one; a folder's **…** menu renames it, or deletes it after saying how many docs move to Main, since deleting a folder never deletes a doc. Main can be renamed but not deleted. Move a doc with **Move to folder** in its row menu, and click a folder's name to collapse it; collapsed folders are remembered in your browser. Creating, renaming, and deleting folders and moving docs need `docs:write`; someone who can only read sees just the folders holding a doc they can open. A doc cloned into another project lands in that project's Main. The API routes are `GET`/`POST /api/docs/folders`, `PUT`/`DELETE /api/docs/folders/{id}`, and `POST /api/docs/{id}/move`; over MCP, `project_get` lists a project's `doc_folders`, `project_update` changes them, and `doc_create`, `doc_update`, and `doc_list` take `folder_id`.
- **Rich editor.** The canonical representation is structured rich text, not raw markdown — you get real formatting and styling. Markdown is a conversion surface: it's what LLMs, imports, exports, and integrations read and write, converted to and from the structured document automatically.
- **`@` mentions.** Typing `@` opens an autocomplete of the workspace's people, relevant tickets, and other documents. Picking one inserts a live reference: a ticket mention renders as a chip laid out by the workspace's mention chip template (its id and status out of the box, the instance's until the workspace sets its own), a document mention shows its current title, and a person shows their picture and display name. Mentioning someone tells them in their inbox. If you can't access what's tagged, the chip stays visible but inert — you see the title, not the content.
- **Watchers.** A doc's edits notify its watchers, not the whole workspace. You become a watcher when you create a doc or save an edit to its title or body, whether in the editor, over MCP, or through a play acting for you. The eye button in the doc's header shows how many people watch it; click it to see who, and to **Watch** or **Stop watching** yourself, which only needs read access. Stopping sticks: your own later edits don't make you a watcher again until you choose to watch. Creating a doc notifies nobody, and a mention always reaches the person mentioned, watching or not. The API routes are `GET /api/docs/{id}/watchers` and `PUT`/`DELETE /api/docs/{id}/watchers/me`; over MCP, `doc_get` lists `watchers` and `doc_update` takes `watch`.
- **Real-time collaboration.** Multiple people can edit the same document at once. Changes appear live, presence shows who's viewing or editing (with cursors and selections), and normal concurrent edits merge automatically. A conflict prompt only appears for edits that genuinely can't be merged. Offline edits sync once you reconnect.
- **Versions.** Lightweight history is kept for recovery, plus deliberate named versions for milestones you want to come back to — not a version for every keystroke.
- **Attachments.** Paste, drop, or use the `/image` slash command to attach files. Everything the doc owns is listed under the body with download and delete; deleting the doc deletes its files too.
- **Lock.** Locking a doc makes it read-only for everyone until someone unlocks it: the title and body can't be changed from the editor, the API, or MCP, and a lock icon marks it in the list. Lock and **Unlock** sit in the doc's row menu and its settings menu, and a locked doc says **Locked** at the top with an **Unlock** button. Either needs `docs:lock`, which a role grants apart from editing; without it Lock and Unlock are hidden. Starting a doc play locks the doc too, whoever starts it, and it stays locked after the run ends. Archiving, cloning, and deleting still work. The API routes are `POST /api/docs/{id}/lock` and `/unlock`, and `doc_update` takes `locked` over MCP.
- **Clone and delete.** Each doc in the list shows only its title; hover it for a menu with **Lock**, **Move to folder**, **Clone**, and **Delete**. **Clone to…** copies the doc and its attachments into any project you can write docs in; choosing its own project makes a copy beside it. Cloning needs `docs:clone`, deleting `docs:delete`, and both are hidden without them. Deleting is permanent, unlike archiving, and is refused while tickets name the doc as their source; clear their source first. The API routes are `POST /api/docs/{id}/clone` and `DELETE /api/docs/{id}`; over MCP, `doc_create` takes `clone_from_id` and `doc_delete` deletes.

## Tickets

Tickets don't have to come from a document — you can create one directly — but a doc can spawn tickets scoped to a specific slice of work.

- **Types.** Owners define ticket types (`bug`, `feature`, `task`, or anything project-specific), each with its own icon, color, and body template. A new project's `task`, `bug`, and `feature` types copy the instance's body templates (see [Templates](/docs/guide/memories/#templates)).
- **Branches and PRs.** From a ticket's Development section you can create a new branch (choosing repository and base branch) or link an existing branch or PR. A branch or PR can link to more than one ticket. Pull requests live here, on the ticket page — there's no separate pull requests page.
- **Bugs.** A bug is a ticket of the type named `bug`, and it always says which ticket it was found in. Use **Report a bug** on the ticket page to file one linked to that ticket, or **Report a bug** in the board's add menu, where you can tick **Origin unknown** instead. The normal New ticket dialog offers every other type. A done ticket is never reopened: a bug found after done is a new ticket, and the done ticket's page lists it under **Bugs found after done**. A type renamed away from `bug` stops being treated as one.
- **Source.** A ticket filed from a doc keeps that doc as its source, shown first under **Links** on the ticket page with its title linking back to the doc. **Source doc…** in the Links `+` menu sets or replaces it from the project's docs, and the row's **×** clears it; a ticket has at most one source, always a doc you can read. The API route is `PATCH /api/tickets/{id}/source`, and `ticket_update` takes `doc_id` over MCP, an empty one clearing it.
- **Blocked by.** A ticket can wait on others. Its card shows a blocked icon until every blocker reaches a done column, but it still moves freely. A play on a blocked ticket asks you to confirm first.
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
