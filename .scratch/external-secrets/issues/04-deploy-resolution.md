# 04 — When a runner resolves references, and what a failed resolution does to a deploy

**Status:** resolved
**Type:** grilling
**Blocked by:** 01, 02, 07

## Question

When does the runner resolve: every deploy with no cache, or cached for a time? If a reference cannot be
resolved (missing item, no access, 1Password down), does the deploy fail before build, naming the
variable but never a value, and how does that show in the deploy history and the trail? Resolved values
land in the compose `.env` (mode 0600) and in `-e` arguments today (`internal/runner/executor.go`,
`writeComposeEnv`): is that acceptable for referenced values, or do they need another route into the
containers?

## Answer

Decided 2026-11-12.

- **What the server sends:** the stack's env unchanged, with references as plain strings. The runner
  recognises them with the same parser the server's save check uses (ticket 02).
- **When:** every deploy, with no cache. All of a deploy's references go in one `ResolveAll`, which
  costs one read and about 0.7 s (ticket 07). A value rotated in 1Password takes effect on the next
  deploy, retry or rollback.
- **Runner client:** one 1Password client per runner process, created on the first deploy that needs
  it and kept, so its start-up (about 2.6 s of CPU) is paid once.
- **Order:** the runner resolves first, before pull, build or `compose up`.
- **Failure:** any reference that fails (missing item or field, a vault the token can't reach,
  1Password down, or no token on the runner) fails the deploy and leaves the running containers
  untouched. The error lists each failing variable and its reason (`DB_PASSWORD: field not found`),
  never a reference or a value, since deploy history readers may lack `stacks:write`. The existing
  Retry on a failed deploy is the way to try again; nothing retries automatically.
- **Delivery:** resolved values go into the 0600 compose `.env` and the run stack's `-e` flags exactly
  as literals do (`writeComposeEnv`). The runner keeps nothing after the deploy.
- **Rollback and preview deploys:** a rollback redeploys with the stack's current env (deploys keep no
  copy of it), so it uses each reference's value now, as it already does with literals. Branch rule
  overrides follow the same rules.
- **Deploy log:** one line with the number of references resolved and the time taken, never which
  ones.

Whether Nexul checks before dispatching that the runner can resolve at all is ticket 03.
