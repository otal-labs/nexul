# 06 — Branch deployment lifecycle

**Type:** grilling
**Status:** resolved
**Blocked by:** 05

## Question

Starting position (settled at charting): a branch deployment is a **clone
of the base service definition** — same env vars verbatim (hence same DB),
different container + hostname, joining a chosen docker network. Per-branch
env overrides are fog, not v1.

Decide the lifecycle:

- **Hostname derivation:** `feature/discord-integration` →
  `feature-discord-integration.example.com` — the sanitization rules, the
  base domain source (the network's exit node zone?), collision handling.
- **Container naming:** DP1 requires deterministic, user-chosen,
  per-server-unique names — branch deployments must derive a suffix
  without breaking that ("random names are an explicit non-goal").
- **Creation:** first merge to a matching branch creates the clone — or
  does the owner pre-approve which branches may spawn deployments?
- **Teardown:** branch deleted / PR merged → what happens (auto-remove,
  grace period, manual)? What gets cleaned: container, DNS record, tunnel
  ingress route.
- **Interplay with DP5** (one active deploy per service): is each branch
  deployment its own service record, so the rule applies per-branch?
- **Provenance (DP3):** the deploy records the branch/PR that spawned it.

Close by extending `deployDomains.md` with the lifecycle.

## Answer

Resolved by owner delegation (2026-08-27), flagged for review.

- **A branch deployment is a derived service record** cloned from the
  base: same image/build source (built at the merged ref), env vars
  copied verbatim (shared DB — settled at charting), network from the
  rule, marked `derived_from = <base service>` and keyed by branch.
  DP5 (one active deploy) applies per branch deployment, since each is
  its own service record. Provenance (DP3) records the triggering
  branch/push.
- **Branch slug:** lowercase; `/` and any non-`[a-z0-9-]` → `-`; collapse
  runs; trim leading/trailing `-`; cap so `<base>-<slug>` fits Docker
  names and `<slug>` fits a 63-char DNS label. Collision after slugging →
  reject with a clear error (deterministic names, DP1's non-goal on
  random suffixes holds).
- **Container name** `<base>-<slug>`; **hostname** from the rule's
  template with `{branch}` = slug (e.g. `{branch}.example.com` →
  `feature-discord-integration.example.com`), exposed through the rule's
  network gateway (ticket 01).
- **Creation:** first push to a branch matching a rule creates the clone
  and deploys it. Configuring the rule *is* the owner's opt-in — no
  per-branch approval step.
- **Teardown:** branch deleted → immediate auto-teardown (container,
  DNS record, ingress route/labels). Manual teardown is a first-class
  action. Grace periods and PR-merged-triggered teardown are fog
  (`git.branch_deleted` is the one unambiguous signal; PR-merged still
  leaves the branch alive).
- **Listing:** branch deployments list under their base service, not as
  top-level services.
