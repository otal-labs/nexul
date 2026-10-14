# A machine or a tunnel is refused while something uses it, and the refusal names what

Deleting a machine or a Cloudflare tunnel takes away what other things stand on. A stack aimed at a deleted machine can
no longer deploy, because its machine is a name nothing answers to; a gateway or hostname routed through a deleted
tunnel stops serving. Most actions here warn and let the person go on, but these two cannot be put right by going on:
the deploy fails, the hostname goes dark, and the person finds out later, away from the thing they deleted.

Decision: both deletes are refused while anything uses the thing, and the refusal names each user and says what to do
first. Nothing is removed on the person's behalf.

- **One error.** `apperrs.InUseError` carries the thing, the users and the fix. It matches `ErrConflict`, so HTTP
  answers 409 and MCP passes the same text on; the web shows it as a toast.
- **A tunnel** is used by the instance's own address, a gateway running through it, a hostname exposed on one of those
  gateways, and its cloudflared stack.
- **A machine** is used by every stack aimed at it, branch deployments included, every gateway on it, and every runner
  on it that is connected. Deleting it takes Delete on the machine (`machines:delete`, ADR 0151), and a machine the
  caller can't see is not found.
- **What goes with a free machine.** Its offline runners, their credentials, its rules, and the enrollment codes still
  waiting to put a runner on it, in one transaction with the `machine.deleted` event. This follows removing a computer,
  which retires its runner: a runner that comes back is refused as removed and uninstalls itself. An offline machine
  therefore goes in one step.
- **Connected runners are users, offline ones are not.** A connected runner is live work on the host, and deleting the
  machine would have to uninstall it from under a job. Removing it first is a deliberate act, and the refusal says so.
- **The frame is addressed to the people who saw the machine.** A machine's rules go with it, so by the time the frame is
  delivered nobody's access to it can be read. The event carries the ids of those who saw it when it went (the instance
  Owner and everyone a rule reached), and the live audience rule delivers it to them alone, so the card leaves every
  open page that showed it and no one else hears a machine existed.
- **The way back.** Enrolling a runner on the machine's name makes the machine again, with the adder holding every action
  on it and nobody else; the old rules do not return.

The trade-offs: the person has to move or remove each user first, so a machine with many stacks is a longer job than a
cascade would be; the refusal names stacks to anyone holding Delete on the machine, who already holds root on that
host; a stack created in the instant between the check and the delete can still land on a machine that is going, and
shows as a stack aimed at a machine nothing answers to.

Rejected: cascading the delete through stacks, gateways and hostnames, which makes one button remove a team's
deployments and public addresses; a soft delete with a restore, which keeps a name the runners can no longer reach and
makes the machine list lie about what exists; refusing for offline runners too, which would leave a dead machine's card
with no way off the page but removing each runner first; and warning in the confirm only, which does not stop the
deploy that fails afterwards.

Decided 2026-10-14.
