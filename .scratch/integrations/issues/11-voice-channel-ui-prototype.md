# 11 — Voice channel UI prototype

**Type:** prototype
**Status:** open
**Blocked by:** 09

## Question

The Discord-style surface: voice channels in the channel list, join/leave,
who's in the channel (from outside), the in-call view with participant
tiles, screen share, camera, mute/deafen, and how this coexists with
CH4's floating thread dock and the dedicated chat page.

Raise fidelity with a cheap concrete artifact to react to — run
design-mode (top-10 references → the owner picks → save the pick as the locked reference) for
the look, then a throwaway prototype of the in-call layout and channel-list
occupancy. The lock is placement/structure in Mono Console tokens, per the
chat surfaces precedent (CH4).

Resolves: the locked reference + layout decisions the implementation
tickets will build against.

## Comments

Owner delegated execution (2026-08-27): building v1 directly instead of a
throwaway prototype — the owner named Discord as the structural reference at
charting, and the chat surfaces' look is already locked (CH4, Mono
Console tokens), so the prototype's question is answered by the real
thing. Ticket stays **open** as the place for the owner's reaction to the
built UI; a design-mode polish round happens here if wanted.

First live reaction (the owner, 2026-08-28): "liking it so far", two fixes
requested and shipped (`e8047081`) — console noise (livekit-client info
logging, now capped at warn) and "I don't see myself when I join"
(Discord reference screenshot): audio-only participants had no tile and
self-presence waited on the 30s poll. Now: optimistic self in the
occupant list on connect, occupants listed under the channel row with
GitHub avatars (identity→login via the members lookup), and avatar
tiles with a speaking ring for camera-off participants in the call view.

Second reaction round (the owner, 2026-08-28, Discord screenshots as
reference; shipped `a4aadf1c`): joining no longer spawns the chat panel
(text stays behind the message icon); a Discord-style **VoiceDock**
strip above the sidebar account area now owns the call app-level
(connection survives panels/pages; mic/camera/screen-share/leave
controls; connecting/error/setup states); presence became server-side —
`Join` registers the occupant at token mint and a new leave endpoint
removes it, so new tabs and other users see who's in the call instantly
instead of waiting on the 30s poll; occupant-list indent now matches
each surface's row geometry.
