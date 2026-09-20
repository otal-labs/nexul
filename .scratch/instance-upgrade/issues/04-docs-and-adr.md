# 04 — Docs: upgrade guide, ADR, glossary

**Status:** resolved
**Type:** task
**Blocked by:** 01, 02, 03 (describe what shipped).

## Build

- `website/src/content/docs/docs/guide/upgrade.md`: a "From the web UI"
  section first (where the button is, what happens, the helper container,
  `docker logs nexul-upgrade` for failures, the limits: images only, public
  registry, compose installs only), then the existing manual path.
- ADR `0054-instance-upgrade-runs-in-a-helper-container.md`: the decision
  (helper container started by the instance runner, resolved by the server
  on boot), the rejected host-runner and detached-shell shapes, the
  trade-off (no live log after the restart, one stopped helper container
  per host).
- `CONTEXT.md`: glossary entry for **Instance upgrade** and **Upgrade helper**.
- `website/src/content/docs/docs/guide/mcp-server.md`: list the two new tools if
  the page enumerates tools.

## Answer

Done on the feature branch in commit 400e54e: upgrade guide "From the web UI" section, ADR 0054, CONTEXT.md entries for Instance upgrade and Upgrade helper, MCP tool list.
