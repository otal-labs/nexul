# Wayfinder map: integrations

> Section codes in this map and its tickets (`DP1`, `RN3`, `DN4`, `CN3a`,
> `GP2`, `TP1`, `AM5`, `CH6`…) point into the per-domain requirement docs
> that were dissolved into `docs/adr/` on 2026-09-11. To read the original
> text: `git show 5257e483:docs/<domain>Domains.md`.

One map, three tracks (settled at charting, 2026-08-27): Cloudflare
deploy→domain, branch-driven deployments, LiveKit voice channels.

**Status (2026-08-27): implemented.** The owner delegated resolution and
execution ("Agent mode, do the work"); all three tracks were built the
same day via an agent-mode fleet and merged to master through the
`feature/integrations` PR. Decision records live in the tickets (each
flagged *owner-delegated* — open to revision on review). Still open:
tickets [04](issues/04-expose-service-ux.md) and
[11](issues/11-voice-channel-ui-prototype.md) await the owner's reaction to the
built UI. The LiveKit client was verified live against the owner's real
LiveKit Cloud server (ListRooms + token mint), and end-to-end in the
running app: LiveKit connector saved through Settings with verify-at-save
passing, voice channel `lounge` created, join reached the WebRTC
connecting state. The remaining human step is granting the mic permission
and seeing the call view live (ticket 11's reaction).

## Destination

Three tracks clear to build (specs written, implementation ticket queues
ready for agent-mode `/implement`):

- **A — Cloudflare deploy→domain:** any deployed service can be hosted at a
  hostname on a domain the owner controls, via Cloudflare tunnel or reverse
  proxy; Cloudflare's connector card goes live in Settings with DB-backed
  app config (env vars gone).
- **B — Branch-driven deployments:** merge into `dev` → deploy to QA, merge
  into `main` → deploy to prod, merge into `feature/x` → a preview
  deployment at a derived hostname (`feature-x.example.com`), cloned from
  the base service with the same env (shared DB), on a chosen docker
  network.
- **C — LiveKit voice channels:** Discord-style voice channels in chat with
  screen share and camera, on a bring-your-own LiveKit server (Cloud or
  self-hosted — URL + API key + secret, stored DB-encrypted).

## Notes

- Domain docs (dissolved into `docs/adr/` on 2026-09-11; in git history) — `dnsDomains.md`,
  `deployDomains.md`, `connectorsDomains.md`, `chatDomains.md` ground every
  ticket; override where they miss something and update them when a ticket
  closes.
- Standing rule (the owner, charting round 1): **DB-encrypted config over env
  vars, everywhere.** Nexul should be maximally configurable at
  runtime; no new env-var config, and existing env-var config (Cloudflare
  OAuth app) migrates when touched. Secrets never logged or serialized.
- Grilling tickets: invoke `/grilling` + `/domain-modeling`. The prototype
  ticket additionally runs `design-mode`. Research tickets follow
  `/research`; findings land in `.scratch/integrations/research/`.
- Implementation (after this map) happens via agent mode; this map is
  planning only.

### Settled at charting (grilling rounds 1–2)

- One map, three tracks; Cloudflare first.
- Track A is both halves: deploy→domain is the headline, connectors
  migration the cleanup.
- Track C v1 scope: voice channels + screen share + camera. No DM calls.
- LiveKit is bring-your-own (URL + key + secret); works for Cloud and
  self-hosted identically.
- Branch deployments start from clone-with-same-env (same DB); the
  deployment picks a docker network, and each network has an exit node
  (tunnel or reverse proxy) giving it internet reachability via domain.

## Decisions so far

<!-- one line per closed ticket: gist + link -->

- [08 — LiveKit integration facts](issues/08-livekit-integration-facts.md) — Go SDK `server-sdk-go/v2` (v2.18.1) + `@livekit/components-react` (2.9.24) confirmed current; screen share is a `TrackSource`, not a separate grant; webhooks are JWT-signed and identical on Cloud/self-hosted (best-effort delivery); URL+key+secret is the full config surface (self-hosted also needs UDP 50000-60000/TURN, doc-only); server and SDKs are Apache 2.0.
- [02 — Routing mechanics gap analysis](issues/02-routing-mechanics-gap-analysis.md) — tunnel lifecycle is solid but lacks an origin-URL helper, network-match validation, and multi-hostname ingress (currently replace-only); reverse-proxy path has zero proxy-specific code; recommend Traefik's Docker label provider as least machinery for "Nexul writes the routes."
- [01 — Networks, exit nodes, and the deploy→domain model](issues/01-network-exit-node-model.md) — canonical term **gateway** (tunnel or proxy service, one per docker network); dns owns gateways + exposures (hostname→service); run-strategy only in v1. *(owner-delegated)*
- [03 — Cloudflare onto the connectors surface](issues/03-cloudflare-connector-migration.md) — reuse dns's OAuth impl behind connectors with DB app-config; tokens move to `connector_credentials`; env vars removed; manual API-token path via ticket 10's mechanism. *(owner-delegated)*
- [05 — Environment vs branch model](issues/05-environment-vs-branch-model.md) — no environment object; **branch deploy rules** on the service ({pattern, network, hostname template, name suffix}); exact + wildcard are one mechanism. *(owner-delegated)*
- [06 — Branch deployment lifecycle](issues/06-branch-deployment-lifecycle.md) — derived service records, `<base>-<branch-slug>` naming, clone-with-same-env, auto-teardown on branch delete, DP5 per clone. *(owner-delegated)*
- [07 — Merge → deploy automation](issues/07-merge-deploy-automation.md) — normalized push event is the sole trigger; deploy domain owns the consumer; rules are the opt-in. *(owner-delegated)*
- [09 — Voice channel domain model](issues/09-voice-channel-domain-model.md) — fifth conversation kind `voice_channel`; `internal/voice` owns LiveKit (minimal stdlib client bias); presence = webhooks + ListRooms reconciliation; spec written as `docs/voiceDomains.md`. *(owner-delegated)*
- [10 — Non-OAuth connectors](issues/10-non-oauth-connectors.md) — `Manual []CredentialField` on Connector, encrypted-JSON storage in `connector_credentials`, verify-at-save; LiveKit is the first manual connector. *(owner-delegated)*

## Not yet specified

- **Per-branch env overrides** — a feature deployment pointing at its own
  DB instead of the shared one. Additive over ticket 06's clone model;
  revisit once the lifecycle model exists.
- **Branch-deployment management UI** — where previews are listed, torn
  down, promoted. Waits on tickets 05–06.
- **Rich voice presence** — "in voice" indicators outside the channel list
  (sidebar, member list, board). Waits on ticket 09's presence model.
- **Non-Cloudflare DNS providers' interaction with branch deployments** —
  DN1's provider interface is the seam; whether hostname derivation needs
  provider-specific care is invisible until 06 resolves.

## Out of scope

- **Additional DNS providers (Namecheap, Squarespace, …)** — DN1's
  interface already keeps them additive; wiring a second provider is a
  fresh effort.
- **Domain registration/purchase** — dnsDomains boundary; Nexul
  manages records for domains you own.
- **Nexul-deploys-LiveKit (one-click dogfood)** — ruled out of v1 at
  charting (round 2 Q2): bring-your-own decided; SFU networking (UDP/TURN)
  doesn't ride the tunnel story anyway. Fresh effort if ever.
- **DM / group calls** — charting round 1 Q3: channels are the v1 core;
  calls in DMs are future direction.
- **TLS automation for non-Cloudflare paths** — already deferred in DN5.
