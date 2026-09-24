# 16 — Verify the tunnel-per-computer path

**Type:** research
**Status:** resolved
**Blocked by:** 15

## Question

The connection decision rests on three facts. Confirm each against the code
and current Cloudflare and T3 Code sources:

1. Nexul's existing Cloudflare tunnel code (`internal/dns`) can create a
   remotely managed tunnel, fetch its connector token, route a hostname to a
   port on a machine that is **not** a docker runner (a laptop running
   `cloudflared` as a service), report connector status, and delete it all
   again. What changes are needed?
2. Cloudflare Access applications, policies, and service tokens can be
   created and deleted through the API with a scoped API token, on a free
   Cloudflare plan; which token permissions that adds to what Nexul's
   Cloudflare connector asks for today.
3. T3 Code's pairing exchange and its WebSockets work through a Cloudflare
   Tunnel hostname with Access service-token headers added, and whether
   `t3client` can add those headers through its custom HTTP client.

Also: the per-OS commands to install `cloudflared` as a service with a
tunnel token (Linux, macOS, Windows), and how connector status reads over
the API so the dialog can wait for "online".

Findings to `research/tunnel-per-computer.md`; if a fact fails, reopen
ticket 15.

## Answer

Full findings: [research/tunnel-per-computer.md](../research/tunnel-per-computer.md).
All three facts hold; no blocker, ticket 15 stays resolved. Nothing was
exercised against a live tunnel yet.

- **Tunnel code works as-is** (`internal/dns/cloudflare/tunnel.go`: create,
  token, route a hostname to `http://127.0.0.1:<port>`, CNAME, status,
  rotate, delete). Only the docker launch of `cloudflared` assumes a runner;
  laptops get an install command instead.
- **Ordering matters.** Create the Access app *before* the CNAME so the
  hostname is never public. Teardown: rotate the token (disconnects the
  laptop — Cloudflare refuses to delete a tunnel with live connections),
  delete the tunnel, the DNS record, then the Access app; `DeleteTunnel`
  alone leaves the CNAME and route behind.
- **T3 Code needs nothing special** through a tunnel: no Host or Origin
  checks on its API or WebSocket paths.
- **`t3client` can carry the Access headers** everywhere through
  `Options.HTTPClient`: one header-adding wrapper, applied only to computer
  hostnames so the secret never goes to URL-paired machines.
- **Access by API works on the free plan.** One **instance-wide service
  token**, admitted by every computer's Access app — Cloudflare allows 50
  service tokens per account, so one per computer would cap an instance at
  50 computers. Set `service_auth_401_redirect` for a clean 401.
- **The Cloudflare connector asks for two more permissions**: Access: Apps
  and Policies Edit, Access: Service Tokens Edit.
- **One manual prerequisite**: the owner enables Zero Trust once in the
  Cloudflare dashboard (team name and plan, payment details even on free)
  before any Access call works. The pairing flow must detect and explain it.
- **Per-OS install**: Linux via the package repo then `sudo cloudflared
  service install <token>` (system service only); macOS via Homebrew, a
  login service is enough; Windows via MSI or winget, admin prompt, manual
  updates.
- **"Online" needs two checks**: Cloudflare's tunnel status turning
  `healthy` only proves `cloudflared` reached Cloudflare, so the dialog also
  probes T3 Code's `/.well-known/t3/environment` through the hostname.
  Re-pointing a hostname takes 1–2 minutes.
- **Checks for implementation**: a service-token WebSocket upgrade through
  Access; Cloudflare's idle-WebSocket timeout on the presence connection (a
  small ping fixes it); the connector's permission check must not mint a
  real service token; whether T3 Code keeps its port across restarts (the
  route targets a fixed port).
- **Rough size**: an Access client (~150 lines), per-computer create and
  teardown (~200 lines), one stored field for the Access app id, the header
  wrapper, and the two connector permissions.
