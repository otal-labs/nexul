# Wayfinder map: Phone app and the settings split

**Label:** wayfinder:map

## Destination

A locked design, ready to slice into implementation tickets, in two halves,
web first:

1. **Web.** Settings split by who they affect: **Your settings** behind a
   gear in the sidebar footer (Profile, Appearance, Devices, Tokens,
   T3 pairing), and **Configuration** in the sidebar's Workspace section with
   a "This workspace" group and an admin-only "Whole instance" group. Devices
   lists every signed-in browser, desktop app and phone with sign-out, and
   connects a phone by QR code. The web app officially targets 768px and up.
2. **Phone.** An Android app in `native/` (Expo, React Native) that signs in
   by scanning that QR code, carries a chosen subset of pages restyled for a
   phone, updates over the air from a self-hosted update server, and ships as
   an `.apk` on GitHub releases.

## Notes

- Domains: `internal/auth` (sessions, ADR 0041 is being superseded),
  `web/src/components/settings`, `web/src/components/sidebar`, and the new
  `native/` package.
- Skills every session should consult: `/grilling` and `/domain-modeling`
  for grilling tickets, `design-mode` for any prototype ticket about looks,
  `/research` for research tickets.
- Read `practices/` per `AGENTS.md` before any code, including prototypes.
- Standing preferences from the owner, treat as fixed:
  - Web first, then the phone.
  - Android only. Apple is not on this map; sideloading is the owner's
    normal workflow, no hand-holding needed.
  - The phone signs in **only** by QR code until the hosted offering
    (cloud.nexul.io) exists. Nobody registers from a phone.
  - The server must always be at least as new as the app. An app newer than
    its server refuses to sign in until the server is upgraded; an app older
    than its server keeps working, so the HTTP API stays additive.
  - OTA updates come from a self-hosted update server, set up now, not later.
  - No "v1" or version-tier framing; say "for now" or "first cut".
- Research so far: [React Native best practices](research/react-native-best-practices.md), [self-hosted update server](research/self-hosted-update-server.md).

## Decisions so far

- [01 — Scope and ground rules](issues/01-scope-and-ground-rules.md) — the page picks, the three-scope settings split, stored per-device sessions, QR-only phone sign-in, 768px web, Android `.apk` via GitHub Actions, self-hosted OTA, Uniwind + react-native-reusables.
- [07 — Self-hosted update server requirements](issues/07-self-hosted-update-server.md) — Xprem with Postgres (no Expo account, no EAS CLI) as a Nexul stack at `xprem.nexul.io`; CI publishes with `eoas`; its signing master key cannot be recovered and needs an offline backup.
- [02 — Stored per-device sessions](issues/02-stored-device-sessions.md) — a `sessions` table beside personal access tokens (never merged), sliding 30/90-day expiry, hourly last-active writes, instant sign-out, live events, no MCP tool; a new ADR supersedes 0041.
- [03 — Settings split: routes, placement and old links](issues/03-settings-split-routes.md) — `/settings` is Your settings, `/configuration` the rest, every old link redirects; a normal page; Profile edits name and avatar and links Google/Discord/GitHub sign-ins; mention chips become per workspace.
- [04 — Web on phones after the 768px rule](issues/04-web-on-phones.md) — ADR 0080: `web/` is 768px and up with a dismissible "get the Android app" banner below it, no clean-up of old classes; `website/` stays mobile first.

## Not yet specified

- More than one instance on one phone: a switcher, or one instance per
  install. Hangs on the QR exchange and the session model.
- Workspace switching on the phone, and what a phone does for a user in
  several workspaces.
- Offline behaviour: what a phone shows with no connection, and whether any
  query cache persists across launches.
- How mention chips, attachments and rich blocks render in native chat and
  docs, once the page scope says which of them the phone shows.
- End-to-end testing of the phone against a real instance (Maestro against
  nexul-box or an emulator), once the repo layout and CI are settled.

## Out of scope

- iOS and the App Store. Android first; revisit once the app is ready.
- Signing in on the phone through GitHub, Google or Discord. QR only until
  cloud.nexul.io.
- Registration and onboarding wizards on the phone.
- Phone versions of Memories, Topology, Automations, the project interview,
  Members, Configuration, Project settings, and every wizard.
- Each instance serving its own app bundle. It needs the updater's crash
  rollback switched off (see the research).
- A self-run push relay. Instances call the push service directly.
- Device location from a GeoIP database; Devices shows the IP instead.
