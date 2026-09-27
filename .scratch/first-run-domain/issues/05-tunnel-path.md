# 05 — Tunnel path in setup

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 04

## What to build

The tunnel choice opens the Cloudflare API token ticker dialog (`ManualConnectorDialog`, required checks must pass) under the setup pass, then the existing tunnel stepper (account, machine, deploy, hostname, route checks). When the checks pass, the hostname goes to the shared finish from 04. The bundled `instance` machine and the seeded project are preselected.

## Acceptance criteria

- [ ] A fresh instance reaches "live at https://<host>" through a tunnel with no sign-in
- [ ] A token missing a required permission cannot continue
- [ ] The owner wizard later shows Cloudflare as connected

## Read first

`practices/react-guide.md`, `practices/testing.md`, `web/src/components/settings/ManualConnectorDialog.tsx`, `web/src/components/dns/*`.
