# 07 — Self-hosted update server requirements

**Type:** research
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

What does running Xprem (formerly expo-open-ota) take? How it is installed (container image, binary), what storage it needs (local disk or S3-compatible), how bundles are published to it and whether publishing still needs an Expo account or EAS CLI, how code signing keys are set up, how channels and rollbacks work, and what it costs to run. Compare against Expo's reference server and hot-updater only if Xprem falls short.

## Answer

Use Xprem (repo now `mercuretechnologies/xprem`, v3.2.5, MIT core) in control-plane mode, deployed as a Nexul compose stack at `xprem.nexul.io`. The stack is `ghcr.io/mercuretechnologies/xprem:v3.2.5` plus Postgres 16, with updates on a named local volume. At idle it used about 17 MiB for the server and 46 MiB for Postgres.

Control-plane mode needs no Expo account and no EAS CLI. CI publishes with `npx eoas@3.2.5 publish --nonInteractive`, using a dashboard-issued `EOO_TOKEN`; the stateless mode is the only one that needs an Expo token.

The server generates the RSA signing key and seals it in Postgres under `DB_KEYS_MASTER_KEY_B64`, which cannot be recovered, so back it up offline. The app commits the downloaded certificate and sets it as `updates.codeSigningCertificate`.

Channels, branches, progressive rollouts, rollback to the embedded bundle, and republish are all in the free core. SSO, RBAC, token scoping and automatic rollback are commercial.

Risks:
- One maintainer wrote nearly all of it (bus factor 1).
- Nothing purges old updates yet (issue #263).
- Its docs warn against the fingerprint runtime policy that the stack recommendation picked. That needs a decision before the first build.

Full findings, sources, compose sketch and install checklist: `.scratch/native-app/research/self-hosted-update-server.md`.
