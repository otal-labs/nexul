# 24: A stack's deploy actions ignore Deploy to on its machine

Type: bug
Status: open
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Machine, Locked action, Source), ADR 0151, ADR 0152, `practices/react-guide.md`
(F1 to F7 and the self-review), `practices/go.md`, the spec, `client-core/actionState.ts`.

## Problem

Starting a deploy takes `deploys:write` on the stack and Deploy to on the stack's machine, checked on every deploy a
person starts (ADR 0151). The stack page asks only the first: it reads the stack's answers from one explain request,
and explain says nothing about the machine.

Found on 2026-10-14 walking a real install as DevOps (role holds `deploys:write` and `stacks:write` in the project, but
no rule puts the role on the stack's machine):
- Redeploy, Rollback, Build & deploy, Deploy now and a failed deploy's Retry all show live.
- Pressing Redeploy toasts `deploy Northwind web: not found: machine "nexul-box-ra23"`, the refusal after the click
  that this effort exists to remove. The message also names no way out.
- `GET /api/permissions/explain?resource_type=stack&resource_id=<stack>` as that person lists `deploys:write` as
  allowed.
- With a rule giving the role See on the machine the same click works, so the machine is the only thing missing.

## What to build

Decide where the machine joins the answer, then build it:
- Server: the stack's `deploys:write` answer in explain folds in Deploy to on its machine (a machine the viewer can't
  see is not found, so it reads as refused with the machine's name, which the stack page already shows as plain text).
- Or client: the five deploy controls read the machine's answer from the machine list through `machineAnswer`, the way
  the Runners page does, and treat an unseen machine as refused.

Server is the smaller change for the web and the phone together; it must stay inside the statement guard
(`TestStatements_ExplainCostsTheSameForAnyNumberOfActions`).

## Acceptance criteria

- [ ] A person without Deploy to on the stack's machine sees Redeploy, Rollback, Build & deploy, Deploy now and Retry
      hidden or locked with a reason that names the machine.
- [ ] The same person gets the reason from explain, so the permissions panel agrees with the buttons.
- [ ] A person with Deploy to sees them live and the deploy starts.
- [ ] The phone's Redeploy and Cancel deploy follow the same answer.
