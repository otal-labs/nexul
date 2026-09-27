# First run sets up the domain before GitHub

**Status:** ready-for-agent

## Problem

A fresh install tells the owner to open `http://<ip>:<port>`, and the first
screen asks for a GitHub App. A GitHub App's callback must be the instance's
final https address, which doesn't exist yet, so the owner either registers
`localhost` (wrong for a server) or leaves to set up a domain and HTTPS by hand
with no guidance. The DNS stepper that could do it only appears after
sign-in, at the end of the owner wizard.

Three facts from the current code shape the fix:

- OAuth has to start on the final origin. The state cookie is set on the host
  that starts `/auth/github`, the callback comes back to the instance URL, and
  the cookie is Secure when that URL is https. Starting from `IP:port` fails
  with "oauth state mismatch". Sessions are per-origin (localStorage), so
  nothing done on `IP:port` carries over as a login either.
- Saving the Cloudflare token and every `/api/dns/*` call require a signed-in
  user, and before any user exists `POST /api/auth/bootstrap` is open to
  whoever reaches the port first.
- The wizard's reverse-proxy path deploys nothing (an empty stack), and the
  Traefik gateway has no ACME and nothing forwarding to the Nexul server,
  which is a host service, not a container.

## The flow

1. `nexul install` installs on port **5123** by default and prints a
   **setup code**. The summary says the domain, HTTPS and ports 80/443 are set
   up from the website, not by the installer.
2. The owner opens `http://<ip>:5123`. The only screen is "Enter the setup
   code". A correct code unlocks a **setup pass** for this browser.
3. **Domain step**, no skip. The owner picks how traffic reaches Nexul:
   - **Cloudflare tunnel**: Cloudflare API token in the ticker dialog (won't
     continue until the required checks pass), then the existing tunnel
     stepper (account, machine, hostname, route checks). Cloudflare issues
     the certificate.
   - **Reverse proxy**: type the domain, point an A/AAAA record at the shown
     public address (Nexul creates it when a Cloudflare token is connected),
     Nexul waits until it resolves here, deploys Traefik on 80/443 with
     Let's Encrypt, and forwards the domain to Nexul.
   - **I already have HTTPS**: type the https address that already reaches
     this server through the owner's own proxy; Nexul checks it answers.
   Every path ends with the same check (the domain answers Nexul's own
   bootstrap-status over HTTPS) and then stores it as the instance URL.
4. "Nexul is live at https://<domain>. Continue there." The link carries the
   setup code in the URL fragment (`/setup#code=…`), so it never reaches
   logs or Cloudflare.
5. On the domain: the GitHub App bootstrap (instance URL already filled and
   fixed), then GitHub sign-in starting from the domain, then the owner wizard
   without its DNS step, then connectors as today.

Desktop installs (macOS, Windows) are for trying Nexul on one computer, where
`http://localhost:5123` is a valid GitHub callback. The installer marks them
local, and first run there skips the domain step and uses localhost.

## Decisions

- **Setup code** (owner, 2026-09-27): the server writes a fresh code to
  `<data>/enroll/setup` on every boot until the first user exists, like the
  bundled runner's enrollment code (`hostcred.MintCode`, hash stored, file
  0600), and the installer prints it. `nexul status` prints it again while it
  is still valid, so a lost terminal is not a dead end.
- **Setup pass**: `POST /api/setup/unlock {code}` returns a short-lived bearer
  (one hour) that `RequireAuth` accepts only while no user exists and only on
  an allowlist: the setup routes, the Cloudflare connector's manual
  verify/save, `/api/dns/*`, `/api/machines`, `/api/projects`, deploys the DNS
  step reads, and `/api/auth/bootstrap*`. Bootstrap now requires the pass too,
  which closes today's first-visitor-wins gap. Once a user exists every setup
  pass stops working.
- **Default port 5123** (owner, 2026-09-27): the reverse proxy needs 80/443,
  so Nexul never takes them. The installer still asks, and an existing
  install keeps its port.
- **Paths** (owner, 2026-09-27): tunnel, reverse proxy, "I already have
  HTTPS". The bare A record without HTTPS is dropped from first run; it stays
  available on the DNS page after setup.
- **No skip** (owner, 2026-09-27): the one deliberate gate, like machine setup,
  because nothing after it works without a domain.
- **Cloudflare credentials stay a pasted API token** (owner, 2026-09-27). An
  OAuth callback cannot exist before the domain does. Pre-release concern 06
  is narrowed accordingly.
- **Instance URL written by setup**: a setup-pass-only
  `PUT /api/setup/instance-url`, gated on the shared HTTPS check, so the GitHub
  bootstrap no longer takes the URL as a field on server installs.

## Out of scope

- Moving an instance to a new domain after setup (the owner's existing
  `PUT /api/auth/settings` covers it).
- Cloudflare OAuth.
- Certificates for exposed stacks beyond what the proxy path needs for the
  instance itself.

## Tickets

01 install on 5123 · 02 setup code · 03 setup pass · 04 domain step and
handoff · 05 tunnel path · 06 "I already have HTTPS" path · 07 research:
Traefik to a host service with Let's Encrypt · 08 reverse-proxy path ·
09 bootstrap on the domain · 10 desktop local mode · 11 docs, ADR, CONTEXT

## Contract

The shape every ticket builds against, fixed before the work split.

**Setup code** — `nxs_` + `hostcred.MintCode`'s random part; the server
writes it to `<data>/enroll/setup` (0600) on every boot while no user exists
and deletes the file once one does. Only its sha256 is stored, valid 24h.

**Status** — `GET /api/auth/bootstrap-status` (public) gains:
`setup_open` (no user exists yet), `instance_url` (stored, may be empty),
`local` (desktop install). Existing fields stay.

**Unlock** — `POST /api/setup/unlock {"code"}` (public):
- 200 `{"token","expires_at"}`: the setup pass, a bearer valid one hour.
- 400 `invalid_code` (wrong or expired), 429 after 10 failures in 10 minutes
  from one address, 409 `setup_done` once a user exists.

**Setup pass** — sent as `Authorization: Bearer <token>`. `RequireAuth`
accepts it only while no user exists, with the caller's identity set to the
user id `setup`, and only on:
- `/api/setup/*`, `/api/auth/bootstrap`, `/api/auth/bootstrap/verify`
- `/api/connectors/cloudflare/manual`, `/api/connectors/cloudflare/manual/verify`,
  `GET /api/connectors`
- `/api/dns/*`, `GET /api/machines`, `GET /api/projects`,
  `GET /api/services/{id}/deploys`, `GET /api/deploys/{id}/logs`
Anything else with a pass is 401.

**Instance URL** — `PUT /api/setup/instance-url {"url"}` (pass only, no user
may exist): refuses `http://` unless the install is local, runs the existing
`VerifyInstanceURL` check, stores it, returns `{"instance_url"}`.

**Bootstrap** — `POST /api/auth/bootstrap` and `/bootstrap/verify` require
the pass. When an instance URL is stored, the request's `instance_url` is
optional and must match it if given.

**Local install** — the installer writes `NEXUL_LOCAL=1` into the server's
env on macOS and Windows only (optional; unset means a server install).

**Reverse proxy** —
- `GET /api/setup/public-address` (pass or signed-in owner): `{"ipv4","ipv6"}`,
  the server's public addresses as the internet sees them.
- `GET /api/dns/resolve?host=<domain>`: `{"addresses":[...]}`, what the
  domain resolves to right now.
- `POST /api/dns/instance-proxy {"domain","email"}` (`email` optional):
  deploys the gateway Traefik on 80/443 with Let's Encrypt, routing the domain
  to the Nexul server on the host; returns `{"service_id"}` so the page can
  watch the deploy like the tunnel's. Retry-safe.
