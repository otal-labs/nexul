# 26: Two small leftovers from the walkthrough

Type: task
Status: resolved
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action), ADR 0152, `practices/react-guide.md`.

## What was seen

Found on 2026-10-14 walking a real install as Client, Junior Developer and DevOps.

1. **Clumsy default reasons.** A reason that falls back to the permission's catalog label reads badly where the label
   is long. The board's column grips and New category say `Your role can't create and update projects and board
   settings`; Deploy actions without a verb say `Your role can't create and update deploys`. Pass a `verb` where the
   label is a mouthful ("reorder columns", "add a category") and read every locked reason once for the rest.
2. **Runners in a Client's sidebar.** A Client holds no machine, so the Runners page shows only "No runners on the
   machines you can see yet." The nav item leads to an empty page. Hide it for someone who sees no machine and holds
   no way to add one, and keep it for anyone who can add a runner.

## Acceptance criteria

- [x] No locked reason on the board, ticket, doc, stack, deploy and Runners pages reads as a permission label.
- [x] A person who sees no machine and can't add one has no Runners link.

## Answer

- Locked reasons: the catalog labels stay as they are, because the roles editor shows them and "Create and update
  tickets" is right there. Each control gated by a write permission (or `projects:delete`) passes the `verb` it
  performs instead: board Create menu, column grips, categories, statuses, ticket types, repositories, the ticket's
  property rows, pass and fail, doc lock and unlock, branch deploy rules and the wizard steps. `practices/react-guide.md`
  says a write-gated control passes a verb. Deploy, runner and machine controls already had one.
- Runners link: shown when the viewer sees a machine (`GET /api/machines`, the query the Runners page reads) or can add
  one (`machines:create` in the auth `/me`'s `instance_permissions`, or the instance Owner). The sidebar, the command
  palette and the phone's More tab follow it; the phone checks `GET /api/runners`, since it has no machine list. The
  route still opens for every member.
