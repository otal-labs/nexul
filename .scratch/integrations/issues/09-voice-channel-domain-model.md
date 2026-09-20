# 09 — Voice channel domain model

**Type:** grilling
**Status:** resolved
**Blocked by:** 08

## Question

Chat has four conversation kinds (CH1: channels, DMs, channel threads,
ticket threads). Where do voice channels fit?

- A fifth **conversation kind**, or a `voice` flag on channels — and does
  a voice channel have its own text chat alongside (Discord pairs them)?
- **Room mapping:** one LiveKit room per voice channel, created lazily on
  first join or eagerly? Who mints tokens (server-side use-case per CH-*
  patterns) and what grants per member (v1 enforces no permissions, per
  CH1)?
- **Presence:** occupancy shown in the channel list without joining —
  driven by LiveKit webhooks (per 08's findings) bridged onto the existing
  live-events socket (CH3's "no second transport" rule)?
- **Screen share + camera:** any domain-level modeling needed, or purely
  client concerns?
- **Boundaries:** chat owns voice channels; what exactly does the LiveKit
  connector own (credentials only, per ticket 10)? Does `@Agent` have any
  presence in voice v1 (recommend: no — out of scope line)?
- Sharpen terms into `CONTEXT.md` (voice channel, room, participant).

Close by extending `chatDomains.md` (or a new `voiceDomains.md` — decide
which) and updating its future-direction line.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review. Full spec:
`docs/voiceDomains.md` (written as part of this resolution).

- **Fifth conversation kind, `voice_channel`** — a real conversation, so
  it carries its own text messages for free (Discord's text-in-voice).
- **New package `internal/voice`** owns everything LiveKit: token minting,
  presence, webhook receiver. Chat owns the conversation; voice takes a
  small conversation-checker seam, never imports chat wholesale.
- **One room per voice channel**, room name = conversation ID, created
  implicitly on first join (token carries the room-create grant). Tokens
  minted server-side: identity = user ID, grants roomJoin + canSubscribe +
  canPublishSources (mic, camera, screen share, screen-share audio) —
  per ticket 08, screen share is a track source, not a separate grant.
  v1 enforces no permissions beyond membership, matching CH1.
- **Presence** (who's in, from outside): LiveKit webhooks
  (`participant_joined/left`, `room_started/finished`) into a public
  receiver, **plus** ListRooms polling reconciliation while any client
  is on a chat surface — webhooks can't reach localhost, so polling is
  the dev path and the drift-corrector (08 flagged best-effort delivery).
  Occupancy publishes over the existing live-events socket (CH3: no
  second transport).
- **Dependency stance:** the Go side avoids the full server SDK if a
  minimal client (JWT HS256 token mint + ListRooms + webhook verify) is
  a few hundred stdlib lines — this repo is dependency-averse; verify
  claims format against `research/08-livekit-facts.md`. The frontend
  necessarily takes `livekit-client` + `@livekit/components-react`.
- **@Agent has no voice presence in v1** (out of scope on the map).
- `CONTEXT.md` terms: *voice channel*, *occupancy*.
