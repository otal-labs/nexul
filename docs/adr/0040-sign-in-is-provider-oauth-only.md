# Sign-in is provider OAuth only, and the provider credentials live in the database

ADR 0061 supersedes the allowlist paragraph below. Provider OAuth and
database-backed provider configuration remain unchanged.

Amended: a user holds one or more sign-in identities instead of being keyed
by a single provider (the amendment at the end).

Nexul has no username/password and stores no passwords: a person signs
in through GitHub, or through the optional Google/Discord providers an owner
turns on later. Nothing to hash, nothing to reset, no credential store to
breach — and the same GitHub App that authenticates also backs the git
provider work, so the account a developer already has is the account they use.

The App's client ID and secret (and each optional provider's) are encrypted
columns on the single settings row, not env vars, so a fresh instance is
bootstrapped from a browser at `/setup` and can be re-pointed at a different
App without a redeploy. The cost is a chicken-and-egg first run — the very
first screen of a new instance is an unauthenticated form that can configure
login — so `Bootstrap` verifies the credentials live against GitHub before
storing them and stops being served once a user row exists.

Who may sign in is one instance-wide allowlist keyed by whatever login the
provider yields: a GitHub username or a verified email. One list rather than
one per provider, because a GitHub login can never contain `@`, so the two
kinds can never collide. An unverified email is refused at sign-in.

Decided: 2026-07-25 (GitHub OAuth); optional providers and DB-backed
credentials 2026-09-03.

## Amendment: one user, many sign-in identities

The user row used to carry its provider and provider user id, so a person
who signed in with GitHub could never also sign in with Google without a
second account. Identities now live in their own rows, one per provider per
user, and sign-in resolves the user through them. A signed-in person links
another provider from their Profile by running that provider's OAuth in
link mode: the state carries the signed-in user, signed with the instance
secret, so the public callback attaches the account to them and never to
whoever a cookie names. No admission step: they are already in. Unlinking is
the way back out and is refused for the last identity, so nobody can lock
themselves out; an identity attached to another user is refused without
changing either user. The user's login, name and avatar follow the identity
they were created with, so a linked account never renames them, and the
identity row itself syncs on every sign-in. Agents get no tool for this:
linking is an interactive browser flow, and unlinking a person's sign-in is
a session-class act (ADR 0083). Decided 2026-09-28.
