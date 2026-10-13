# 02: Every one-way door

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

What can be created or changed but never removed, undone or edited again, from the web, the phone, HTTP or MCP?
Start from the four the walks found: a machine (no delete route), a Cloudflare tunnel (delete exists on the
server and MCP, not on the web), a stack's environment and secrets (editable only in the creation wizard), and a
failed deploy (no retry). Then compare every create against its delete or undo across every domain, and list:
- copy that promises a way back that doesn't exist ("Add it again to bring it back", "their computers are
  disconnected")
- states that can be entered but not left

For each finding, note what the server already supports, so the fix may be web-only.

## Answer

The full table is in `research/02-one-way-doors.md`, read at `0e023490`.

There are 15 doors, and the four known ones are confirmed.

**Web-only**, because the server and MCP already have the way out:
- Ticket delete: `tickets:delete` has no button.
- DNS records the web creates.
- Archived docs: restorable, but invisible.

**Mostly web-only**, each with a server caveat:
- Tunnel delete. The server checks neither gateways nor computers that use the tunnel, and it leaves the CNAME
  behind.
- Stack environment. `PATCH` already supports it, but `GET` returns values unmasked, so who may read them is open.
- Deploy retry. A deploy doesn't store the ref it built.

**Client-only**: on the phone, message edit and delete, and deploy cancel.

**Copy only**: the account reactivate confirm.

**Server first**:
- Machine delete.
- Unlinking a ticket's branch and PR.
- Workspace delete.
- Revoking an enrollment code.
- Marking a notification unread.

**Probably by design**, which ticket 06 rules on: threads can't be deleted (ADR 0094), and an instance can't
downgrade.

**Copy that promises a way back that isn't there**:
- The automations host's "Add it again to bring it back".
- Reactivate, which doesn't mention computers.
- The stack page's "Attach a repository … from here".
- "Change it later from the DNS page".

Rollback is enabled for compose and build deploys that have no image, and returns 409.
