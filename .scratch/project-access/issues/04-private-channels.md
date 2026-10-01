# 04 — Private channels

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

How do Private channels behave?

- Who makes a channel private: at creation only, or switchable either way,
  and what happens to the people who lose it when a public channel turns
  private.
- Who adds and removes members, and whether a member may leave.
- Whether `#general` can ever be private (and so whether a Restricted member
  has any ordinary channel at all).
- Voice channels: private too, and what a non-member sees of a call.
- What a non-member sees of a private channel: nothing, the way a hidden
  project reads as not found.
- DMs with a Restricted member: may anyone in the workspace start one, and
  may the Restricted member start one with anyone?

## Answer

Settled with the owner, 2026-10-01.

- **Switchable both ways.** Making a channel private opens a picker: the
  person switching is in it and picks who else stays. Making it public
  again shows its history to the whole workspace, and the confirmation says
  so.
- **Membership.** Anyone in a private channel adds people; removing someone
  else takes `channels:write`; anyone may leave, except the last member.
- **`#general` is always public**, so a Restricted member's channel list is
  exactly the private channels they have been added to.
- **Voice channels follow the same rules.** A non-member does not see a
  private voice channel, cannot join it, and cannot tell a call is on.
- **Non-members see nothing**: a private channel they are not in reads as
  not found, in lists, search, links, and live frames.
- **The Owner sees every private channel**, messages included, through the
  same bypass as every other permission (ADR 0042), so no private channel
  can be orphaned beyond reach.
- **DMs are open both ways**: anyone in the workspace may DM a Restricted
  member, and they may DM anyone, since they see everyone's name already.
