# 02 — Routing mechanics: what the code covers vs what's missing

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Gap analysis for hostname→container routing, both entry paths, grounded in
the actual code (`internal/dns/`, `internal/dns/cloudflare/`, deploy/runner
service provisioning):

1. **Tunnel path:** what does the existing tunnel lifecycle already do
   (create, route hostname → tunnel, ingress rules, credential rotation)?
   What exactly is missing to route a hostname to a *specific container on
   a specific docker network* — i.e. does `cloudflared` on the same
   network resolve container names via Docker DNS, and do our ingress
   rules support `http://<container-name>:<port>` origins today?
2. **Reverse-proxy path:** nothing proxy-specific exists in code. For a
   Nexul-deployed Traefik/Caddy/nginx: how would hostname→container
   routes be managed programmatically (docker labels vs config file vs
   API), and which proxy fits the "Nexul writes the routes" model
   best with least machinery?
3. **Cloudflare API surface:** confirm which record/tunnel operations the
   implemented client covers vs what deploy→domain needs (wildcard
   records? per-hostname CNAMEs to `<tunnel-id>.cfargotunnel.com` exist
   already?).

Output: a findings doc listing existing capabilities, the concrete gaps,
and a recommendation per path. Facts from code first, then official
Cloudflare/proxy docs for anything the code doesn't answer.

## Answer

Tunnel path is solid on lifecycle (create/route/rotate/delete, and the
CNAME-to-`<tunnel-id>.cfargotunnel.com` record already exists), but has
three real gaps: no helper builds the `http://<container-name>:<port>`
ingress origin from a target service, nothing enforces that `cloudflared`
and its target share a docker network, and `RouteTunnelHostname` replaces
the whole ingress list rather than appending — a second hostname on the
same tunnel would silently clobber the first. Docker's embedded DNS does
resolve container names by container name on user-defined networks (not
the default bridge), and every `run`-strategy service already requires a
non-default network, so the mechanism works today — it's just not wired
up or validated. Reverse-proxy path has zero proxy-specific code; nothing
picks a proxy or a route-write mechanism. Recommendation: Traefik with its
Docker label provider — it reads routes straight from container labels the
runner already sets via `docker run`, so it needs the least new machinery
(no persisted route table, no config file, no reload signal), at the cost
of Traefik needing Docker socket access. Cloudflare API surface covers
everything current lifecycle needs; wildcard records and multi-hostname
ingress are the two gaps to close before branch/preview deployments (track
B) can share one tunnel or wildcard hostname.

Full findings: `.scratch/integrations/research/02-routing-mechanics.md`
