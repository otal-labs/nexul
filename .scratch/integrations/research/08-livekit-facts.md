# 08 — LiveKit integration facts

Research date: 2026-08-27. All facts verified live against LiveKit's official
docs, GitHub repos, pkg.go.dev, and npm — training-data knowledge of LiveKit
is stale and was not relied on for versions or API shapes.

## 1. Server side (Go): token minting, grants model, room APIs

**Package:** `github.com/livekit/server-sdk-go/v2`
**Current version:** v2.18.1 (July 2026), per
[pkg.go.dev/github.com/livekit/server-sdk-go/v2](https://pkg.go.dev/github.com/livekit/server-sdk-go/v2).
Import the module at the `/v2` path — the unversioned `server-sdk-go` path is
the old v1 and shouldn't be used for new code.

**Token minting** goes through `github.com/livekit/protocol/auth` (re-exported
by the server SDK):

```go
at := auth.NewAccessToken(apiKey, apiSecret)
grant := &auth.VideoGrant{
    RoomJoin: true,
    Room:     room,
}
at.SetVideoGrant(grant).SetIdentity(identity).SetValidFor(time.Hour)
token, err := at.ToJWT()
```

**Grants model** — the token carries a `VideoGrant` (Go/JS name; called
`VideoGrants` in the Python SDK) with these fields, confirmed against the
[JS SDK's `VideoGrant` interface](https://docs.livekit.io/reference/server-sdk-js/interfaces/VideoGrant.html)
and the Python `access_token` reference:

- `room` — room name; required for `roomJoin`/`roomAdmin`.
- `roomJoin` — permission to join `room` as a participant.
- `roomCreate`, `roomList`, `roomAdmin`, `roomRecord` — server/admin-level
  room-management permissions (create, list, control a specific room, start
  a recording).
- `canPublish` / `canSubscribe` — coarse publish/subscribe toggles. If
  neither is set, both default to enabled.
- `canPublishData` — data-channel publish, defaults to true if unset.
- `canPublishSources` (`TrackSource[]`) — **fine-grained allow-list of which
  track sources this participant may publish; when set it supersedes
  `canPublish`.**
- `canUpdateOwnMetadata`, `hidden`, `recorder`, `ingressAdmin`,
  `destinationRoom`, `agent`, `canManageAgentSession`,
  `canSubscribeMetrics` — narrower permissions (self-metadata edits,
  invisible/recording bot participants, ingress control, SIP/agent forwarding).

**Screen share is not a separate grant** — it's one of the enum values of
`TrackSource`, alongside `CAMERA`, `MICROPHONE`, `SCREEN_SHARE`, and
`SCREEN_SHARE_AUDIO` (protocol buffer enum, confirmed via
[TrackSource enum docs](https://docs.livekit.io/reference/server-sdk-js/enums/TrackSource.html)).
To let a participant screen-share but not use camera/mic, set
`canPublishSources: [TrackSource.SCREEN_SHARE, TrackSource.SCREEN_SHARE_AUDIO]`
instead of a blanket `canPublish: true`. For a Discord-style channel where
camera + screen share + mic should all be allowed, either leave
`canPublish: true` (unrestricted) or explicitly list all four sources if
per-capability control is wanted later.

**Room APIs** are on `RoomServiceClient` (obtained via the SDK's
`LiveKitAPI` client, e.g. `api.Room()`), taking protobuf request types:
`CreateRoom(ctx, *livekit.CreateRoomRequest)`,
`ListRooms(ctx, *livekit.ListRoomsRequest)`,
`DeleteRoom(ctx, *livekit.DeleteRoomRequest)`. These map directly onto the
"create a voice channel = create a room" and "list active channels" needs.

**Repo:** [github.com/livekit/server-sdk-go](https://github.com/livekit/server-sdk-go)
(protocol types live in the separate `github.com/livekit/protocol` module).

Fits this repo's stdlib-first Go backend fine — it's a single well-maintained
dependency, no transitive sprawl (grpc/protobuf plus the LiveKit protocol
package), and is the officially blessed way to mint tokens; hand-rolling JWT
grants against LiveKit's protobuf claim shape isn't worth it.

## 2. Client side (React): components library

**Package:** `@livekit/components-react`
**Current version:** 2.9.24, per the npm registry
([npmjs.com/package/@livekit/components-react](https://www.npmjs.com/package/@livekit/components-react)).

**Peer dependencies** (must already be in the app, or get added):
- `react` >=18, `react-dom` >=18
- `livekit-client` ^2.20.1 (the underlying WebRTC client SDK — this is the
  real weight; components-react is a thin React layer over it)
- `@livekit/krisp-noise-filter` (optional peer, only needed for AI noise
  suppression — skip it, out of scope)
- also needs the companion `@livekit/components-styles` package for default
  CSS (per the
  [installation guide](https://docs.livekit.io/reference/components/react/installation/))

**Direct dependencies:** `@livekit/components-core` (the framework-agnostic
core the React hooks wrap), `clsx`, `jose`, `events`, `usehooks-ts`, `tslib`.

**What it provides for free**, confirmed via the
[component/hook reference](https://docs.livekit.io/reference/components/react/component/gridlayout/)
and the [components-js GitHub repo](https://github.com/livekit/components-js):
- **`ParticipantTile`** + `useParticipantTile` hook — renders a participant's
  video/avatar with data attributes (`data-lk-audio-muted`,
  `data-lk-speaking`, `data-lk-source`, etc.) for CSS-driven styling — this
  is the participant-tile building block a Discord-style grid needs.
- **`GridLayout`** + `useGridLayout` — auto-fits tiles to available space,
  reduces visible-tile count under space pressure, and prioritizes speaking
  participants/screen-shares in the layout — handles the "who's on screen"
  layout logic so it doesn't need to be hand-built.
- **Audio rendering** — `AudioTrack`/`useTracks` handles attaching remote
  audio tracks to `<audio>` elements automatically; no manual WebRTC
  track-to-element wiring needed.
- **Screen share view** — screen-share tracks come through the same
  `ParticipantTile`/track-source machinery (`TrackSource.SCREEN_SHARE`), with
  layout hooks prioritizing them automatically.
- **`VideoConference`** — a full prefab (grid + focus layout + control bar +
  basic non-persistent chat) that can be used as-is or as a reference for a
  custom Discord-style layout.

**Bundle weight:** no bundle-size figures are published in the docs/README
(unable to confirm a byte figure live); expect this to be a non-trivial
addition given the WebRTC client SDK underneath (`livekit-client` itself
wraps the browser WebRTC API, encryption, adaptive stream, etc.) — worth a
bundlephobia/build-size check once it's actually installed, but there's no
lighter official alternative for a full-featured client, and hand-rolling
raw `livekit-client` without the components layer would just reimplement
`ParticipantTile`/`GridLayout` by hand.

## 3. Presence without joining: webhooks

**Delivery:** LiveKit's server sends an HTTP **POST** to a URL(s) you
configure (self-hosted: `webhook` section of the server YAML config;
Cloud: Settings → Webhooks in the dashboard, per-API-key). Payload is a
`WebhookEvent` JSON body with `Content-Type: application/webhook+json`.
Source: [docs.livekit.io/home/server/webhooks](https://docs.livekit.io/home/server/webhooks).

**Auth:** every webhook request carries an `Authorization` header containing
a signed JWT whose claims include a sha256 hash of the payload body, signed
with the API key/secret pair associated with that webhook. Verify it
server-side with the SDK's webhook receiver (`WebhookReceiver` in Node; Go
and Java have equivalent receiver helpers) — this validates the signature
against your known API secret and decodes the event, guarding against
spoofed webhook calls. **Must use the raw POST body for verification**, not
a pre-parsed JSON body, since the signature is computed over the raw bytes.

**Relevant events** for "who's in the channel" without actually joining:
`room_started` (fires when the first participant joins an empty room),
`room_finished` (fires when the room closes — either explicit `room.close()`
or empty-room timeout), `participant_joined` (fires once the participant's
media connection is fully established/`active`), `participant_left` (fires
after the participant's leave cleanup completes). There's also
`participant_connection_aborted`, `track_published`/`track_unpublished`, and
egress/ingress events, not directly needed for presence.

**Delivery guarantees:** LiveKit retries failed deliveries multiple times
before giving up on an event, and preserves ordering — a newer event won't
be delivered ahead of an older one still pending/retrying. This is
best-effort, not exactly-once — worth reconciling periodic state (e.g. via
`ListRooms`/`ListParticipants`) rather than trusting the webhook stream as
the sole source of truth for presence, especially across a service restart
where in-flight webhooks could be missed.

**Cloud vs self-hosted parity:** no functional or payload differences are
documented between the two — same event set, same JWT-signed
`Authorization` header, same retry/ordering behavior. The only difference is
*configuration surface*: Cloud webhooks are set up per-API-key in the
dashboard, self-hosted webhooks are set in the server's YAML config file.
For a bring-your-own model this matters operationally (an owner using Cloud
configures the webhook URL in their LiveKit Cloud dashboard; an owner
self-hosting sets it in their `livekit.yaml`) but the receiving/verification
code on Nexul's side is identical either way.

## 4. Config surface: is URL + API key + secret really everything?

**Yes, for the client/server SDK connection itself** — `LIVEKIT_URL`
(a `wss://` URL, shown at the top of the Cloud project dashboard, or the
self-hosted server's own address), `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`
are the full credential set the SDKs need to mint tokens and call the room
APIs on both Cloud and self-hosted. Source:
[docs.livekit.io/cloud/keys-and-tokens](https://docs.livekit.io/cloud/keys-and-tokens/)
and the LiveKit Python API reference. No region field is required in the
SDK/token layer — Cloud handles region routing transparently behind the
single project URL; "region" only surfaces as a Cloud dashboard concept for
where agent workers run, not as something the app's config needs to supply.

**What's NOT covered by those three values, and belongs in "document, don't
solve" territory for self-hosted owners** (per
[docs.livekit.io/home/self-hosting](https://docs.livekit.io) network/ports
guidance):
- **TCP 7880** — the signaling/WebSocket API port, normally sits behind a
  load balancer terminating TLS (this is what the `wss://` URL points at).
- **UDP 50000–60000** (configurable range) — ICE/media; each participant
  consumes two ports from this range. This is the one that trips up
  self-hosted owners behind NAT/firewalls/most consumer routers.
- **TCP 7881** — ICE-over-TCP fallback for when UDP is blocked.
- **UDP 7882** — optional single-port ICE/UDP mux, an alternative to opening
  the whole 50000–60000 range.
- **TURN**, needed when clients are behind symmetric NAT and can't
  connect via plain ICE — TURN/TLS on 5349 (put behind a network LB, or use
  443 if not using one) and TURN/UDP (also doubles as STUN) on 3478. LiveKit
  self-hosted can run a built-in TURN server, but it's a separate piece of
  config from the three core credentials.

None of this is Nexul's problem to solve (per `map.md`, dogfood
one-click hosting of LiveKit itself is explicitly out of scope) — it only
needs to be surfaced as documentation/checklist copy for an owner who
chooses self-hosted over Cloud, e.g. "make sure UDP 50000-60000 (or your
configured range) is open, and set up TURN if your users are behind
restrictive NATs."

## 5. Licensing

`github.com/livekit/livekit` — the SFU media server itself — is licensed
under the **Apache License 2.0**, confirmed by fetching the repo's `LICENSE`
file directly
([raw.githubusercontent.com/livekit/livekit/master/LICENSE](https://github.com/livekit/livekit/blob/master/LICENSE)).
The client/server SDKs (including `server-sdk-go` and
`components-react`/`components-js`) are Apache 2.0 as well (spot-checked via
[server-sdk-kotlin's LICENSE](https://github.com/livekit/server-sdk-kotlin/blob/main/LICENSE),
same license across the org's SDK repos).

Apache 2.0 is fully permissive for the bring-your-own model: it allows
commercial use, self-hosting, and modification without any copyleft
obligation or fee back to LiveKit. An owner can self-host the open-source
server for free under this license, or use LiveKit Cloud (a separately
priced hosted offering built on the same open-source core) — either way,
Nexul integrating against a URL + key + secret that the owner supplies
is not constrained by LiveKit's license in any way. No further license
research is needed here.
