# Channels may be private

Every member read every channel (ADR 0087, ADR 0094), so a team had nowhere to talk that a client in the same
workspace could not read, and a Restricted member (ADR 0097) would have had no channel at all or every one.

Decision: a text or voice channel may be private. Only its members see it and read it; to anyone else it reads as not
found in lists, unread counts, links, messages, attachments, voice join and occupancy, and live frames, so a
non-member cannot tell a call is on.

- **Switchable both ways** with `channels:write`. Going private keeps the person switching and whoever they pick;
  everyone else loses it at once. Going public drops the member list and opens the whole history to the workspace,
  and the confirmation says so.
- **Membership.** Anyone in a private channel adds people; removing someone else takes `channels:write`; anyone may
  leave except the last member. Members are the conversation's participant rows, the table DMs use.
- **`#general` is always public**, so a Restricted member's channels are exactly the private ones they are in.
- **The Owner sees every private channel**, messages included, through ADR 0042's bypass, so none can be orphaned
  beyond reach.
- **DMs are open both ways**, Restricted members included, since everyone already sees everyone's name (ADR 0086).
- Membership changes publish `chat.conversation.members_changed`, reaching the channel's readers and the people it
  removed; a private channel's deletion reaches only its members and the Owner.

The trade-offs: the Owner can read every private channel, which is the price of no channel ever being lost; a
private channel turned public exposes its history, warned about rather than prevented. Each read of a channel now
checks membership for private ones, one more lookup per channel.

Rejected: private at creation only, which makes a mistaken public channel permanent; and hiding private channels from
the Owner, which leaves one nobody can manage once its members are gone.

Amends ADR 0094, whose "reading a channel takes membership alone" now holds for public channels only, and ADR 0087's
list of what every member reads. Decided 2026-10-01.

Amended 2026-10-01: a private channel's and a DM's message events carry `members_only` and reach no integration or
automation, the way they reach no non-member.
