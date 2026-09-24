# How a remote Nexul can reach a user's T3 Code

Verified 2026-09-24 against `pingdotgg/t3code` at `main` commit
`b2b43bef73447c483ceae486890cb79f01c369cb` (nightly `v0.0.43-nightly.20260924.2200`; latest
stable is `v0.0.42`, 2026-09-16, whose notes already include T3 Connect work), read through the
GitHub API, and against this repository's working tree. Nexul's T3 client is pinned to the
`v0.0.34` wire protocol (`internal/t3client/client.go`), so behavior between v0.0.34 and main
is not guaranteed to match.

## Summary

- T3 Code has four remote-access paths of its own: **direct pairing** to a reachable
  address (`t3 serve --host <ip>`, or desktop "Network access"), **Tailscale Serve**
  (`t3 serve --tailscale-serve` / `t3 pair --tailscale`, an `https://<machine>.<tailnet>.ts.net/`
  URL), **desktop-managed SSH** (the T3 desktop app starts a server on a remote host and
  forwards a port), and **T3 Connect**, a hosted relay run by T3 that gives each linked
  environment a managed **Cloudflare Tunnel**. The T3 server spawns `cloudflared` for it.
- The T3 Connect relay is a control plane only. Clients get a short-lived credential from
  the relay and then talk to the tunnel hostname directly. Access needs a Clerk account (the
  user's T3 Connect login) and a DPoP key, and the relay's public client IDs are limited to
  `t3-web` and `t3-mobile`. Nothing in the code or docs describes third-party server
  integrations.
- `t3 pair` prints `<base>/pair#token=<credential>` plus the raw token and an expiry. The
  token is one-time and lasts 5 minutes by default (`--ttl`). On a loopback-bound server it
  warns that the URL is only reachable locally.
- Nexul's pairing accepts any `http(s)` URL (`internal/pairing/model.go`). A Tailscale Serve
  URL therefore works **with no code change**, provided the Nexul server can reach the
  tailnet. The same goes for a user-run Cloudflare Tunnel with no Access policy.
- The runner channel (`/ws/runner`) is authenticated by one **instance-wide shared secret**
  and carries flat JSON job frames with a 64 KiB read limit and one job slot. It is not
  per-user and not a byte stream. A relay would need its own endpoint with a per-user
  credential. The automations dial-in (`/ws/automations`, per-automation token) is the
  closer precedent.
- Nexul's T3 traffic is HTTP (`/.well-known/t3/environment`, `/oauth/token`,
  `/api/auth/websocket-ticket`) plus WebSockets (`/ws`). A new WebSocket is opened per
  list/turn call, alongside one held for presence. A relay has to carry several concurrent
  full-duplex streams per computer. `t3client.Options.HTTPClient` is an existing seam where
  a relay-backed transport could be injected.
- The Nexul desktop app is a thin Electron window. Its connection token holds no identity,
  and nothing keeps running after the window closes (except the default macOS behavior).
  Today it has no background process that could host a relay.
- Nexul's Cloudflare tunnel support (`internal/dns`) uses the **instance's** Cloudflare
  account and deploys `cloudflared` as a Nexul service through a runner. Its API calls
  (create tunnel, route hostname) could be reused. Its agent provisioning assumes a runner
  on the target machine, and it has no Cloudflare Access support.

## 1. T3 Code's own remote story

### What `t3 pair` produces

Source: [`apps/server/src/cli/pair.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/cli/pair.ts),
[`apps/server/src/startupAccess.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/startupAccess.ts).

- It finds a running server through `server-runtime.json` and a probe of
  `/.well-known/t3/environment`, then mints a pairing link in that server's database:
  `createPairingLink({ scopes: AuthStandardClientScopes, subject: "one-time-token", label: "t3 pair" })`.
- TTL flag: `"Token TTL, for example 5m, 1h, or 15 minutes. Defaults to 5 minutes."`
- Output: a QR code, then `Pairing URL: …`, `Token: …`, `Expires: <ISO>`.
- URL form (`buildPairingUrl`): base URL with path `/pair`, and the token in the **fragment**
  (`#token=<credential>`), so it never reaches a server log.
- Base URL without `--tailscale`: `http://<host>:<port>`. For a wildcard bind it uses the
  first non-internal IPv4 address; with no host it uses `localhost`. On a loopback base it
  adds: `"This URL is only reachable from this machine. Re-run with --tailscale, or restart the server with a reachable --host."`
- With `--tailscale` it configures `tailscale serve` to proxy the local port and pairs through
  `https://<MagicDNS name>[:port]/`. It refuses to overwrite a mapping that fronts a different
  environment or a non-T3 service. The mapping persists across restarts.

Nexul's side ([`internal/t3client/pair.go`](../../../internal/t3client/pair.go)) exchanges the
token at `POST <serverURL>/oauth/token` using the RFC 8693 token-exchange grant and
`subject_token_type=urn:t3:params:oauth:token-type:environment-bootstrap`. It receives a bearer
session and falls back to 30 days when `expires_in` is absent ("T3's DEFAULT_SESSION_TTL").

### Binding beyond localhost and built-in remote features

Source: [`docs/user/remote-access.md`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/docs/user/remote-access.md).

- LAN or private network: `t3 serve --host <private-ip>`. On the desktop app,
  **Settings → Connections → Network access**, which restarts the app. `"A loopback address
  such as 127.0.0.1 reaches only the device opening the link."`
- Tailscale HTTPS: `t3 serve --tailscale-serve`, or `t3 pair --tailscale` for a running
  server. The desktop toggle is **Tailscale HTTPS**.
- Desktop-managed SSH: `"T3 Code starts or reuses a server there and opens the port forward
  for you."` The T3 client initiates this toward a remote host, which is the opposite
  direction from what Nexul needs.
- Hosted web app (`app.t3.codes`): `"needs an HTTPS endpoint. It connects directly to your
  server; a hosted pairing link does not make an unreachable backend reachable."`
- T3 Connect (`t3 connect`, or desktop **Settings → Connections**): `"makes an environment
  available to your other devices without setting up router forwarding."`

### T3 Connect: hosted relay plus managed Cloudflare Tunnel

Sources: [`docs/internals/t3-connect.md`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/docs/internals/t3-connect.md),
[`infra/relay/README.md`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/infra/relay/README.md),
[`apps/server/src/cloud/ManagedEndpointRuntime.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/cloud/ManagedEndpointRuntime.ts),
[`apps/server/src/cli/connect.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/cli/connect.ts),
[`packages/contracts/src/relay.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/packages/contracts/src/relay.ts).

- There is an official hosted relay (a Cloudflare Worker; the production stage is the
  `relay.<zone>` domain). `"T3 Connect uses Clerk for cloud identity. The relay manages
  environment links, credentials for reaching environments, and managed tunnel
  allocations. After bootstrap, clients send application traffic through the environment's
  tunnel hostname; the relay Worker does not proxy their HTTP or WebSocket sessions."`
- The transport is a Cloudflare Tunnel. The T3 server spawns `cloudflared tunnel run` with
  `TUNNEL_TOKEN` taken from the relay's `endpointRuntime` (`providerKind: "cloudflare_tunnel"`),
  supervised with crash backoff. `t3 connect` offers to download the "relay client", which
  prints `Relay client ready · cloudflared <version>`. Tunnel hostnames look like
  `prod-<digest>.<RELAY_TUNNEL_ZONE_NAME>`. `RelayManagedEndpoint` carries `httpBaseUrl` and
  `wsBaseUrl`. A third provider kind, `t3_relay`, exists in the schema, but in the runtime
  only `cloudflare_tunnel` is wired; anything else returns `unsupported`.
- Client flow: sign in with Clerk, then `POST /v1/client/dpop-token`, which exchanges the
  Clerk token for a DPoP-bound relay token (`client_id` is one of `"t3-mobile" | "t3-web"`).
  Next, `POST /v1/environments/:id/connect` with `clientProofKeyThumbprint`. The relay asks
  the environment to mint a one-time credential bound to that key, and the client exchanges
  it directly with the environment for an environment session. `"The relay never receives
  that session token."`
- Limits: `"Managed tunnels expose only a validated loopback HTTP origin."` There is a
  per-account tunnel cap (`environment_link_limit_exceeded`). On a normal shutdown the tunnel
  is released, and the hostname is kept.
- `t3 connect link --publish-only`: "Link to publish agent activity only — no managed tunnel.
  Reach this environment out of band (e.g. Tailscale)."

### How remote clients are authenticated

Source: [`docs/internals/environment-auth.md`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/docs/internals/environment-auth.md).

- `"The environment issues its own sessions and enforces their capabilities."` Pairing
  delegates scopes, and an exchange can narrow a grant but never widen it. Browser cookies,
  bearer tokens, and DPoP tokens all map onto the same scoped session model.
- `"Bearer and DPoP clients obtain short-lived WebSocket tickets through authenticated HTTP
  so long-lived tokens stay out of socket URLs."` Nexul already does this in `mintWSTicket`.
- Scope reach: `"orchestration:read permits reading files the server account can read,
  including absolute paths outside a project."` The scopes Nexul requests are
  `orchestration:read orchestration:operate terminal:operate review:write relay:read`.
- Revocation: the host's **Settings → Connections** or `t3 auth`.

### Could Nexul use T3 Connect?

Nothing documents a third-party server acting as a T3 Connect client. The obstacles found:

- The user's Clerk login has to be held server-side, which Nexul does not do today.
- The client IDs are fixed to `t3-web` and `t3-mobile`.
- A DPoP key has to be implemented.

**Not verified:** whether a Nexul server could reach the managed tunnel hostname (public
HTTPS/WSS) using an ordinary `t3 pair` token and the existing bearer flow. The docs say
authority "survives transport changes", but nothing confirms that a plain `/oauth/token`
exchange is accepted through the tunnel, or where a user would see the hostname. It needs a
live test before anyone relies on it. Because the relay releases the tunnel on shutdown,
the endpoint is also only up while the T3 server runs.

## 2. Nexul's runner pattern

Sources: [`internal/runner/handler.go`](../../../internal/runner/handler.go),
[`internal/runner/protocol.go`](../../../internal/runner/protocol.go),
[`internal/runner/client.go`](../../../internal/runner/client.go),
[`internal/runner/config.go`](../../../internal/runner/config.go),
[`server/cmd/routes.go`](../../../server/cmd/routes.go),
[`internal/automations/dialin.go`](../../../internal/automations/dialin.go).

- **Endpoint:** `/ws/runner` is mounted on the dedicated WS listener and on the main HTTP
  listener, the second so that a proxy only has to forward one origin.
- **Authentication:** `?token=` is compared in constant time against **one instance-wide
  secret** (`repo.Secret`). `runner_id`, `name`, `version`, `machine`, `os`, and `arch` are
  self-reported query parameters. A new connection with the same `runner_id` replaces the
  old one. The runner reads `NEXUL_SERVER_WS` and either `NEXUL_RUNNER_SECRET` or
  `NEXUL_RUNNER_SECRET_FILE`. It carries no user identity.
- **Framing:** one flat JSON `Frame` per WS message, discriminated by `type` (heartbeat,
  assign_build, assign_deploy, cancel, progress/result, deploy_log, discover, join_networks,
  update, assign_upgrade). Correlation uses the job `id`. There is no stream or channel
  abstraction and no binary frames. The inbound read limit is 64 KiB by default. Each
  connection holds a single job slot (`runnerConn.job`).
- **Presence:** the runner sends a heartbeat every 10s, and a watchdog closes the socket
  after 3 missed beats. Connecting calls `repo.SetConnected(true)` and publishes
  `runner.connected`; a disconnect sets it to false, fails any in-flight job, and publishes
  `runner.disconnected`. The client reconnects with exponential backoff (1s up to 30s).
- **Per-principal precedent:** `/ws/automations` authenticates with a per-automation token
  (`svc.AuthenticateToken`) and keys connections by automation ID. This is the existing
  per-identity dial-in shape.

### What relaying T3 over an outbound channel would take

Nexul's T3 client ([`internal/t3client/client.go`](../../../internal/t3client/client.go),
[`harness.go`](../../../internal/t3client/harness.go),
[`internal/presence/keeper.go`](../../../internal/presence/keeper.go)) makes these calls:

- HTTP GET `/.well-known/t3/environment`, which is also re-probed at turn start.
- HTTP POST `/oauth/token` and `/api/auth/websocket-ticket`.
- A WebSocket on `/ws?wsTicket=…` running Effect RPC. Streams need client `Ack` frames, and
  inbound frames can reach 32 MiB (`maxFrameBytes = 1<<25`; attachments travel inline).
- Every `ListProjects`, `ListProviders`, `StartTurn`, `Interrupt`, and `Answer` opens its own
  `Connect`, while the presence keeper holds one more connection per computer as long as the
  owner has a browser socket open.

A relay therefore has to carry **several concurrent full-duplex streams per computer**, both
HTTP request/response and long-lived WebSockets. A request/response RPC over the runner
frame set would not be enough. The seams that already exist:

- `t3client.Options.HTTPClient` is honored for every HTTP call and for `websocket.Dial`
  (`DialOptions{HTTPClient}`). A `http.Transport` whose `DialContext` opens a stream over the
  user's relay connection would leave the T3 protocol code untouched. Today it is wired
  once, in `server/cmd/services.go`, as `t3client.NewHarness(t3client.Options{Logger: logger})`,
  so it would need to become per-computer.
- `github.com/coder/websocket` v1.8.15, already a dependency, provides
  `websocket.NetConn(ctx, c, msgType) net.Conn`, which turns a WS into a byte stream.
  Several streams over one WS would need a multiplexer, either a library (none is in
  `go.mod`) or one WS per stream.
- On the user's machine, a relay agent would dial Nexul with a per-user credential, accept
  stream-open requests, and dial `127.0.0.1:<t3 port>` for each. Presence would come from
  the relay socket (the runner heartbeat pattern) rather than from the held T3 WebSocket.
- Security properties: no inbound port on the user's machine, and T3's own bearer auth
  still applies end to end. The relay has to pin the local target to the T3 loopback
  port, so that a compromised Nexul server cannot use it to reach other local services.
  This is the same concern T3 addresses by exposing "only a validated loopback HTTP origin".

## 3. Nexul's desktop app

Sources: [`desktop/package.json`](../../../desktop/package.json),
[`desktop/electron/main.ts`](../../../desktop/electron/main.ts),
[`desktop/electron/token.ts`](../../../desktop/electron/token.ts),
[`internal/auth/conn_token.go`](../../../internal/auth/conn_token.go),
[`internal/auth/model.go`](../../../internal/auth/model.go),
[`website/src/content/docs/docs/guide/desktop-app.md`](../../../website/src/content/docs/docs/guide/desktop-app.md).

- `"Nexul desktop client: a thin Electron shell around the server's web app, bootstrapped by
  connection tokens."` It loads the instance origin in a sandboxed `BrowserWindow`
  (partition `persist:nexul`), and sign-in is the normal GitHub OAuth in the same frame.
- The connection token is an HS256 JWT holding `instance_url`, `mcp_url`, `version`, `iat`,
  and `exp`, with a 30-day TTL. `"no identity or credentials, so it isn't secret."` The
  client checks only its structure. Reachability comes from a probe of `/api/auth/me`.
- Process model: the main process only handles IPC for the instance vault and probing. On
  `window-all-closed` it quits, except on macOS. There is no tray, no background service,
  no auto-start, and no Node-side network client toward the instance.
- To host a relay, it would need a main-process relay client, a per-user credential (for
  example a PAT, or a new token type minted after sign-in, since the connection token
  carries none), and a way to keep running while the window is closed. Otherwise
  presence is lost whenever the app closes.

## 4. Tunnel option

### Cloudflare Tunnel (user-run)

- `"cloudflared initiates an outbound connection through your firewall from the origin to the
  Cloudflare global network"` ([Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/)).
  T3 Connect already carries T3's HTTP and WebSocket traffic this way, which shows the
  protocol works through a tunnel.
- The user needs a Cloudflare account and zone, has to install and run `cloudflared` with a
  tunnel token and a route to `http://127.0.0.1:<t3 port>`, then pair Nexul to
  `https://<hostname>`. That URL passes `validateServerURL` today.
- A published hostname is reachable from the internet unless a Cloudflare Access policy
  gates it. Machine clients pass Access with `CF-Access-Client-Id` and
  `CF-Access-Client-Secret` headers
  ([service tokens](https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/)).
  Nexul's `t3client` sends no such headers today. With the transport injection described in
  section 2, they could be added per computer.

### Tailscale

- T3 supports it natively: `t3 pair --tailscale` publishes the server through **Tailscale
  Serve**, which is tailnet-only (`"route traffic from other devices on your Tailscale network"`;
  public exposure is the separate **Funnel** feature,
  [Tailscale Serve](https://tailscale.com/kb/1312/serve)).
- The Nexul server has to be a node on the same tailnet. Nexul has no Tailscale integration
  today (no match for `tailscale` in the repo). A Docker deployment would add a Tailscale
  container (`TS_AUTHKEY`, `TS_STATE_DIR`; `TS_USERSPACE` is on by default,
  [Docker params](https://tailscale.com/docs/features/containers/docker/docker-params)).
  Userspace mode means the Nexul container needs a route or proxy into the tailnet. That
  was not verified here.
- On a multi-user instance, every user's machine has to share a tailnet with the server,
  which makes the server a peer on each user's private network (or requires tailnet
  sharing).

### Security cost of a reachable agent endpoint

- The paired session covers `terminal:operate` and `orchestration:operate`, which means
  running shell commands and agent turns as the T3 server's OS user. `orchestration:read`
  can read any file that account can read. The endpoint is therefore a remote-execution
  surface guarded only by T3's session auth.
- A public tunnel with no Access policy puts that auth, including the unauthenticated
  `/.well-known` descriptor and the `/oauth/token` exchange, on the internet. A leaked
  5-minute pairing token, or a leaked 30-day bearer stored encrypted in Nexul, is usable from
  anywhere. T3's docs: `"Treat pairing URLs and authorization codes as passwords."`
- Tailscale Serve and Cloudflare Access narrow who can reach the endpoint at all. An
  outbound relay exposes no endpoint, but it puts the Nexul server in the path, so a
  compromised server can drive every connected user's machine. That trust already exists
  today through the stored bearer tokens.

### Nexul's existing Cloudflare tunnel support

Sources: [`internal/dns/tunnel.go`](../../../internal/dns/tunnel.go),
[`internal/dns/tunnel_usecase.go`](../../../internal/dns/tunnel_usecase.go).

- It manages tunnels in the **instance's** Cloudflare account (the API token needs the Tunnel
  permission). `CreateTunnel` stores the tunnel token encrypted, and `RouteTunnelHostname`
  adds the ingress rule and CNAME. `ProvisionTunnelAgent` deploys `cloudflare/cloudflared:latest`
  as a Nexul service through the runner/deploy path, with `TUNNEL_TOKEN` in its env and
  health checked at `:20241/ready`.
- Reusable in principle: the create, route, rotate, and delete API calls could mint a
  per-user tunnel and hostname under the instance's zone and hand its token to the user.
- Not reusable as-is: agent provisioning assumes a runner and Docker on the target machine.
  A laptop running T3 Code is not a deploy machine. The routes also carry no Cloudflare
  Access policy, so the hostnames would be public.
- Its purpose is exposing deployed stacks, not reaching user machines. The overlap is at the
  API-client level only.

## 5. Connection models compared

| Model | What the user installs or does | Ports exposed on user machine | Reuses existing Nexul code? | Effort | Risks / blockers |
|---|---|---|---|---|---|
| **A. Nexul dials T3 over Tailscale** | Install Tailscale, join the server's tailnet, run `t3 pair --tailscale`, paste URL and token | None publicly; T3 reachable on the tailnet only | Yes. Pairing, t3client, and presence are unchanged. The server needs to join the tailnet (deployment work, no Go change) | Low in code; medium in ops per install | Every user shares a tailnet with the server; the containerized server needs tailnet routing (unverified); the tailnet admin burden falls on the operator |
| **B. Nexul dials T3 over a user-run Cloudflare Tunnel** | Cloudflare account and zone, run `cloudflared` pointed at the T3 port, optionally an Access app plus service token | None locally; a public hostname at Cloudflare's edge | Pairing works unchanged without Access. Access needs per-computer headers (new). `internal/dns` API calls could mint tunnels in the instance account | Low without Access; medium with Access or Nexul-minted tunnels | Without Access, a shell-capable endpoint is on the internet behind T3 auth only; heavy setup for a non-technical user |
| **C. Outbound relay from a runner-style binary** | Install a small Nexul agent binary (service), give it a per-user credential | None | Partly: runner client patterns (reconnect/backoff, heartbeat, WS), automations-style per-principal auth, `coder/websocket` `NetConn`, `t3client.Options.HTTPClient` seam. Needs a new endpoint, per-user auth, stream multiplexing, per-computer transport, and presence from the relay socket | Medium to high | Runner auth is instance-wide and cannot be reused for per-user relays as-is; multiplexing is new code or a new dependency; the relay must pin the loopback target; one more binary to install and update |
| **D. Relay inside the Nexul desktop app** | Install the Nexul desktop app, sign in, keep it running | None | Same server side as C. Desktop gains a main-process relay; the connection token holds no credential, so a new one is needed | Medium to high, plus desktop packaging | The app quits when its window closes (non-macOS) and has no tray or background mode, so presence ends with the window unless that changes; still requires T3 Code running too |
| **E. T3 Connect (T3's hosted relay)** | Sign in to T3 Connect, run `t3 connect` | None locally; a managed Cloudflare Tunnel hostname | No. Needs Clerk tokens held server-side, a DPoP implementation, and a client ID outside `t3-web`/`t3-mobile` | High and unsupported | Clear blocker: no documented third-party client path; depends on T3's hosted service and account tunnel limits. Pairing a plain `t3 pair` token through the managed hostname is untested |
| **F. T3 direct pairing on a reachable address** | `t3 serve --host <ip>` / desktop "Network access", reachable from the server (LAN, VPN, port forward) | T3 port on the chosen interface | Yes, unchanged | None in code | Only works when the server and the user's machine share a network; port-forwarding a shell-capable endpoint to the internet carries the same risk as B without Access |
