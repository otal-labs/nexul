---
title: Importing
description: Bring another tool's records into a project as tickets or docs, with their comments, labels, assignees and files.
sidebar:
  order: 10
---

An import copies another tool's records into one [project](/docs/guide/projects-and-repositories/) as tickets or docs. It reads first and writes only once you confirm, so you see how many records there are and where people and statuses will land before anything changes.

It reads three sources today, and more will follow on the same page:

- **GitHub issues**: one repository attached to the project, with each issue's comments, labels and first assignee.
- **Linear**: one team, with each issue's comments, labels, priority, project, due date, assignee, files and blocking links.
- **Notion pages**, into docs.

Only the workspace's Owner can run an import.

## Run one

1. Open **Configuration → Import**.
2. Pick the source and the project. For GitHub, a project with several repositories asks which one, and a project with none needs one [attached](/docs/guide/projects-and-repositories/) first. For Linear, paste a personal API key and name the team by its key, such as `ENG`; leave the team empty when the key reaches only one.
3. Press **Scan**. Nexul counts the issues and comments and works out how they map. The page updates by itself.
4. Check the summary, then press **Import**. Progress shows as it goes, and **Cancel import** stops it at any point.

A scan waits 24 hours to be confirmed, then cancels itself. A workspace runs one import at a time.

GitHub issues are read as Nexul's [GitHub App](/docs/guide/github-app/), so this import needs no token.

A Linear import reads with the personal API key you paste in, which you create in Linear's security settings. Nexul keeps it sealed while the import runs and deletes it when the import ends; it is never shown again or sent anywhere but Linear.

A repository whose [issue sync](/docs/guide/projects-and-repositories/#sync-github-issues-to-tickets) is on can't be imported: its issues are already tickets, and turning sync on brings the rest in. Import is for a one-time move; sync keeps the two in step.

## How records map

### From GitHub

- **Tickets.** Each issue becomes a ticket with its title and body, and a line at the end linking back to the issue and naming who opened it.
- **Statuses.** An open issue lands in the project's first backlog column and a closed one in its first done column. A status the project has no column for gets a new column, which the summary marks as new.
- **People.** The first assignee becomes the ticket's developer when they are someone who can open the project, matched by the GitHub account linked to their sign-in, or by email for tools that give one. Anyone unmatched stays unassigned, and the ticket's body says "Assigned in GitHub to" them. The summary lists everyone and who they match.
- **Comments.** Each comment lands in the ticket's thread under its author's name on GitHub, with when they wrote it. Nobody can edit it, and it mentions nobody.
- **Labels.** Each label carries over.
- **Files.** Images and files people dropped into issues and comments are copied into the ticket, up to 10 MB each. A larger file, or one GitHub won't serve without signing in, stays a link.

### From Linear

- **Tickets.** Each issue becomes a ticket with its title and description, and a line at the end linking back to the issue. A sub-issue comes in as its own ticket, with a "Sub-issue of" line naming its parent.
- **Statuses.** Each workflow state lands in the column of the same name. A state the project has no column for gets a new column of the matching kind: triage, backlog and unstarted states are backlog, started is in progress, and completed and canceled are done. A canceled issue also gets the label `canceled`.
- **Labels and priority.** Each label carries over, and the priority becomes a label: `priority:urgent`, `priority:high`, `priority:medium` or `priority:low`.
- **Projects.** A Linear project becomes the [category](/docs/guide/docs-tickets-and-board/#the-board) of the same name, added the first time the project lacks it.
- **Due dates and people.** The due date carries over, and the assignee becomes the developer when their email matches someone who can open the project.
- **Comments and files.** Comments land in the thread under their author's name on Linear, and uploaded files are copied in as they are from GitHub.
- **Blocking.** When one issue blocks another, the ticket waits on its blocker the way a [blocked-by link](/docs/guide/docs-tickets-and-board/#tickets) does. Other relations, such as related or duplicate, are counted in the summary and skipped.

If Linear asks Nexul to slow down, the import waits and carries on by itself, from where it stopped.

Importing tells nobody: no notification goes out and no [auto play](/docs/guide/plays/) starts for an imported ticket. Once it is in, it is an ordinary ticket.

## From Notion

A Notion import reads the pages you shared with an internal integration and writes them as [docs](/docs/guide/docs-tickets-and-board/).

1. In Notion, create an internal integration with read content access and copy its secret.
2. Open each page to import, and add the integration under **Connections**. Its child pages come with it.
3. In **Configuration → Import**, pick **Notion pages** and the project, paste the secret, and press **Scan**. To import only some pages, paste their links under **Root pages**, one per line; left empty, every top page shared with the integration comes in.

- **Folders.** Each root page becomes a doc folder named after it. A folder of that name the project already has is reused.
- **Pages.** Every page under the root becomes a doc in that folder, the root included. A parent's doc links each of its child pages where they sat.
- **Links.** A link or mention of another page in the import points at its doc. A page outside the import keeps its Notion link.
- **Databases.** Each row of a database becomes a doc that opens with a table of its properties, and the page holding the database lists its rows. A linked view of a database is skipped.
- **Files.** Images and files uploaded to Notion are copied into the doc while the import runs, since Notion's links to them expire within the hour. Files linked from elsewhere stay links.
- **Blocks with no match.** Embeds, synced blocks, columns and equations come in as their text or a link, and the import counts how many.

One import reads up to 5,000 pages, down to 24 levels below its root page. The secret is deleted when the import ends. If an import fails or you cancel it, a doc it had started but not finished is archived, and the next import writes it again.

## Running it again

An import skips everything an earlier import brought into the workspace from the same source, so running it again only adds what is new. A ticket or doc you edited since is never overwritten. If the server restarts during an import, it carries on where it stopped without creating anything twice.

Records that could not come in are listed with the reason, up to 200, and the count goes on past that. Earlier imports stay listed under the form with what each brought in.

## From an agent

An agent runs the same import through one [MCP tool](/docs/guide/mcp-server/), `import_run`: with `source` and `project_id` it starts a scan, with the import's `id` it returns progress, and `confirm` or `cancel` act on it.
