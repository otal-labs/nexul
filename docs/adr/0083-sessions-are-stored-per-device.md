# Sessions are stored per device

Supersedes ADR 0041. A session used to be a stateless HMAC token with a 24h
life and no server-side record, so there was nothing to list and nothing to
revoke short of rotating the secret. A phone app that stays signed in for
months, and a Devices page that shows what is signed in and lets one device
be signed out, both need a row.

Decision: a `sessions` table holds one row per signed-in device: user, token
hash, client (browser, desktop, phone), platform, label, IP, created, last
active, expires. Tokens carry a `ses_` prefix and share the generate, hash
and lookup helpers of personal access tokens, but live in their own table:
"sign out everywhere else" must never reach an agent's or a paired computer's
token. Every sign-in path (provider OAuth, invitation acceptance, dev login)
creates a session from the request, with browsers and the Electron desktop
app recognised from the user agent by a few lines of matching and a phone
reporting its own model. The per-request auth reload loads the session with
the user and rejects a missing or expired row, so deleting the row signs the
device out on its next request or WebSocket dial. Expiry slides: 30 days
since last use for a browser or the desktop app, 90 for a phone, with last
active, IP and expiry written at most once an hour per session so a busy tab
is not a write per request. A user's expired rows are deleted at their next
sign-in; there is no background job.

The HTTP gateway lists a user's own sessions with the current one flagged,
signs one out, signs out everywhere else, and signs out the current session,
which the web app's Sign out calls before clearing its local state. No
permission bit: a session is the user's own. `session.created` and
`session.revoked` are catalog events written through the outbox and pushed
live, never carrying the token, so a Devices page updates the moment a phone
connects.

There is no MCP tool. Agents act through personal access tokens, and a tool
that lists or signs out a person's devices would let an agent sign a person
out; the routes require a session and refuse a personal access token for the
same reason.

`NEXUL_AUTH_SECRET` stays: it still keys at-rest encryption, GitHub webhook
signatures, setup passes and connection tokens. Tokens issued before this
change stop working on deploy and everyone signs in once; there was no
production data to migrate. The cost is one indexed read per request against
the single-writer SQLite file, which ADR 0041 avoided, accepted because that
read is what makes a device revocable at all.
