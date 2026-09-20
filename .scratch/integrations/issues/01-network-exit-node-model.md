# 01 — Networks, exit nodes, and the deploy→domain model

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

the owner's framing (charting round 2): a branch/service deployment "selects
which docker network it can connect to, as each docker network will have an
exit node (maybe the wrong word) — basically cloudflare tunnel to the
internet, or reverse proxy to the internet, all via domain."

Pin the domain model that makes deploy→domain real:

- **Canonical term.** Docs say "entry path" (DN5); the owner said "exit node."
  Pick one, record it in `CONTEXT.md`.
- **What owns what.** DN4 splits it: dns owns the record; deploy/topology
  own routing. Is the network↔exit-node attachment a topology concept, a
  deploy concept, or a new one?
- **Cardinality.** One exit node per docker network? Per server? Can a
  network have both a tunnel and a proxy? Can two networks share one exit
  node?
- **What "expose a service" means.** Given a service on network N with exit
  node E: a hostname routes through E to the container — who stores that
  mapping, and how do topology's `domain` nodes anchor to it (dnsDomains
  Boundaries flagged this as a follow-up)?
- **Exit nodes are themselves deployed services** (DN5: `cloudflared` /
  Traefik deployed by Nexul). How is one attached to a network —
  created as part of "make this network reachable," or linked after?

Cross-check against `internal/dns/tunnel*.go` (tunnel + ingress lifecycle
already built) and DP1 (run-strategy services already pick a docker
network; compose takes networks from the file — what does network selection
mean for compose services?).

## Answer

Resolved by owner delegation ("Agent mode, do the work", 2026-08-27) —
decisions flagged for the owner's review, grounded in the owner's charting answers and
ticket 02's findings.

- **Canonical term: gateway.** A gateway is a Nexul-deployed service
  attached to exactly one docker network, giving services on that network
  internet reachability via hostnames. The owner's "exit node" and the docs'
  "entry path" both mean this; `CONTEXT.md` gets the term. Two kinds:
  `tunnel` (cloudflared) and `proxy` (Traefik).
- **Cardinality v1:** at most one gateway per network; a gateway serves one
  network. Multiple gateways per network is fog, not v1.
- **Ownership:** dns owns gateways and **exposures** — an exposure is
  {hostname, service, container port, gateway}. This amends DN4's "routing
  is deploy/topology's job": dns already owns tunnel ingress routing in
  code, and the proxy path's routing (Traefik labels) is data the runner
  consumes at container start via a seam — dns still never imports deploy.
- **Expose a service** (tunnel path): validate the service's
  `DockerNetwork` matches the gateway's network, append (read-modify-write,
  fixing 02's replace-only bug) an ingress rule with origin
  `http://<container-name>:<port>`, create the CNAME. Proxy path: create
  the A/AAAA record; routing labels are applied at the service's next
  deploy — the expose flow offers "redeploy now".
- **Compose-strategy services:** not exposable in v1 (their networks come
  from the compose file); fog.
- Topology `domain`-node anchoring stays a follow-up (fog), unchanged.

`dnsDomains.md` DN4/DN5 to be updated by the implementing agent.
