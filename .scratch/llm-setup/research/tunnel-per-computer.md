# Verify the tunnel-per-computer path

Verified 2026-09-24 against this repository's working tree, `pingdotgg/t3code` at `main` commit
`b2b43bef73447c483ceae486890cb79f01c369cb`, `cloudflare/cloudflared` at tag `2026.9.1`
(latest release, 2026-09-11), `cloudflare/cloudflare-go` at `main` commit `a451147` (2026-09-04),
and developers.cloudflare.com (read directly, plus through the `cloudflare/cloudflare-docs` source
on Context7). Items marked **Unverified** need a live test.

## Summary

All three facts the connection decision rests on hold. Nothing blocks it. There is one manual
setup step, and there are two account limits that shape the design.

**Works as-is**

- Tunnel lifecycle in `internal/dns/cloudflare/tunnel.go`: `CreateTunnel` (remotely managed,
  `config_src: "cloudflare"`, token returned and stored encrypted), `TunnelToken`,
  `RouteTunnelHostname` (ingress rule to any `service` string, so `http://127.0.0.1:<port>` on the
  connector's machine works), the proxied CNAME in `Service.RouteTunnelHostname`,
  `GetTunnel` / `Service.TunnelStatus` (Cloudflare's `status`: `inactive`, `healthy`, `degraded`,
  `down`), `RotateTunnelCredentials` (new secret plus connection cleanup), `RemoveTunnelHostname`,
  `DeleteTunnel`, and `DeleteRecord`.
- T3 Code needs nothing special behind a tunnel. T3 Connect's own relay routes
  `hostname → http://127.0.0.1:<port>` with no `originRequest` overrides. T3's server has no
  Host or Origin allowlist on its API and WebSocket routes.
- `t3client` sends every HTTP call and every WebSocket handshake through
  `Options.HTTPClient`. `coder/websocket` performs the upgrade with `HTTPClient.Do`, so a
  header-adding `RoundTripper` covers both.
- Cloudflare Access apps, policies, and service tokens are all creatable and deletable by API
  with account-scoped token permissions, on the Zero Trust Free plan. Service tokens do not
  consume seats.

**Needs changes (rough size)**

1. Cloudflare Access client: create a self-hosted app with an inline `non_identity` policy,
   delete the app, and create/refresh/rotate/delete a service token. About 150 lines plus tests
   in `internal/dns/cloudflare`.
2. Connector permissions: add **Access: Apps and Policies Edit** and **Access: Service Tokens
   Edit** to the `cloudflare` entry in `internal/connectors/registry.go` and to
   `TokenVerifier.VerifyCheck`. About 20 lines. The empty-POST probe that `requireWrite` uses is
   **unverified** against the Access endpoints.
3. Per-computer use case, create and teardown order, plus install-command rendering. About 200
   lines, plus one stored field (the Access app id) next to the existing tunnel row.
4. Teardown gap. `Service.DeleteTunnel` deletes neither the CNAME nor the ingress rule, and
   Cloudflare refuses to delete a tunnel that still has active connections. Teardown must rotate
   and clean up first, then delete, then delete the record. The pieces exist; only the
   sequencing is new.
5. A header-injecting `RoundTripper` for `t3client`, scoped to computer hostnames and wired in
   `server/cmd/services.go`. About 30 lines.
6. Probably a WebSocket keepalive on held T3 connections, because Cloudflare closes idle
   WebSockets. About 15 lines. **Unverified** whether T3 already sends traffic often enough.

**Constraints, not blockers**

- The account needs a Zero Trust organization before any Access API call can work. Onboarding
  means picking a team name and a plan, and entering payment details **even for the Free plan**.
  That is a one-time dashboard step for the owner, not something Nexul can do.
- Limits per account: **50 service tokens**, 500 Access applications, 1,000 tunnels. Use one
  instance-wide service token held by the Nexul server, admitted by every computer's app. A token
  per computer would cap the instance at 50 computers.
- Tunnel `healthy` only means `cloudflared` reaches Cloudflare. The dialog must also probe
  `/.well-known/t3/environment` through the hostname with the Access headers before it
  reports "online".

Ticket 15 does not need reopening.

## 1. Nexul's tunnel code

Sources: `internal/dns/cloudflare/tunnel.go`, `internal/dns/cloudflare/client.go`,
`internal/dns/tunnel.go`, `internal/dns/tunnel_usecase.go`, `internal/dns/provider.go`,
`internal/dns/exposure_usecase.go`, `internal/dns/handler.go`, `internal/dns/mcp.go`.

Provider (`*cloudflare.Client`, the `dns.TunnelProvider` interface):

| Capability | Function | API call | Fit for a laptop |
|---|---|---|---|
| Create a remotely managed tunnel | `CreateTunnel(ctx, name)` | `POST accounts/{acct}/cfd_tunnel` with `{"name", "config_src":"cloudflare"}`; token read from the response | As-is |
| Fetch the connector token | `TunnelToken(ctx, id)` | `GET .../cfd_tunnel/{id}/token` | As-is |
| Ingress to an arbitrary origin | `RouteTunnelHostname(ctx, id, hostname, service)` | read-modify-write `PUT .../configurations`, trailing `http_status:404` catch-all | As-is. `service` is free text, and `http://127.0.0.1:<port>` resolves on the connector's machine |
| List and remove routes | `ListTunnelHostnames`, `RemoveTunnelHostname` | same config endpoint | As-is |
| Connector status | `GetTunnel(ctx, id)` → `Tunnel.Status` | `GET .../cfd_tunnel/{id}` | As-is (status only; `connections[]` is not decoded) |
| Rotate token and kick connectors | `RotateTunnelCredentials(ctx, id)` | `PATCH .../cfd_tunnel/{id}` with a new 32-byte `tunnel_secret`, then `DELETE .../connections` | As-is |
| Delete the tunnel | `DeleteTunnel(ctx, id)` | `DELETE .../cfd_tunnel/{id}`, 404 treated as success | Fails while connectors are connected (see below) |
| Account resolution | `accountID` | zone's `account.id`, then `/accounts` | As-is |

Use cases (`dns.Service`):

- `CreateTunnel` dedupes by name. That is fine for a per-computer name like
  `<slug>-<8 random>`. It encrypts the token (`encryptTunnelToken`) and publishes
  `dns.tunnel_changed`.
- `RouteTunnelHostname` adds the ingress rule, then creates or updates the proxied CNAME
  `<id>.cfargotunnel.com` (`Proxied: true`, TTL 1), and stores `Hostname`, `ZoneID`, `RecordID`,
  and `Service` on the tunnel row. One hostname per tunnel matches one computer per tunnel.
- `TunnelStatus` returns the live `status` from the provider merged into the stored row, token
  blanked. It is already exposed over HTTP (`handler.go`) and MCP.
- `RotateTunnelCredentials` re-encrypts the new token. Its comment says "cloudflared stays up
  until the agent redeploys". On a laptop the user would have to reinstall the service with the
  new token.
- `DeleteTunnel` calls the provider's delete and drops the row. It does **not** delete the CNAME
  (`RecordID`) or the ingress rule. The exposure path (`DeleteExposure`) does remove both, but
  for gateway tunnels only.

What assumes a Docker runner:

- `ProvisionTunnelAgent` deploys `cloudflare/cloudflared:latest` through the `ServiceProvisioner`
  seam (ADR 0017/0037) as a Nexul service on a runner, with `TUNNEL_TOKEN` in env, command
  `tunnel --no-autoupdate --metrics 0.0.0.0:20241 run`, and health at
  `http://localhost:20241/ready`. `gateway_usecase.go` calls it. None of this applies to a
  laptop.
- `exposure_usecase.go` builds origins from container names (`tunnelOrigin(target.Name, port)`)
  on a shared Docker network. That does not apply either.

What changes for a laptop running `cloudflared` as an OS service:

- Skip `ProvisionTunnelAgent`. Decrypt the stored token (the unexported `decryptTunnelToken`
  already exists) and render a per-OS install command (section 4). Readiness comes from the API
  `status`, not from `:20241/ready`.
- Route `http://127.0.0.1:<T3 port>`. T3 Connect uses the literal loopback IP
  (`formatOriginService`, below). The T3 port may differ per machine. Because the tunnel is
  remotely managed, a port change is one `RouteTunnelHostname` call from the server, with no
  reinstall on the laptop. **Unverified:** whether T3 desktop's port is stable across restarts.
  The CLI default is `DEFAULT_PORT = 3773` (`apps/server/src/config.ts`).
- Teardown must work while the laptop is still connected. Cloudflare: `DELETE cfd_tunnel/{id}`
  "Permanently deletes a Cloudflare Tunnel from an account. The tunnel must have no active
  connections."
  ([delete](https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/subresources/cloudflared/methods/delete/)).
  A bare cleanup lets `cloudflared` reconnect with the still-valid token. The working order is
  therefore `RotateTunnelCredentials` (the old token "can no longer establish new connections",
  followed by `DELETE .../connections`), then `DeleteTunnel`, then `DeleteRecord` for the CNAME,
  then delete the Access app. The laptop's service keeps retrying with a dead token until the user
  runs `cloudflared service uninstall`, so the disconnect UI should show that command.
- Order on create: create the Access app **before** the CNAME. A proxied CNAME to a routed tunnel
  is publicly reachable the moment it exists.
- `internal/access` is an existing Nexul domain (instance access, not Cloudflare). Cloudflare
  Access code belongs in `internal/dns/cloudflare` to avoid the name clash.

## 2. Cloudflare Access via API

### Endpoints

All are account-scoped. Paths come from the `cloudflare-go` API listing
([zero_trust/api.md](https://github.com/cloudflare/cloudflare-go/blob/main/zero_trust/api.md))
and the pages linked below.

| Operation | Endpoint |
|---|---|
| Create app | `POST /accounts/{id}/access/apps` |
| Delete app | `DELETE /accounts/{id}/access/apps/{app_id}` |
| App-scoped policy (optional) | `POST /accounts/{id}/access/apps/{app_id}/policies` |
| Reusable policy (optional) | `POST /accounts/{id}/access/policies` ([create policy](https://developers.cloudflare.com/api/resources/zero_trust/subresources/access/subresources/policies/methods/create/)) |
| Create service token | `POST /accounts/{id}/access/service_tokens`. The response holds `client_id` and `client_secret`: "This is the only time Cloudflare Access will display the Client Secret." |
| Refresh (extend) | `POST .../service_tokens/{id}/refresh`, which extends by one year |
| Rotate secret | `POST .../service_tokens/{id}/rotate`. `previous_client_secret_expires_at` sets a grace period; the default expires the old secret immediately (`cloudflare-go` `AccessServiceTokenRotateParams`) |
| Delete token | `DELETE .../service_tokens/{id}` |
| Enable or disable | `PUT` with `"enabled"` |

Source for the token rows:
[service tokens](https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/).

Self-hosted app body, from `cloudflare-go` `AccessApplicationNewParamsBodySelfHostedApplication`
(`zero_trust/accessapplication.go`):

- `type: "self_hosted"` and `domain` (required): "The primary hostname and path secured by
  Access."
- `policies`: "Items can reference existing policies or create new policies exclusive to the
  application. Reusable and inline policies are mutually exclusive." One inline policy per app
  keeps create and delete to a single call each.
- `service_auth_401_redirect: true`: "Returns a 401 status code when the request is blocked by a
  Service Auth policy." Set it. Without it, a blocked non-browser request is sent toward the login
  page, which Go's client follows. `t3client`'s `Harness.do` would then report an HTML body
  instead of a clean auth error. (That the redirect is the default is **inferred** from the
  field's description.)
- `session_duration` and `read_service_tokens_from_header` (a single-header alternative) are
  optional.

The exact response shape of the self-hosted create was not read: the API reference page timed
out twice. The fields above come from the generated Go SDK.

Policy: `decision: "non_identity"`, which the dashboard labels **Service Auth**. Use
`include: [{"service_token": {"token_id": "<id>"}}]` for one token, or
`[{"any_valid_service_token": {}}]` for any token in the account
([create policy](https://developers.cloudflare.com/api/resources/zero_trust/subresources/access/subresources/policies/methods/create/),
[policies](https://developers.cloudflare.com/cloudflare-one/access-controls/policies/): "Service
Auth rules … enforce authentication flows that do not require an identity provider IdP login,
such as service tokens"). Pin the one Nexul token; `any_valid_service_token` would also admit
tokens the owner mints for other purposes.

Service token lifetime: `duration` defaults to "1 year in hours (8760h)", or `"forever"`
(`cloudflare-go` `AccessServiceTokenNewParams`). Use `forever`, or schedule `refresh`.

### Permissions

From the [API token permissions reference](https://developers.cloudflare.com/fundamentals/api/reference/permissions/).
The API reference says "Write"; the dashboard and Nexul's labels say "Edit".

| Permission | Scope | Needed for | Nexul asks today |
|---|---|---|---|
| Zone → Zone: Read | Zone | zone list, account lookup | Yes (`zone_read`) |
| Zone → DNS: Edit | Zone | CNAME | Yes (`dns_edit`) |
| Account → Cloudflare Tunnel: Edit | Account | tunnel CRUD, config, token, connections, status | Yes (`tunnel_edit`) |
| Account → Access: Apps and Policies: Edit | Account | app plus inline policy create/delete ("Grants write access to Cloudflare Access applications and policies") | **No, add** |
| Account → Access: Service Tokens: Edit | Account | service token create/rotate/refresh/delete | **No, add** |
| Account → Access: Organizations, Identity Providers, and Groups: Read | Account | optional: detect that Zero Trust is onboarded | No, optional |

Today's list: `internal/connectors/registry.go` (the `cloudflare` entry's `Checks`), checked by
`internal/connectors/cloudflare/verify.go` (`Verify`, `VerifyCheck`, `requireWrite`). The
Cloudflare OAuth path requests only `offline_access DNS Read DNS Write`
(`internal/connectors/cloudflare/client.go`), and the registry sets `OAuth: nil`, so the API
token is the path that matters. The website guide (`website/src/content/docs/docs/guide/topology-and-dns.md`)
does not list permissions; the connector dialog renders them from the registry.

**Unverified:** `requireWrite` proves a permission by POSTing `{}` and treating 400 as allowed.
For `access/service_tokens`, `name` is required, so `{}` should be rejected with a 400. Test live
before relying on it, so the check never mints a real token.

### Free plan

- Service tokens are the Free-plan machine credential. mTLS "is not included in the Free plan.
  Free customers can use service tokens to authenticate automated systems."
  ([mTLS](https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/mutual-tls-authentication/), via Context7).
- Seats: organizations "can use Access service tokens to allow access to applications without
  consuming seats"
  ([seat management](https://developers.cloudflare.com/cloudflare-one/team-and-resources/users/seat-management/)).
  The Free plan covers up to 50 users (reference architecture page, via Context7).
- Account limits, the same on all plans: Applications 500, Service tokens 50, Reusable policies
  500, `cloudflared` tunnels per account 1,000, active replicas per tunnel 25
  ([account limits](https://developers.cloudflare.com/cloudflare-one/account-limits/), via Context7).
- Prerequisite: a Zero Trust organization. Onboarding asks for a team name and a plan, and "If you
  chose the Zero Trust Free plan, this step is still needed but you will not be charged"
  (payment details)
  ([setup](https://developers.cloudflare.com/cloudflare-one/setup/)). **Unverified:** the exact
  error the Access API returns before onboarding. Surface it as a setup step in the connector
  dialog.
- API rate limit: 1,200 requests per 5 minutes per user or token; exceeding it blocks all calls
  for 5 minutes (the fundamentals API-rate-limits partial, via Context7). A status poll every 2–3
  s during one dialog is well inside that. Do not keep a background poll per computer.

### Client authentication and WebSockets

- Headers on every request: `CF-Access-Client-Id: <id>` and
  `CF-Access-Client-Secret: <secret>`
  ([service tokens](https://developers.cloudflare.com/cloudflare-one/identity/service-tokens/)).
  A Go client without a cookie jar must send them on every request, which a `RoundTripper` does.
- WebSockets: supported on all plans
  ([WebSockets](https://developers.cloudflare.com/network/websockets/)), and "Cloudflare Tunnel
  has full support for Websockets" (Tunnel FAQ, via Context7). Access treats the upgrade as an
  ordinary GET carrying the headers. Cloudflare's Workers docs say Worker-level Access policies
  reject WebSockets with 403 and point to "a hostname-based Access application instead", which is
  the type used here. **Unverified live:** a service-token WebSocket upgrade through a hostname
  app. No page states it outright; it is inferred from the above.
- Idle WebSockets: "Cloudflare will close a WebSocket connection when no data is transmitted in
  either direction for a period of time", and restarts during releases also terminate them
  (same page). The page gives no duration (**unverified**: commonly cited as 100 s). The
  presence keeper holds one T3 WebSocket per computer, and `t3client` sends no pings; it only
  answers a server `Ping` (`internal/t3client/client.go`). Expect periodic drops and redials
  unless T3 traffic keeps the socket busy. A `conn.Ping` every ~30 s in the keeper is the small
  fix, if a live test shows the drops.

## 3. T3 Code through a tunnel

Sources, at `b2b43bef`:
[`infra/relay/src/environments/ManagedEndpointProvider.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/infra/relay/src/environments/ManagedEndpointProvider.ts),
[`apps/server/src/cloud/http.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/cloud/http.ts),
[`apps/server/src/http.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/http.ts),
[`apps/server/src/ws.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/ws.ts),
[`apps/server/src/auth/utils.ts`](https://github.com/pingdotgg/t3code/blob/b2b43bef73447c483ceae486890cb79f01c369cb/apps/server/src/auth/utils.ts).

- **T3 Connect's tunnel setup matches this design.** The relay creates the tunnel with
  `configSrc: "cloudflare"`, then `putConfiguration` with
  `ingress: [{ hostname, service: formatOriginService(origin) }, { service: "http_status:404" }]`,
  where `formatOriginService` returns `http://${host}:${port}` for a loopback host. There is no
  `originRequest` and no `httpHostHeader` rewrite, so T3's server receives the public tunnel
  hostname as `Host` and serves HTTP and WebSockets normally. Nexul's `RouteTunnelHostname`
  writes the same shape (`originRequest: {}`).
- **No Host or Origin allowlist on the API or WebSocket paths.**
  - `http.ts`: CORS (`browserApiCorsLayer`) only matters to browsers. The one
    `isLoopbackHostname` branch redirects static-asset requests to the dev server when `devUrl`
    is set (development only).
  - `ws.ts`: the upgrade reads `wsTicket` and optional `clientSurface`/`clientAppVersion` query
    parameters (`readClientConnectionOrigin`, which is attribution metadata, not the HTTP
    `Origin`). No Host or Origin check was found.
  - `auth/utils.ts`: `isRemoteReachableHost` inspects the **bind** host to decide cookie naming
    and `Secure` flags. It never inspects the request `Host`.
  - `device/DeviceHubProxy.ts` rewrites `Origin` only for its own device-hub proxy.
- **The one forwarded-header check does not affect Nexul.** `cloud/http.ts`
  `cloudLinkProofHandler` rejects requests carrying `x-forwarded-host` or `x-forwarded-proto`
  ("Invalid managed endpoint origin."), and `isAllowedEndpointOrigin` requires a loopback URL.
  That handler serves T3 Connect's link-proof exchange only (`relay:write` scope, called
  locally). Nexul's calls (`/.well-known/t3/environment`, `/oauth/token`,
  `/api/auth/websocket-ticket`, `/ws`) do not touch it.
- A GitHub code search for `rebinding`, `headers.host`, `sec-fetch-site`, and `x-forwarded`
  under `apps/server/src` found only the files above.
- **Nexul's `t3client` can inject the Access headers everywhere.**
  - All HTTP calls go through `Harness.httpClient()` (`internal/t3client/pair.go`: `Version`,
    `exchange`) or through `Connect` → `mintWSTicket(ctx, httpClient, …)`
    (`internal/t3client/client.go`).
  - WebSockets: `websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{HTTPClient: httpClient})`.
    In `coder/websocket` v1.8.15 (`dial.go`), the handshake is `opts.HTTPClient.Do(req)` after
    `req.Header = opts.HTTPHeader.Clone()`, so a wrapping `RoundTripper` sees the upgrade request.
    `HTTPClient.Timeout > 0` bounds only the dial context (`cloneWithDefaults`), not the
    connection. `DialOptions.HTTPHeader` exists too, but `HTTPClient` is already threaded through
    every path.
  - Every `Harness` method passes `h.Options` into `Connect`
    (`internal/t3client/harness.go`). The single wiring point is
    `t3client.NewHarness(t3client.Options{Logger: logger})` in `server/cmd/services.go`.
  - With one instance-wide service token (section 2 limits), one `http.Client` whose
    `RoundTripper` clones the request and adds both headers **only when the host is a computer
    tunnel hostname** is enough, with no per-computer plumbing. Scoping matters because the same
    harness also dials Tailscale or LAN URLs, and the secret must never leave for those.
  - A 403 from Access surfaces as `ErrInvalid` from `Harness.do`, or as a failed dial, which
    `Connect` wraps as `ErrRetryable`. A distinct "Access rejected" error would help the dialog.
- The pairing URL (`https://<hostname>`) passes `validateServerURL`
  (`internal/pairing/model.go`) unchanged.
- **Unverified live:** the full `t3 pair` token exchange and a turn over a tunnel hostname behind
  Access. The code gives no reason it would fail.

## 4. Installing `cloudflared` as a service with a token

Sources: `cloudflare/cloudflared` at `2026.9.1`, `cmd/cloudflared/{linux,macos,windows}_service.go`,
Cloudflare docs via Context7 (`partials/cloudflare-one/tunnel/install-and-run-tunnel.mdx`,
`cloudflared-debian-install.mdx`, `downloads.mdx`, `update-cloudflared.mdx`), and the
[macOS service page](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/as-a-service/macos/).

With a token argument, `service install <TOKEN>` on every OS writes the token to a file in the
config directory and starts the service with `--token-file`, so the token stays out of process
arguments.

| OS | Install `cloudflared` | Install service | Runs as | Updates |
|---|---|---|---|---|
| Linux (Debian/Ubuntu) | `sudo mkdir -p --mode=0755 /usr/share/keyrings && curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \| sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null`; `echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" \| sudo tee /etc/apt/sources.list.d/cloudflared.list`; `sudo apt-get update && sudo apt-get install cloudflared` | `sudo cloudflared service install <TOKEN>` | root; systemd unit `/etc/systemd/system/cloudflared.service`, token in `/etc/cloudflared` | The installer adds `cloudflared-update.timer` (daily `cloudflared update`) unless `--no-update-service`. Docs: automatic updates are not available for package-manager installs, so update with `apt-get install --only-upgrade cloudflared` |
| Linux (RHEL) | `curl -fsSl https://pkg.cloudflare.com/cloudflared.repo \| sudo tee /etc/yum.repos.d/cloudflared.repo`; `sudo yum update && sudo yum install cloudflared` | same | same | same |
| Linux (Arch) | `pacman` (community repo) | same | same | same |
| macOS | `brew install cloudflared` | `sudo cloudflared service install <TOKEN>` (the docs' form) **or** `cloudflared service install <TOKEN>` without sudo | With sudo: LaunchDaemon `/Library/LaunchDaemons/com.cloudflare.cloudflared.plist`, runs at boot. Without: LaunchAgent in `~/Library/LaunchAgents`, "will only run when the user is logged in", token under `~/Library/Application Support/com.cloudflare.cloudflared` | `brew upgrade cloudflared`, then restart the service |
| Windows | MSI or exe from `https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.msi`. `winget install --id Cloudflare.cloudflared` works too: the `microsoft/winget-pkgs` manifest is at 2026.9.1, but Cloudflare's docs do not mention winget | Administrator prompt: `cloudflared.exe service install <TOKEN>` | Windows service via SCM (`mgr.Connect`, `StartAutomatic`). No `ServiceStartName` is set, so it runs as LocalSystem (**inferred** from the SCM default). Token in `C:\ProgramData\cloudflared`, with ACLs restricted by `createTokenFile` | "Instances of cloudflared do not automatically update on Windows"; update with `cloudflared update` and restart |

Uninstall: `cloudflared service uninstall`, with the same privilege it was installed with. Linux
and macOS remove the token file.

User-level versus system-level:

- **Linux:** `cloudflared` offers no user-level mode. It writes to `/etc/systemd/system`, so sudo
  is required. A hand-written `systemd --user` unit running
  `cloudflared tunnel run --token-file …` would avoid root, but `cloudflared` does not generate
  one (**unverified** in practice).
- **macOS:** both modes exist. Since T3 Code itself runs in the user session, the non-sudo
  LaunchAgent is enough and avoids an admin prompt. The trade-off: no tunnel before login, which
  does not matter while T3 isn't running either.
- **Windows:** system service only, and it needs an administrator prompt.
- Any of these reaches `127.0.0.1:<port>` on the same machine, so the account the service runs
  as does not affect reachability.

## 5. Reading connector status

- `GET /accounts/{id}/cfd_tunnel/{tunnel_id}` returns `status`: `inactive` ("tunnel has never
  been run"), `degraded` ("active but in an unhealthy state"), `healthy` ("active and serving
  traffic normally"), and `down` ("cannot serve traffic due to missing edge connections"). It
  also returns `connections[]` (`colo_name`, `client_id`, `opened_at`, `origin_ip`,
  `client_version`) and `conns_active_at` / `conns_inactive_at`
  ([get tunnel](https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/subresources/cloudflared/methods/get/)).
  A healthy tunnel typically shows four connections across two colos (the docs' example in
  `create-remote-tunnel-api.mdx`, via Context7).
- `GET .../cfd_tunnel/{tunnel_id}/connections` lists connectors with `arch`, `version`,
  `run_at`, and `conns[]`, including `origin_ip`
  ([connections](https://developers.cloudflare.com/api/resources/zero_trust/subresources/tunnels/subresources/cloudflared/subresources/connections/methods/get/)).
  That gives the dialog a "connected from <ip>, cloudflared <version>" line. Nexul does not
  decode `connections` today; adding the fields to `tunnelEnvelope` is a few lines.
- Both need only Cloudflare Tunnel Read or Write, which Nexul already holds.
- Nexul already exposes this as `dns.Service.TunnelStatus`, over HTTP and as the MCP tool. The
  dialog can poll it every 2–3 s until `healthy`, then probe
  `GET https://<hostname>/.well-known/t3/environment` with the Access headers
  (`Harness.Version`). Cloudflare: "The tunnel status only reflects the connection between
  cloudflared and the Cloudflare network … A tunnel can appear Healthy while users are unable to
  connect to an application" (tunnel observability, via Context7). The probe is what proves T3 is
  reachable.
- Propagation delays:
  - T3's own code, when re-pointing an existing hostname at a **new** tunnel: "the public
    hostname's route to the new tunnel takes 1-2 minutes to propagate"
    (`cloud/http.ts`, `releaseManagedTunnelOnShutdown`).
  - Status reaching `healthy` after `cloudflared` starts, a fresh CNAME resolving, and a new
    Access app taking effect: Cloudflare's docs give no numbers (**unverified**). Plan the dialog
    for about 2 minutes with visible progress, and avoid resolving the hostname before the CNAME
    exists, so the Nexul host does not cache a negative answer.
