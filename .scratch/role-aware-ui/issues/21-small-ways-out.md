# 21: Unlink a ticket's branch or PR, revoke an enrollment code, mark a notification unread

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- Each needs a server change, an MCP tool change inside the budget, and a web control, as in the spec's table.
- Guide pages.

## Acceptance criteria

- [x] Each way out is tested on the server and the web.

## Answer

Built on 2026-10-14; the phone is left for a device pass.

- **Unlink a branch or PR.** `DELETE /api/tickets/{id}/branches` and `/prs` take the link in the query and answer with the ticket's links as they stand. They need `tickets:write`, as linking does. An unlinked link that isn't there is a no-op. `ticket_update` gains `unlink_pr` and `unlink_branch`. The ticket page shows a hover × on each linked branch and PR to anyone with `tickets:write`. A PR that still names the ticket in its description or branch is linked again by its next PR event, and unlinking does not re-check whether the ticket should now finish.
- **Revoke an enrollment code.** Runner codes get an opaque id (migration 0112), so the hash never leaves the server. `GET /api/runners/enrollments` lists the unused codes the caller may revoke and `DELETE /api/runners/enrollments/{id}` revokes one. The rule is the maker, someone with Manage on the code's machine, or the instance Owner; a computer's code is its owner's alone, and the server's own code is nobody's. No new tool: `machine_list` shows `open_enrollments` and `host_delete` takes `code_id`. The Runners page lists the runner codes and Computers lists the computer ones, each with Revoke. Automations host codes are not covered.
- **Mark a notification unread.** `POST /api/notifications/{id}/unread`, and `notification_update` takes `unread: true` with an id. It clears the read time, so the retention sweep spares it. A read inbox row's right-click menu has Mark unread; it puts a doc row's newest notification back.
- None of the three publishes an event, as linking, reading and minting a code publish none. Open pages refresh from the mutation, and the codes list also follows `runner.connected` and `runner.personal_changed`.
- **Phone.** The inbox shows read state and swipes to mark read, so Mark unread belongs there too, with a hook and a swipe or accessibility action on read rows. The phone has no development section and no way to add or revoke a runner, so those two stay web-only.
