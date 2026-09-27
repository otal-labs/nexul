# 06 — End-to-end install on the test box

**Status:** ready-for-human
**Type:** task
**Blocked by:** 01, 02, 03, 04

Build a release locally, serve it to `nexul-box` with `NEXUL_RELEASE_URL`,
and run the one-liner: server, logs at `/openobserve/`, bundled runner and
automations host all active on one chosen port with port 80 already taken.
Install a second named runner and automations host from the rendered
commands, deploy once, Remove both from the UI and confirm their services and
directories are gone, then `nexul upgrade` and `nexul uninstall --purge`.
