# 02 — Update GitHub App homepage + callback URLs

**Status:** ready-for-human

**Blocked by:** None — but only actionable once a real domain exists.

**What to build:** The GitHub OAuth App's **Homepage URL** and **Authorization
callback URL** are placeholders, made up at creation time because no real app
existed yet — homepage a placeholder domain, callback
`http://localhost:8080/auth/callback`. Both must be updated to the real domain
once it exists. The redirect_uri Nexul sends is derived from the
instance URL set via the bootstrap screen / owner wizard (`internal/auth`'s
`Settings.InstanceURL` + `/auth/callback`, no env var involved since T5), so
updating the instance URL there is what needs to match the App setting exactly.
GitHub requires an exact match, so OAuth will fail in prod until it does.

The App being edited is the development OAuth App; its local-dev callbacks are
`/auth/callback` and `/auth/connectors/github/callback` under both
`http://localhost:5173` (Vite) and `http://localhost` (plain compose). How to
create and configure an App from scratch is in the docs site's GitHub App
guide, `website/src/content/docs/docs/guide/github-app.md`.

- [ ] Homepage URL updated to the real domain
- [ ] Authorization callback URL updated to the real domain
- [ ] Instance URL (bootstrap screen / owner wizard) matches the App setting exactly
- [ ] OAuth login verified end to end against the non-localhost host

## Surface when

- Implementing or testing the auth / OAuth callback flow against any
  non-localhost host.
- Setting up a real domain, reverse proxy, TLS, or a staging/prod host.
- Packaging or release work starts.
- The owner mentions a domain, a URL, or "deploy to a server".
