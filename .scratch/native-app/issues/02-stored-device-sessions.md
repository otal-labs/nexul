# 02 — Stored per-device sessions

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

What exactly replaces the stateless 24-hour session token? The table and its columns, how platform and client are recorded (user agent for browsers, reported by the desktop and phone apps), the sliding-expiry rules, how the per-request check folds into the existing user reload, what happens to tokens issued before the change, and which surfaces it reaches: the HTTP routes for listing and signing out devices, whether agents get an MCP tool for it (list only, or none), and the events a sign-out publishes. Ends with the ADR that supersedes ADR 0041.

## Answer

Agreed with the owner 2026-09-28.

- A `sessions` table, separate from personal access tokens but sharing their
  generate, hash and lookup helpers. Columns: user, token hash, client
  (browser, desktop, phone), platform and label ("Linux · Chrome",
  "Android · Pixel 8"), IP, created, last active, expires. Tokens carry a
  `ses_` prefix. Kept apart from personal access tokens because "sign out
  everywhere else" must never reach an agent's or a paired computer's token.
- Browsers and the desktop app (Electron user agent, labelled "Nexul
  desktop") are recognised from the user agent at sign-in with a few lines of
  matching, no new dependency. The phone reports its model in the QR
  exchange.
- Sliding expiry: 30 days since last use for web and desktop, 90 for a
  phone. Last active, IP and expiry are written at most once an hour per
  session.
- The per-request user reload also loads the session; a deleted row fails
  the next request. Web logout deletes the session server-side.
- HTTP: list my sessions (current one flagged), sign one out, sign out
  everywhere else. Own sessions only; no permission bit.
- Events `session.created` and `session.revoked`, same shape as the personal
  access token topics, never carrying the token, pushed live so Devices
  updates when a phone connects.
- No MCP tool: agents act through personal access tokens and must not sign a
  person's devices out. The ADR records this.
- Cutover: stateless tokens stop working on deploy, everyone signs in once.
  All three mint sites (OAuth sign-in, invitation acceptance, dev login)
  create a session. `NEXUL_AUTH_SECRET` stays; it still keys at-rest
  encryption, GitHub webhooks and connection tokens.
- Expired rows for a user are deleted at that user's next sign-in; no
  background job.
- A new ADR, "Sessions are stored per device", supersedes ADR 0041.
