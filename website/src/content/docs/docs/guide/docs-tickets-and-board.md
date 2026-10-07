---
title: Docs, tickets, and the board
description: Write the plan as a doc, break it into tickets, and move them across the project's board to done.
sidebar:
  order: 10
---

Work in Nexul usually starts with a doc: what you want and why. Tickets are the pieces of work cut from it, and the board shows where each one is. All three belong to a project, under its name in the sidebar: **Board**, **Interview**, **Docs**, **Memories**, and **Settings**.

## Docs

Open **Docs** under the project. The list groups docs by folder, newest first; the toggle in its header sorts by last edited instead. **Pin** a doc from its row menu to keep it at the top. Pins and collapsed folders are kept in your browser, so they're yours alone.

Write in the editor with formatting, tables, and code blocks; type `/` for the block menu. Several people can edit at once: you see who's there and where their cursors are, and edits merge as you type. If an edit can't merge, a banner offers **Keep my version** or **Use server version**.

- **Mention** with `@`: a person, a ticket, or another doc becomes a live chip. Pasting a link to a ticket or doc in the workspace does the same. Mentioning someone tells them in their inbox. A chip you can't open shows its title only.
- **Attach** files by pasting, dropping, or `/` then **Image**. The doc's files are listed under the body.
- **Watch** with the eye button in the doc's header. Watchers hear about edits. You start watching a doc when you create it or save an edit, and **Stop watching** sticks until you choose to watch again.

### Folders

Every doc lives in one folder of its project. New docs land in **Main** unless you start them from another folder's **+**. **New folder** in the list header adds one, and a folder's **…** menu renames or deletes it. Deleting a folder moves its docs to Main; it never deletes a doc. Main can be renamed, not deleted. Move a doc with **Move to folder** in its row menu.

### Lock, clone, and delete

The row menu also has **Lock**, **Clone**, and **Delete**.

- **Lock** makes the doc read-only for everyone, people and agents alike, until someone presses **Unlock**. Locking needs `docs:lock`, granted apart from editing. Starting a doc play locks the doc too.
- **Clone** copies the doc and its files into any project where you can write docs.
- **Delete** is permanent. It's refused while a ticket names the doc as its source; clear that ticket's source first.

### Clarify a doc

When a doc is a rough idea, run **Clarify via AI** on it. The agent reads the doc and asks a round of questions about what it leaves open.

1. Switch the doc to **Questions**. The number beside it counts questions waiting on an answer.
2. Answer, skip, or add your own under **Anything else?**. Anyone who can edit the doc can answer, whenever they like.
3. Press **Clarify via AI** again for the next round. The agent reads the earlier rounds too.
4. When a round finds no gaps, the agent writes the answers into the doc. Press **Close**, then **To tickets via AI** to cut it into work.

The doc keeps its author's words until that last round. It's locked only while a round runs.

## Tickets

Create a ticket with **New ticket** in a board column, or from a doc so the ticket keeps that doc as its source. Pick a type; its body template fills in the sections to write. Set a **Developer** to build it and a **Tester** to check it, both optional.

The ticket page has everything else:

- **Links.** The **+** adds **Blocked by…**, **Blocks…**, **Found in…**, or **Source doc…**. A blocked card shows an icon until every blocker is done, but still moves freely; a play on it asks you to confirm.
- **Development.** Create a branch, or link an existing branch or pull request. A branch or PR can link to more than one ticket.
- **Testing.** In a Testing column it shows where to test and the acceptance criteria, with **Pass** and **Fail**. Fail asks for steps to reproduce, the expected and actual results, and a screenshot. It never points at production.
- **Thread.** The ticket's conversation, where plays post their runs and agents leave notes.

### Bugs

A bug is a ticket of the type named `bug`, and it always says which ticket it was found in. Use **Report a bug** on the ticket page to file one linked to it, or **Report a bug** in the board's add menu, where you can mark the origin unknown. A done ticket is never reopened: a bug found later is a new ticket, listed on the done one under **Found after done**.

### Finished

A ticket is finished once it has at least one merged pull request and none still open. Pull requests closed without merging don't count. The default **Ticket finished** automation then moves it to the status picked in its configuration; see [Automations](/docs/guide/automations/#the-defaults).

## The board

**Board** shows one project's tickets as columns. You design the columns, but each belongs to one of five stages, in order: backlog, progress, review, testing, done. Plays and automations read the stage, never the column's name, so rename freely. Only done is final.

- Drag a card between columns to change its status.
- Group cards into swimlanes by category, such as a sprint. Dragging a card to another swimlane changes its category.
- **Filter** by category, label, type, and status, or **Search**.

Columns, ticket types with their icons and body templates, labels, and categories are set per project, in the project's **Settings → Board** and **Categories**.
