# A computer's clone permission is read back from T3, and an empty project covers what has nothing to clone

Cloning a repository onto a computer needs the `source-control:write` scope on the computer's T3 session. A computer
paired before Nexul asked for that scope holds the older set, so its clone failed first and only then said to pair it
again, and the project picker offered a Clone button that could not work.

Decision: Nexul asks T3 what the session may do instead of recording it. T3's `GET /api/auth/session`, called with the
session's bearer token through the computer's runner like every other request to T3, answers `permissions`, the full
list, and `scopes`, the subset with the older names. The list is read when the picker loads the computer's projects and
when a clone starts. The picker says to pair again, with a button, before any click, and a clone without the scope is
refused before T3 is asked to start one.

- **Unknown is not lacking.** A T3 that has no such endpoint, answers no `permissions`, or cannot be asked counts as
  able: T3 itself refuses a clone it will not take, so a guess against a working computer would be the only wrong
  answer. Only `permissions` is read, because `scopes` lists the older names alone and never names the clone scope;
  an empty `permissions` is a grant of nothing.
- **Harness-neutral.** The harness seam gains `CanClone`; the scope's name stays inside the T3 packages.

Rejected: storing the granted scopes on the computer when it pairs. That needs a migration, leaves every computer
paired before it unanswered until it re-pairs, and goes stale when a session is revoked or re-granted inside T3.

A project with no GitHub repository, or a computer with no T3 project at all, had nothing to clone, and setup refused
a computer without a T3 project ("open any project in T3 Code first"), which a computer run from the command line
cannot do. Decision: `computer_project_create` and the picker's Create empty project take a folder name instead of a
repository, and T3's `projects.mutate` with `project.create` and `createWorkspaceRootIfMissing` makes the folder and
registers it. The folder follows the clone rule: the folder most of the computer's T3 projects share, else `~/Code`,
so `~` is expanded by T3 against the computer's own home. A folder that already holds a T3 project is refused as a
conflict, since T3 answers only "Failed to mutate project." for it. Setup's refusal names both ways out.

- **No new scope.** Creating a project needs only `orchestration:operate`, which every pairing holds, so a computer
  paired before cloning existed can use it.
- **No new tool.** The existing tool takes `repository` or `name`, exactly one of them, keeping the tool budget.

Rejected: T3's `projects.createNew`, which puts the folder under T3's own `~/.t3/projects`, away from the computer's
other projects.

Decided 2026-10-21.
