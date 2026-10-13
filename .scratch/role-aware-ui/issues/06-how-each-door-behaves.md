# 06: How each one-way door behaves

Type: grilling
Status: resolved
Blocked by: 02

## Question

For each door ticket 02 finds, decide the way out and its edge cases. For example:
- Deleting a machine that still has runners or stacks targeting it.
- Deleting a tunnel that gateways or hostnames still use.
- Whether saving a stack's environment redeploys it or waits for the next deploy, and who may read a secret's value
  again.
- Whether retrying a failed deploy rebuilds the same commit or the branch's latest.
- What each confirm says.

## Answer

Owner's calls, 2026-10-13. The doors come from research 02.

**Fixed in this effort**
- Ticket delete
- Cloudflare tunnel delete
- DNS records the web creates
- A way to see and restore archived docs
- Editing a stack's environment
- Deploy retry
- Machine delete
- Unlinking a ticket's branch or PR
- Revoking an enrollment code
- Marking a notification unread
- On the phone: message edit and delete, and deploy cancel
- Workspace archive
- The copy that promises a way back

**Left as they are**: threads can't be deleted (ADR 0094), and an instance can't downgrade.

**How each one behaves**
- **Machine**:
  - Deleting it is refused while any stack deploys to it.
  - The refusal names those stacks and says to move or remove them first.
  - This is a deliberate gate, not a signal, because a deploy aimed at a deleted machine would fail.
- **Tunnel**:
  - Deleting it is refused while any hostname uses it.
  - The refusal names those hostnames and says to edit or remove them first.
  - Once it is free, the delete also removes its DNS record. The server checks this; today it doesn't.
- **Stack environment**:
  - An Environment section on the stack page.
  - Saving applies on the next deploy and offers Deploy now.
  - Secrets are masked once saved. People who can write the stack can reveal them; people who can only read it see
    the keys only. The server stops returning values to readers.
- **Deploy retry**:
  - Retry on a failed deploy rebuilds the same commit with the stack's current settings, so fixing a missing
    environment variable and retrying works.
  - The deploy records the commit it built, which is a server field.
  - Rollback is hidden where there is no image to roll back to.
- **Workspace**: archive hides it from everyone, and the instance owner restores it. A permanent delete comes later.
