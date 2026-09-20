# 02 — Routing mechanics: gap analysis

Grounded in `internal/dns/`, `internal/dns/cloudflare/`, `internal/deploy/`,
`internal/runner/`, and `docs/dnsDomains.md`. External facts (Cloudflare
ingress format, Docker DNS, proxy config models) are cited to official docs
or, where official docs stop short, to consistently-repeated community
practice — called out explicitly below.

## 1. Existing capabilities

### Tunnel lifecycle (Cloudflare)

Fully built end to end, both the account-scoped tunnel API and the
Nexul use-case layer:

- `Service.CreateTunnel` → `Client.CreateTunnel` — creates a
  `config_src: cloudflare` tunnel, encrypts and stores the token
  (`internal/dns/tunnel_usecase.go:18-42`, `internal/dns/cloudflare/tunnel.go:67-83`).
- `Service.RouteTunnelHostname` — does two things in one use-case: sets the
  tunnel's ingress rule via `PUT .../cfd_tunnel/{id}/configurations`
  (`internal/dns/cloudflare/tunnel.go:146-163`), and creates the CNAME
  record `<hostname> → <tunnel-id>.cfargotunnel.com` through the *same*
  `DNSProvider.CreateRecord` used for ordinary records
  (`internal/dns/tunnel_usecase.go:69-115`, record content line 89). This
  answers part of ticket question 3 directly: the per-hostname CNAME to
  `<tunnel-id>.cfargotunnel.com` already exists in code, not just in docs.
- `Service.RotateTunnelCredentials` — issues a fresh `tunnel_secret`,
  force-disconnects existing connectors (`internal/dns/cloudflare/tunnel.go:168-196`).
- `Service.DeleteTunnel` — idempotent delete (404 → success).
- `Service.ProvisionTunnelAgent` — provisions `cloudflared` as a
  Nexul-deployed service, injecting `TUNNEL_TOKEN` into its env
  (`internal/dns/tunnel_usecase.go:174-210`).
- `Service.ProvisionReverseProxy` — provisions an arbitrary image as a
  Nexul service via the same `ServiceProvisioner` seam
  (`internal/dns/tunnel_usecase.go:212-229`); nothing proxy-specific beyond
  "run this image as a service" — no routing.

All of this is exposed through HTTP and MCP (`dns_tunnel_*`,
`docs/dnsDomains.md` DN6) and covered by tests
(`tunnel_usecase_test.go`, `tunnel_test.go`, `tunnel_mcp_test.go`).

### The `ServiceProvisioner` seam and the `run` strategy

`dns.AgentSpec` (`internal/dns/tunnel.go:83-97`) carries `DockerNetwork`,
`Ports`, `Image`, `Env` — everything the deploy domain's `run` strategy
needs. The composition-root adapter `dnsProvisioner.Provision`
(`server/cmd/wire_dns.go:31-55`) maps it 1:1 onto
`deploy.ServiceDef{DockerNetwork, Ports, Env, Strategy: deploy.StrategyRun}`
and calls `deploy.Service.CreateService` + `Deploy`.

The runner's executor confirms what `run` strategy actually does on the
host: `docker run -d --name <service-name> --network <docker-network>
-p <port>...` (`internal/runner/executor.go:319-343`, mirrored in
`startBuilt` for redeploys at lines 198-222). Critically, **the container
name is the service's own name** (`internal/deploy/events.go:80`,
`ev.Service = svc.Name`) — the same name used for `--name`. So any service
Nexul deploys with the `run` strategy on network `N` is reachable
from any other container also on `N` at `http://<service-name>:<port>`,
via Docker's embedded DNS (confirmed below). This is the mechanical
building block the routing story needs — it already exists, just not
wired into the tunnel/proxy flows.

`ServiceDef.Validate` (`internal/deploy/model.go:157-160`) requires
`DockerNetwork` for the `run` strategy but never checks that it matches
any *other* service's network — nothing in code today enforces "cloudflared
and the target container are on the same network."

### Docs vs code

`docs/dnsDomains.md` DN5 and its implementation notes
(`docs/dnsDomains.md:91-117`) describe exactly what's above and no more:
entry paths are deployed services, tunnel lifecycle is built, reverse proxy
is "publishes 80/443 and routes hostnames to containers... deploy/topology's
job" — explicitly marked as *not yet done*.

## 2. Concrete gaps

### Tunnel path

1. **No automatic origin URL construction.** `RouteTunnelInput.Service`
   (`internal/dns/tunnel.go:53-61`) is a free-text string the caller must
   supply (e.g. `http://localhost:80`) — there is no helper that builds
   `http://<container-name>:<port>` from a target service's name and port.
   The caller has to know and pass the right string themselves today.
2. **No network-matching enforcement.** Nothing checks that the
   `cloudflared` agent's `AgentSpec.DockerNetwork` equals the target
   service's `DockerNetwork`. If they differ, the ingress rule will point
   at a hostname `cloudflared` cannot resolve (see next point) and the
   route will 502 at runtime with no earlier validation error.
3. **Docker-DNS resolution is real but conditional, and untested here.**
   Docker's embedded DNS server (`127.0.0.11` inside every container) only
   resolves container names on **user-defined** networks — not on the
   default bridge network
   ([Docker networking docs pattern confirmed via community sources; Docker's own
   docs establish this as the documented distinction between default bridge
   and user-defined networks](https://docs.docker.com/engine/network/)).
   Since `deploy`'s `run` strategy always requires a non-empty
   `DockerNetwork` (`model.go:157-160`), every `run`-strategy container
   Nexul creates is already on a user-defined network, so this
   condition is satisfied by construction — but nothing in the codebase
   documents or asserts that invariant, so a future change (e.g. defaulting
   to the bridge network) would silently break tunnel routing.
4. **Community pattern, not an explicit Cloudflare doc guarantee.** Multiple
   independent write-ups (Frankel's blog, thedxt.ca, bist.be — see Sources)
   describe running `cloudflared` in the same Docker Compose file/network as
   the target app and pointing ingress `service` at
   `http://<container-name>:<port>`, but Cloudflare's own configuration-file
   docs only show `localhost` examples for the `service` field
   ([Cloudflare Tunnel configuration file docs](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/configuration-file/)).
   The `service` field is just a URL cloudflared dials from inside its own
   container/process, so any hostname its resolver can reach works — this
   is a natural consequence of how cloudflared runs, not a documented
   Cloudflare feature per se. Low risk, but worth stating plainly rather
   than assuming.
5. **No teardown/re-route path.** `RouteTunnelHostname` only ever adds an
   ingress rule via `PUT .../configurations`, which *replaces* the whole
   ingress config (`internal/dns/cloudflare/tunnel.go:151-163`, body sets
   `ingress` to exactly two rules: the one hostname plus the catch-all).
   Routing a **second** hostname on the same tunnel today would silently
   overwrite the first — there's no read-modify-write of the existing
   ingress list. This matters directly for the "specific container on a
   specific network" story once more than one service shares a tunnel.

### Reverse-proxy path

Nothing proxy-specific exists in code — confirmed by the earlier grep
(`ProvisionReverseProxy` and one MCP tool description are the only proxy
mentions in the whole codebase, and both just deploy an arbitrary image).
Everything below is a gap, not a partial implementation:

1. No `ReverseProxyProvider` interface, no route CRUD, no model for
   "hostname → container:port" association at a proxy.
2. No mechanism decided for **how** Nexul would write routes: docker
   labels on the target container's `docker run` invocation (which the
   runner already constructs and could extend), a proxy-specific config
   file the runner writes and the proxy watches, or an HTTP/API call to a
   running proxy instance.
3. No decision on **which proxy** — `docs/dnsDomains.md` and one MCP tool
   description informally name Traefik, but this was never grilled or
   specified (see gap analysis below for a recommendation).
4. TLS for the reverse-proxy path is explicitly deferred
   (`docs/dnsDomains.md:60-61`, "TLS automation for non-Cloudflare paths is
   deferred") — out of scope for this ticket but worth flagging as a
   dependency for whichever proxy gets picked.

## 3. Cloudflare API surface: coverage vs need

| Need (ticket Q3) | Covered today | Where |
|---|---|---|
| Per-hostname CNAME to `<tunnel-id>.cfargotunnel.com` | Yes | `internal/dns/tunnel_usecase.go:88-90` |
| Tunnel create/list/get/delete | Yes | `internal/dns/cloudflare/tunnel.go:67-141` |
| Tunnel ingress rule set (single hostname) | Yes, but replace-only (see gap 5 above) | `internal/dns/cloudflare/tunnel.go:146-163` |
| Tunnel credential rotation + force-disconnect | Yes | `internal/dns/cloudflare/tunnel.go:168-196` |
| A/AAAA/CNAME/TXT record CRUD | Yes | `internal/dns/cloudflare/client.go:189-223` |
| Propagation check (A/AAAA/CNAME/TXT) | Yes | `internal/dns/cloudflare/client.go:228-265` |
| **Wildcard records** (`*.example.com`) | **No explicit handling** | `RecordInput.Validate` (`internal/dns/model.go:50-66`) only constrains `Type`/`Name`/`Content`/`TTL≥0` — a `Name` of `*` would pass validation and Cloudflare's API does accept `*` as a record name for wildcards, but nothing in the codebase constructs, tests, or documents this path. `recordNameFor` (`internal/dns/model.go:143-161`) never emits `*` — it only handles the apex (`@`) and exact subdomain cases. |
| **Multi-hostname ingress on one tunnel** | **No** — see gap 5 | — |

So: single-hostname tunnel routing and the CNAME-to-tunnel record are
solid and match the DN5 spec exactly. Wildcard records and multi-hostname
tunnels are the two API-surface gaps, and both matter once branch/preview
deployments (track B) want `feature-x.example.com` style hostnames sharing
one tunnel or one wildcard record.

## 4. Recommendation per path

### Tunnel path

Low-effort to close the gap: no new Cloudflare API calls needed.

1. Add a helper (dns or the composition-root adapter) that derives the
   ingress `service` URL as `http://<service-name>:<container-port>` from
   the target `deploy.ServiceDef`, instead of asking the caller for a raw
   string.
2. Enforce (validation, not just convention) that `cloudflared`'s
   `AgentSpec.DockerNetwork` matches the target service's `DockerNetwork`
   before routing — surfacing a clear `ErrInvalid` instead of a runtime
   502.
3. Change `RouteTunnelHostname` to read the tunnel's current ingress
   config before writing (`GET` then `PUT` with the existing rules plus
   the new one, still ending in the catch-all), so multiple hostnames can
   share one tunnel. This is a Cloudflare API-shape fix, not a new
   capability — the `PUT .../configurations` endpoint already accepts an
   arbitrary-length ingress array.
4. If/when wildcard hostnames are needed for branch previews, extend
   `recordNameFor` to accept a `*` subdomain explicitly rather than relying
   on it slipping through unchanged, and add a corresponding ingress rule
   (Cloudflare's tunnel ingress supports hostname wildcards like
   `*.example.com` directly in the `hostname` field — same mechanism as
   DNS wildcards, confirmed by Cloudflare's own configuration-file docs
   listing wildcard hostname matching as a supported ingress feature:
   [developers.cloudflare.com/cloudflare-one/.../configuration-file/](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/configuration-file/)).

### Reverse-proxy path — which proxy fits "Nexul writes the routes" with least machinery

Compared on one axis: how does Nexul get a hostname→container:port
route into the proxy, given it already controls the `docker run`/`docker
compose` invocation via the runner (`internal/runner/executor.go`)?

- **Traefik — Docker provider (labels).** Traefik watches the Docker
  socket and reads `traefik.*` labels directly off running containers; no
  separate config file or API call is needed at all. Nexul would
  extend the `run`-strategy `docker run` args
  (`internal/runner/executor.go:319-339`) with a handful of `--label`
  flags (router rule = hostname, service port) at deploy time, and Traefik
  picks the route up automatically as the container starts — zero new
  network calls, zero new persisted route model beyond what's already in
  `ServiceDef`. Official docs: [Traefik Docker provider](https://doc.traefik.io/traefik/reference/routing-configuration/other-providers/docker/).
  Trade-off: Traefik needs read access to the Docker socket (`/var/run/docker.sock`),
  which is a real permission the reverse-proxy container would hold.
- **Caddy — Admin API.** Caddy's admin API (`localhost:2019`, on by
  default) accepts `POST /load` with a full JSON config and hot-swaps it
  with zero-downtime connection draining — no socket access needed, no
  file writes, no reload signal. Official docs: [Caddy API](https://caddyserver.com/docs/api).
  This is clean and officially supported, but it means Nexul has to
  own and persist the *whole* routing table itself (build the JSON,
  include every existing route, POST the full document on every change) —
  more state to manage on Nexul's side than Traefik's label
  approach, since nothing there is inferred from the containers
  automatically. Docker-label-driven Caddy exists only via the
  **unofficial** `caddy-docker-proxy` plugin, not Caddy core.
- **nginx (OSS).** No dynamic config API in the open-source build (that's
  an nginx-plus-only feature). Nexul would need to render a config
  file, mount it into the container, and signal `nginx -s reload` — either
  itself or via a companion like `docker-gen`/`nginx-proxy`, which is
  extra moving parts and process management the other two don't need.

**Recommendation: Traefik with the Docker label provider.** It requires
the least new machinery given what Nexul already does — the runner
already constructs the `docker run` command for every `run`-strategy
service, so route configuration is just a few extra label arguments on a
command that's built anyway, with no separate persisted route model, no
polling, and no reload step. This also matches what the code already
informally assumes (`docs/dnsDomains.md`'s "Traefik-style service" wording
and the one MCP tool description naming Traefik). The Docker-socket
exposure is the one real cost, and should be scoped to a socket-proxy
(e.g. read-only, container-list-only) rather than the raw socket if this
gets built — a decision for the implementation ticket, not this research
ticket.

## Sources

- [Cloudflare Tunnel — Configuration file](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/configuration-file/) — ingress `service` field formats, wildcard hostname matching, top-level `originRequest`.
- [Docker networking overview](https://docs.docker.com/engine/network/) — user-defined vs default bridge network DNS resolution behavior.
- [Traefik — Docker provider reference](https://doc.traefik.io/traefik/reference/routing-configuration/other-providers/docker/) — label-based dynamic configuration, port selection rules.
- [Caddy — Admin API](https://caddyserver.com/docs/api) — `POST /load`, zero-downtime config swap.
- Community write-ups confirming the "cloudflared + app on the same Docker network, ingress service points at the container name" pattern (not from Cloudflare's own docs, which only show localhost examples): [My second Cloudflare Tunnel — blog.frankel.ch](https://blog.frankel.ch/second-cloudflare-tunnel/), [Cloudflare Tunnel with Docker — theDXT](https://thedxt.ca/2022/10/cloudflare-tunnel-with-docker/), [Cloudflare Tunnel and containers — Bist.be](https://bist.be/posts/2ndpost/).
