# Posting in a channel takes channels:post

Posting in a conversation took only reading it (ADR 0087), so a channel could be hidden from a role (ADR 0148) but
never made read-only: an announcements channel every member reads and only Leads post in had no way to exist.

Decision: posting in a text or voice channel takes `channels:post`, a verb on the channels domain that roles hold and
a channel's Permission overwrites set for a role or a person, beside `channels:read`. The check reads the channel
first, so a channel someone can't read stays not found, and a refused post answers forbidden with the source sentence.
DMs and doc, ticket and interview threads keep taking only their read.

- **Every path that posts goes through the one check.** A person's message over HTTP or `message_post`, the Agent's
  reply to them and what they typed in their harness post as that person. An integration or automation token posts as
  its creator, and its route now takes the `channels:post` scope where it took `chat:write`. A bot posts while its
  creator may post in its channel, so a bot can't speak where the person who made it can't. The server's own notices
  post as nobody.
- **Channel rules are written with `channels:write`** on a channel the writer reads, under hold-what-you-give, not by
  the Owner alone. The Access page's Public channels grid stays the Owner's and moves only `channels:read`.
- **The upgrade grants it to every role**, with `channels:read` so the role editor's Post verb sits on a rung, because
  every member who read a channel posted in it; integration installs and automations holding `chat:write` gain it,
  because that is the scope their posts took. Nobody's posting changes but a member holding no role, who held no
  bit to keep and now reads public channels without posting in them.
- **Agreed with the owner, 2026-10-13.** A member with no role is read-only in public channels, and their DMs still
  work, so they can message an Owner; their empty state says so. A bot posts only while its creator may post in its
  channel, and when they can't, whether they left, were denied or hold no role, the bot's row, `botwebhook_list`'s
  `stopped` and the sender's 403 say "Stopped: <creator> can't post in #<channel>", so a stopped bot never reads as an
  unknown URL.

The trade-offs: a role an Owner made to read only now shows Post ticked until they untick it; a bot whose creator
left the workspace or lost posting stops posting there, answering 403; a token minted later posts only with the
`channels:post` scope, in a DM or a thread too, since the route can't tell them apart.

Rejected: `channels:post` held by every reader like `channels:read`, with only a rule denying it, which leaves a role
unable to be read-only; granting it only to roles holding `chat:write`, which would have silenced every member whose
role lacked it, since posting never took that bit; and letting a bot post as nobody, which a channel's rules could
never reach.

Amends ADR 0087, whose "posting takes only being able to read it" now holds outside channels, ADR 0094, ADR 0129, and
ADR 0148, whose channel rules set `channels:post` and are written with `channels:write`. Decided 2026-10-13.
