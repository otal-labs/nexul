# T3 Code's own interface through Nexul, for computers that only reach out on 443

**Status:** needs-info

## Problem

A computer behind a proxy that lets out only TCP 443 cannot use T3 Connect. T3's tunnel is `cloudflared`, which
connects to Cloudflare's edge on port 7844 over both TCP and UDP, and both of its protocols use that port. T3 says so
itself and offers no 443 fallback (below). That part belongs to T3.

Nexul's personal runner on that computer already works over 443, because it only opens WebSockets to the Nexul
instance. So the question is whether a person can open T3 Code's own interface (its web page, desktop app or phone
app) on that computer through Nexul. Today they cannot. They can do what Nexul's own surfaces expose (chat, plays,
setup, project picker, clone, MCP), nothing more.

## What exists

### Reaching T3 Code through the runner (ADR 0146)

- The server sends `harness_dial {id}` on the runner's control WebSocket (`internal/runner/streams.go`,
  `Handler.DialComputer`). The runner dials T3 Code on loopback, at the port in T3's runtime file, and opens
  `/api/runners/streams/{id}` back to the server with its credential. Both ends wrap the socket as a `net.Conn` and
  splice bytes. One WebSocket per relayed connection, at most 32 per runner (`maxStreams`), 10 s to claim a stream id.
- The only consumer of `DialComputer` is `harnessHTTPClient` in `server/cmd/wire_computer_streams.go`. It is the
  `http.Client` of the two T3 harness clients (`internal/t3client`, `internal/t3clientv2`), and it refuses any address
  that does not end in `.nexul-computer.invalid` (`pairing.ComputerHostSuffix`, `internal/pairing/model.go`). It
  rewrites `Host` to `127.0.0.1` for T3 Code (`asLoopback`).
- Nothing else uses the stream: no HTTP route of the server forwards a browser's request to a computer's T3 Code.
  `server/cmd/logs_proxy.go` is the only reverse proxy and it fronts OpenObserve. `rg DialComputer` outside tests finds
  the pairing seam and the harness client, nothing else.
- The server holds one T3 bearer session per computer (minted from a one-time token the runner makes with
  `t3 auth pairing create --ttl 5m --label Nexul`, `internal/runner/t3code.go`), re-paired before its 30 days end.
  `FrameT3PairTokenRequest` already lets the server ask the runner for a fresh token at any time.

### What a person gets through it today

Everything is the server's harness client acting for a person, through the `harness.Client` interface
(`internal/harness/harness.go`): list projects and providers, clone a project, start, watch, interrupt, answer a
question in, and settle a turn. Surfaces over that: the chat page and ticket threads in the web and phone apps, plays
and auto plays, setup turns, `@Agent` mentions, the project and model pickers, and the computer's facts. Separately,
the agent on the computer reaches Nexul's MCP server with the computer's token, an outbound HTTPS call, so MCP works
over 443 too. Container and runner logs stream through the same runner connection.

What none of it carries: T3 Code's UI itself. No page, terminal, diff view, file browser, browser tab, settings or
authorized-clients screen of T3 Code is reachable through Nexul, and there is no link from a thread to T3. A person
who wants T3's own UI uses T3's routes (below).

### In T3 v0.1.1 (read at tag `v0.1.1`)

- `apps/server/src/cloud/ManagedEndpointRuntime.ts` spawns `cloudflared tunnel --no-autoupdate ... run`, adding
  `--protocol http2` only after `QUIC_STALLS_BEFORE_HTTP2` (30) QUIC stalls in 5 minutes (or when the user set
  `TUNNEL_TRANSPORT_PROTOCOL`). HTTP/2 is TCP, but to the same edge port.
- `apps/server/src/cloud/ConnectReachability.ts` states why: `cloudflared` finds the edge by an SRV record naming
  `region1.v2.argotunnel.com` and `region2.v2.argotunnel.com` on port 7844, "QUIC and HTTP/2 both use that port".
  When a TCP probe to 7844 times out it reports `edge_unreachable` with "Allow outbound TCP and UDP port 7844".
- `docs/user/t3-connect-troubleshooting.md` ("Port 7844 is blocked on the host") gives the only workaround: a direct
  route, for example the host's Tailscale address, as in `docs/user/remote-access.md` ("Reach one machine several
  ways").
- T3 Connect has no other transport. `docs/internals/t3-connect.md`: the relay Worker brokers links and bootstrap
  credentials and clients then talk through the environment's tunnel hostname; the relay does not proxy sessions
  (except automation webhooks). The other T3 routes are all inbound to the host: LAN or VPN pairing (`t3 serve --host`),
  Tailscale HTTPS (`--tailscale-serve`), desktop-managed SSH, and the hosted app at app.t3.codes, which needs an HTTPS
  endpoint it can reach directly.
- `t3 serve` serves T3's web UI and its API on the same port (static dir plus the RPC and HTTP routes). A session is
  a browser cookie (named from the host, port and state dir; `auth/utils.ts` `resolveSessionCookieName`), a bearer, or
  a DPoP token (`auth/EnvironmentAuthPolicy.ts`), bootstrapped by a one-time pairing token.

## Options

A through D differ in who opens T3's UI and where. All reuse the existing stream; none needs a new runner frame
except the minting call, which exists.

### A. Reverse-proxy T3's HTTP and WebSocket under a Nexul route, behind Nexul sign-in

A route such as `/computers/{id}/t3/*` on the server accepts the browser's request and WebSocket upgrades, checks the
person's Nexul session and computer rule, dials the runner stream, and forwards.

- **Gains:** one URL, works in any browser that can already reach Nexul, nothing to install, no extra firewall rule.
- **Costs and risks:**
  - T3's web app is built for the origin root and its cookie name derives from host and port, so serving it under a
    path prefix needs either a path-rewriting proxy that rewrites HTML and asset URLs, or a dedicated subdomain per
    computer (a wildcard DNS name and TLS cert on the instance, which Nexul's install does not have by default).
    Unverified: whether the T3 web bundle tolerates a base path at all; needs a spike.
  - Whose T3 session does the browser hold? If the proxy injects the server's stored Nexul bearer, every person the
    computer rule lets in acts through the owner's one session, labelled `Nexul`, with no separate revocation and no
    per-person trail in T3. Better: on each open, ask the runner for a fresh one-time token (`FrameT3PairTokenRequest`),
    exchange it as that person's own session, and put that in a cookie scoped to the computer's origin.
  - T3's session can run agents and terminals as the computer's user. Opening it is `Run commands` in ADR 0148's
    terms, not `See` or `Run agents`. A new computer action, or a reuse of `Run commands`, is a decision.
  - The runner enforces loopback and the T3 port; the server proxy adds a browser-facing attack surface on the instance
    (cookie scoping, CSRF between T3's origin and Nexul's, WebSocket origin checks, header stripping). T3's pages would
    run script on an origin that must not be able to read Nexul's own session cookie, which again points to a
    separate origin per computer.
  - Load: a T3 page keeps a long-lived WebSocket plus parallel asset requests, each a new stream with a handshake,
    against a cap of 32 streams per runner that agent work shares. The cap and the 10 s claim window need a look.
- **Effort:** large. Server proxy with upgrade support and tests, per-computer origin and certificates, the session
  exchange, a permission and its guard tests (`practices/testing.md` section 10 style), audit rows for every open
  (every non-GET route is audited), user guide, ADR. Several weeks, with the base-path and origin questions able to
  grow it.

### B. A local port forward from the person's own device (no T3 UI on Nexul's origin)

A `nexul` command on the person's laptop (for example `nexul connect <computer>`) opens `127.0.0.1:<port>` and relays
each connection over a new authenticated WebSocket on the server, which splices it onto the runner stream. The person
then adds that loopback URL in T3's desktop app, or opens it in a browser, with a pairing token the runner mints for
them (`--label` naming the person).

- **Gains:** T3's UI is served from a loopback origin T3 already treats as local, so no path rewriting, no cookie or
  origin problems, and T3 keeps its own session per client, separately revocable. The Nexul server only forwards bytes
  after authenticating the person, exactly as `DialComputer` does now.
- **Costs:** the person installs and runs a command on each device; it does not help a phone running T3's mobile app
  (no loopback to expose), and some managed laptops cannot run it. The server still needs a permission and an audit row
  for every forwarded session, and a forwarding endpoint that streams for as long as T3's UI is open.
- **Effort:** medium. One server WebSocket endpoint over the existing `DialComputer`, the permission, the CLI command
  with a pairing step, tests, guide. Days to a couple of weeks; the CLI is the new surface.

### C. Give T3 clients a Nexul URL to pair with (a public endpoint)

Expose, per computer, an HTTPS endpoint on the instance that T3's own apps accept as an environment URL or direct
route (app.t3.codes needs HTTPS and reaches the endpoint directly). It is option A without Nexul's sign-in in front,
because T3's apps cannot send Nexul credentials; the pairing token in T3's own flow becomes the only gate.

- **Gains:** T3's desktop and phone apps and app.t3.codes work unchanged.
- **Costs:** puts T3's whole API, which runs shell commands as the person, on the public internet at a guessable
  hostname behind a short-lived pairing token alone, which is what T3's own docs warn against doing casually. The owner
  gives up Nexul's access rules at the door. Not recommended without a reason A and B cannot meet.
- **Effort:** as A, plus an abuse surface.

### D. Build the missing pieces natively in Nexul instead of passing T3's UI through

If the real gaps are specific (see a diff, read a file, interrupt and answer, open a terminal) add them to Nexul's own
surfaces through the harness client, the way chat is built. This stays inside Nexul's own auth and design language, and
needs no origin work.

- **Gains:** each piece is small, permissioned and tested like the rest; works on the phone app.
- **Costs:** never reaches parity with T3's UI, and each piece needs T3's protocol to expose it. A terminal is the same
  risk as option A's session, scoped to Nexul's own permission.
- **Effort:** per piece, small to medium. Needs the owner to name which pieces are missed.

### Not an option: making T3 Connect work over 443

That is Cloudflare's edge port, chosen by T3. Nexul cannot change it. Raising it upstream with T3 (a 443 fallback for
the tunnel, or a way to bring the tunnel through the host's HTTP proxy) is the only fix on that side.

## Recommendation

B first if the want is T3's desktop UI from the owner's own laptop: it avoids the origin and session problems that make
A large, keeps T3's per-client sessions, and reuses what exists. D for anything the owner wants on the phone. A only if
the owner needs a no-install browser path and accepts a per-computer origin. Not C.

## Open questions for the owner

1. What do you need from T3's own UI that Nexul's chat does not cover: reading diffs, a terminal, browsing files, the
   browser tab, or just "the real thing" on a screen? The answer chooses between B and D.
2. From which devices: your own laptop, a phone, or someone you shared the computer with? Option B covers only
   devices that run a command.
3. Who may open T3's UI through Nexul: the computer's owner only, or anyone with `Run commands` (a shared person gets
   a shell as the owner's user)? Should it be its own permission?
4. Is a wildcard subdomain and certificate per computer acceptable on an instance, if A is picked? Today installs
   do not have one.
5. Should the connection to the runner be used by a person's T3 UI for long sessions at all, given the 32-stream cap
   per runner that agent work shares?
6. Do you want us to ask T3 for a 443 or HTTP-proxy path for the tunnel in parallel, since it would help every user
   behind such a network and keep Nexul out of it?
7. Does anyone on that network need the Nexul app itself to go through an HTTP proxy (`HTTPS_PROXY`)? The runner dials
   with Go's default HTTP client, which honors the standard variables, but the install writes none into the service and
   no test covers it, so it is unverified.
