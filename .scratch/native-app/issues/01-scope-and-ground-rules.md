# 01 — Scope and ground rules

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Where does this effort end, which pages come back on the phone, how do
settings split, how does a phone sign in and stay signed in, and how does the
app ship and update?

## Answer

Settled with the owner while charting, 2026-09-28.

- **Destination:** both halves on one map, web first (see the map).
- **Pages on the phone:** Login (QR), Home, Inbox, Chat (list and thread),
  Board as a list grouped by status (no drag kanban), Ticket, Docs list and
  Doc (reading; editing depends on ticket 11), Stack, Deploy with its live
  log, and Service (read plus redeploy), Runners (status only), and a trimmed
  Your settings. Everything else stays web-only.
- **Settings split**, by where the data lives today:
  - *Your settings* (gear in the sidebar footer, replacing the light/dark
    toggle): Profile, Appearance, Devices, Tokens (personal
    access tokens only), T3 pairing (computers and defaults).
  - *Configuration* (Workspace section of the sidebar, next to Runners,
    Topology, Automations), two labelled nav groups:
    - This workspace: Roles, Plays, Interview template, Members (moves out
      of the account menu), Danger zone.
    - Whole instance, each by its permission held in any workspace: Instance, Sign-in providers,
      Connectors, DNS, Registered accounts, Mention chips. Mention chips are
      stored instance-wide but gated on a workspace permission today; move
      the section and fix the gate.
  - Project settings stay where they are.
  - The account menu shrinks to Support and Logout.
- **Sessions:** a stored session per device replaces stateless tokens.
  Random token, stored hashed, checked in the same per-request user reload,
  revocation takes effect on the next request, sliding expiry (30 days web,
  90 days phone), last-active written at most hourly. No refresh tokens.
  Supersedes ADR 0041.
- **QR code:** host plus a one-time code, valid 2 minutes, single use,
  regenerable, also shown as text. No user, no long-lived secret. The
  exchange mints the phone's own device session.
- **Phone sign-in:** QR only until cloud.nexul.io.
- **Devices page:** one row per device (platform and client, last active,
  IP, sign-out ✕), current device pinned without ✕, "Sign out everywhere
  else" at the bottom, "Connect a phone" (QR) and "Connect the desktop app"
  (the connection token, moved from Tokens) at the top. No GeoIP location.
- **Web on phones:** the web app targets tablet and desktop, 768px and up.
  Record as an ADR and change the hard rule and the practices file in the
  same change.
- **Distribution:** Android only. An `.apk` from a manually triggered GitHub
  Actions workflow on Linux, attached to a GitHub release under its own tag.
- **OTA:** self-hosted update server from day one (Xprem, formerly
  expo-open-ota, is the research's pick), deployed as a stack on the owner's
  instance behind its own proxy record.
- **Version skew:** the server must always be at least as new as the app.
  An app ahead of its server refuses to sign in; an app behind keeps working.
- **Styling:** Uniwind and react-native-reusables (installed through the
  shadcn CLI). The paid web registries don't cover native.
- **Push:** instances call the push service directly with id-only payloads.
  Needs one Expo project and one Firebase project owned by the publisher;
  nothing for self-hosters to set up.
