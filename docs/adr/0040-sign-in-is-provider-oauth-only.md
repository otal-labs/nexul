# Sign-in is provider OAuth only, and the provider credentials live in the database

ADR 0061 supersedes the allowlist paragraph below. Provider OAuth and
database-backed provider configuration remain unchanged.

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
