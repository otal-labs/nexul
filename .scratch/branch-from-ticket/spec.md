# Create a branch from a ticket

**Status:** needs-triage

> Carried over from the git provider domain specification when it was
> dissolved. An old wish, not a queued build — re-confirm it is still wanted
> before slicing it.

## Problem Statement

A ticket's development panel can link a branch that already exists, but it
cannot make one. Starting work on a ticket therefore means leaving
Nexul, switching to a terminal or the provider's UI, inventing a branch
name by hand, and then coming back to paste it in — and a hand-typed name
often misses the ticket key, which is the only thing that lets a later push or
pull request find its way back to the ticket on its own.

## Solution

From the ticket page, pick a repository and a base branch and get a branch
created on the provider, named after the ticket and already linked back to it.
The name is proposed from the ticket's display key and title, so the link
survives even for people who then work entirely from the command line.

## User Stories

1. As a developer, I want to create a branch from a ticket, so that I can
   start work without leaving the ticket.
2. As a developer, I want the branch name proposed from the ticket's key and
   title, so that later pushes and pull requests resolve back to this ticket
   without my doing anything.
3. As a developer, I want to edit the proposed name before creating it, so
   that I can follow a team convention the default does not match.
4. As a developer, I want to choose which of the project's repositories the
   branch is created in, so that a ticket spanning a frontend and a backend
   repository can have a branch in each.
5. As a developer, I want to choose the base branch, so that a fix off a
   release branch does not start from the default branch.
6. As a developer, I want the base branch list to default to the repository's
   default branch, so that the common case is one click.
7. As a developer, I want the created branch to appear in the development
   panel straight away, so that I can copy its name or open it on the
   provider.
8. As a developer, I want a name that already exists on the provider to fail
   with a clear message naming the collision, so that I neither silently reuse
   someone else's branch nor wonder what went wrong.
9. As a developer, I want creating a branch in a repository whose git host is
   not connected to fail for that repository only, so that the rest of the
   panel keeps working.
10. As an owner, I want branch creation to be the only write the panel
    performs, so that deeper git operations stay where developers expect them.
11. As an agent, I want the same action over MCP, so that "start work on this
    ticket" can create its branch.

## Implementation Decisions

- The git provider seam gains the read and write operations this needs:
  list a repository's branches, read its default branch, and create a branch
  at a given base. Everything else about the provider abstraction is unchanged,
  and the GitHub adapter is the only implementation at first.
- The action goes through the existing per-repository provider resolution, so
  a repository on a different git host uses that host's client and fails in
  isolation when its connector is disconnected.
- A created branch is recorded against the ticket through the existing
  ticket↔branch link, so nothing new is needed to display it and an
  externally-created branch with the same name later links idempotently rather
  than duplicating.
- The proposed name is built from the ticket's project prefix and number plus
  a slug of the title — the same key shape the codebase already parses out of
  branch names and commit messages to resolve a ticket. The prefix is
  immutable precisely so a name written into git history keeps resolving.
- The repository choices come from the ticket's project. A project with one
  repository preselects it.
- Creation is not permitted on a repository the project does not own.

## Testing Decisions

Test through the ticket's use-case layer, not through the provider client: a
good test says "creating a branch for this ticket produces a branch on the
fake provider and a link on the ticket", and never asserts the shape of the
provider's HTTP call. The existing fake git provider used by the webhook and
pull-request tests is the prior art and should grow the new operations. Cover:
the happy path leaves both a provider branch and a ticket link; a name that
already exists surfaces the provider's conflict unchanged; a repository
outside the project is refused; an unconnected git host fails for that
repository alone; the proposed name resolves back to the ticket when parsed by
the existing branch-name resolver.

## Out of Scope

- Commits, pushes, merges, or any other git operation from Nexul —
  the panel stays read-only navigation plus this one action.
- Deleting branches from the ticket page.
- Build and check status in the development panel. Provider check events
  already reach the event bus but nothing consumes them; surfacing them is
  separate work.
- Creating a pull request from the ticket.

## Further Notes

The development panel itself and the ticket↔branch link records belong to the
tickets domain; only the provider operations and the resolution path are the
git provider's. Whoever picks this up should confirm the split with the
tickets work before starting.
