# 07 — Merge → deploy automation shape

**Type:** grilling
**Status:** resolved
**Blocked by:** 06

## Question

The trigger side: merge events already flow through the gitprovider
webhook receiver (GP2 normalizes all GitHub events onto the bus), and
DP6's stance is "nothing auto-deploys out of the box" — deploy automations
ship as opt-in templates.

Decide:

- Is branch-driven deploy an **automation rule** (automations domain
  trigger/action vocabulary, opt-in template per DP6) or a first-class
  property of the branch-deployment mapping from 05/06 — and if the
  latter, does it still ride the automations engine underneath?
- Which normalized event triggers it (push to branch vs PR merged — they
  differ for `dev`/`main` vs `feature/*`)?
- Build source: DP1 services can build from a ref via runners (RN6) —
  confirm the branch deployment builds the merged ref, and what happens
  when the service has no build source (pre-built image services).
- Failure surfacing: where does a failed preview deploy show up (ticket
  dev panel GP4? chat? deploy history only)?

Close by updating `automationsDomains.md` / `deployDomains.md` DP6 as
needed.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review.

- **Trigger: the normalized push event, uniformly.** "Merge into X" is a
  push to X from the provider's point of view — one event covers
  `dev`/`main` and `feature/*` alike. PR-merged is not a second trigger
  path.
- **Deploy domain owns the consumer**, not the automations engine: branch
  deploy rules are service configuration (like health checks), evaluated
  by a typed deploy-domain consumer of the normalized push event
  (GP2's bus). Automations keeps its own separate ability to trigger
  deploys; user-authored rules don't duplicate this mechanism. DP6's
  "nothing auto-deploys out of the box" holds — a rule must be explicitly
  configured.
- **Build source:** the branch deployment builds the pushed ref via the
  base service's build source (RN6). A base service with no build source
  (pre-built image only) rejects wildcard rules at configuration time —
  there is nothing branch-specific to deploy. Exact rules on image-only
  services redeploy the configured image.
- **Failure surfacing:** deploy history + the existing deploy status
  events (board/ticket dev-panel already consume these); nothing new in
  v1. Chat notification is fog (automation template material).
