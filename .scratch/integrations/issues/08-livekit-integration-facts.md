# 08 — LiveKit integration facts

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Ground the voice-channel track in current LiveKit facts (docs are past
training-cutoff territory — verify against official sources, current
versions pinned):

1. **Server side (Go):** the server SDK for token minting — JWT access
   tokens, the grants model (room join, publish, subscribe, screen share —
   is screen share a separate grant or just another track?), room
   create/delete/list APIs, current package + version.
2. **Client side (React):** the components library (`@livekit/components-react`?)
   — what it gives for free (audio rendering, participant tiles, screen
   share view, camera), bundle weight, current version.
3. **Presence without joining:** how to know who's in a room from the
   outside — LiveKit webhooks (participant joined/left, room
   started/ended): delivery model, auth, and whether they work identically
   on Cloud and self-hosted. This feeds the Discord-style "see who's in
   the channel from the sidebar" requirement.
4. **Config surface:** confirm URL + API key + secret is the complete
   credential set for both Cloud and self-hosted; anything else an owner
   must provide (region, TURN config?). Self-hosted network prerequisites
   (UDP port ranges, TURN) — enough to document, not to solve.
5. **Licensing:** confirm the server's license still permits the
   bring-your-own model.

Output: findings doc with versions, package names, and answers per point,
with source links.

## Answer

Verified live against LiveKit's official docs/GitHub/pkg.go.dev/npm (current
as of 2026-08-27). Go server SDK is `github.com/livekit/server-sdk-go/v2`
(v2.18.1), token minting via `auth.AccessToken` + `auth.VideoGrant`, room
CRUD via `RoomServiceClient`. Screen share is **not** a separate grant — it's
a `TrackSource` value (`SCREEN_SHARE`/`SCREEN_SHARE_AUDIO`) under the
fine-grained `canPublishSources` allow-list. React client is
`@livekit/components-react` (2.9.24, peer on `livekit-client` ^2.20.1) —
gives `ParticipantTile`, `GridLayout`, audio-track auto-rendering, and a
`VideoConference` prefab; no published bundle-size figure, worth measuring
once installed. Webhooks (`room_started`/`room_finished`/
`participant_joined`/`participant_left`) POST a JWT-signed payload, verified
via the SDK's `WebhookReceiver`, with identical behavior on Cloud
(dashboard-configured) and self-hosted (YAML-configured) — best-effort
retry/ordering, not exactly-once, so periodic `ListRooms` reconciliation is
worth pairing with it. Config surface for bring-your-own is confirmed as
just URL + API key + secret; self-hosted additionally needs UDP
50000-60000 (or a mux port) open and optionally TURN, but that's
documentation, not something the app configures. LiveKit's server and all
SDKs are Apache 2.0 — fully compatible with bring-your-own commercial use,
no further licensing concern.

Full findings: `.scratch/integrations/research/08-livekit-facts.md`.
