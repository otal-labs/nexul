# 23: The guide, and a walkthrough on a real install

Type: task
Status: resolved
Blocked by: 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), the spec's guide list, the spec.

## What to build

- Finish the guide pages and add the roadmap entry.
- Walk it on `nexul-box` as a Client, a Junior Developer with the switch on, and DevOps.
- Close every door once.

## Acceptance criteria

- [x] The findings are fixed or ticketed.

## Answer

Walked 2026-10-14 on a fresh native install of master (a throwaway box, torn down after) at 768, 1024 and 1440px, as
carol (Client, switch off), jamie (Junior Developer, switch on) and devon (DevOps), on a project with tickets, docs and
stacks that deploy a public image. Accounts were made in the database, since an invitation needs a sign-in provider;
the roles, project and machine rules went through the app.

**Guide and roadmap.** The pages were already mostly written by the door tickets. `people-and-access.md` gains what a
locked action looks like, the two reasons it gives, and the toast a refused list action shows; `phone-app.md` gains the
locked Redeploy and Cancel deploy. The roadmap item moves to shipped in `ROADMAP.md` and `roadmap.astro`.

**Passed.**
- Client: no locked control and no Delete, Reorder, Run or Redeploy on the board, ticket and docs at all three widths; a
  stack is not found; the Runners page offers nothing to add.
- Junior Developer: locked actions with their reasons on the board (column grips, New category), the ticket menu's
  Delete, a doc's Lock, Clone and Delete, the stack's Redeploy, Rollback and Delete stack, a failed deploy's Retry and
  Add runner. Pressing each does nothing: no dialog, no toast, no new deploy, nothing deleted. The stack's Environment
  lists the names with dots. Switching the role's switch off hides the locked controls on an open page at once, and on
  shows them again.
- DevOps: Redeploy, Rollback, Deploy now, Retry and Environment live; Reorder, New play and Add runner locked.
- Doors, each worked once: unlink a branch; delete a ticket; archive and restore a doc; archive a workspace and restore
  it from instance settings; retry a failed deploy (a new deploy of the same image, shown at once); edit a stack's
  environment (reveal, add, change, Save, Deploy now); revoke an unused runner code; delete a machine, refused first
  with `still used by stack Northwind api, connected runner build-1`, then with the runner alone once the stack went,
  then deleted once the runner was removed.

**Fixed.** A stack's own answers named the role in the reason ("Not allowed: role Junior Developer doesn't grant it")
where the rest of the app says "Your role can't deploy". `actionState` now keeps the item's sentence only when a rule
on that item refused, with a case in `actionState.test.ts` that fails without it.

**Ticketed.**
- 24: a stack's deploy actions show live to someone who lacks Deploy to on its machine, and the click is refused with
  `not found: machine`.
- 25: People on this machine shows a raw user id and "Unknown role" to a Manage holder who can't read Team.
- 26: clumsy default reasons (the column grips) and a Runners link for someone who sees no machine.

**Untested.** Deleting a Cloudflare tunnel or a DNS record: the box has no Cloudflare token, so the DNS page lists no
tunnels or records to delete. Mark unread (no notifications to mark) and the phone controls, which need a device.
