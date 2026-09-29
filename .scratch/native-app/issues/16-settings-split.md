# 16 — Settings split: Your settings, Configuration, and every old link

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** tickets 01, 03, 05

## What to build

Split the one Settings page into **Your settings** at `/settings` and
**Configuration** at `/configuration`.

- Your settings: eyebrow "You", title "Settings", sections Profile,
  Appearance, Security, T3 pairing. Security holds a Tokens tab (personal
  access tokens) now; ticket 17 adds the Devices tab in front of it. Profile
  holds the existing display name and picture override form for now; ticket
  19 adds sign-in accounts.
- Configuration: eyebrow "Workspace", title "Configuration", a section nav in
  two labelled groups. *This workspace*: Roles, Plays, Interview template,
  Members, Mention chips, Danger zone. *Whole instance* (each section
  by its permission held in any workspace, ADR 0088): Instance, Sign-in
  providers, Connectors, DNS, Registered accounts.
  The Members page moves in as a section.
- Sidebar: a Configuration entry in the Workspace section after Automations;
  the footer's theme toggle becomes a gear icon button linking to
  `/settings`; the account menu keeps Support and Logout only.
- Redirects: `/settings?section=<moved section>` goes to
  `/configuration?section=<section>` keeping every other query parameter
  and the hash; `/members` goes to `/configuration?section=members`. Update
  the in-app links and the two server redirects (connector OAuth callback,
  pairing setup link) to the new URLs directly.
- Settings-style pages (Your settings, Configuration, Project settings,
  Stack) show their section nav as a top row below 1024px and as a side
  column from 1024px.
- Update the website docs pages that name Settings sections.

Reference build: branch `proto/your-settings` (footer gear, routes, nav
breakpoint). Do not copy its mock data, Vite stub, or prototype store.

## Acceptance criteria

- [ ] Every old `/settings?section=…` and `/members` URL lands on the same content at its new home, query and hash intact
- [ ] The gear opens Your settings; Configuration is reachable from the sidebar; the account menu shows Support and Logout
- [ ] Whole-instance sections are hidden without their permission, and a group left empty drops its label
- [ ] Verified at 768, 1024 and 1440px

## Surfaces

- UI routes, sidebar, account menu; server redirect URLs
- Docs: website guide pages that name Settings sections

## Read first

`practices/react-guide.md` (F1–F7 and the self-review list),
`practices/design-language.md` (Tabs and settings shell), `practices/testing.md`,
and tickets 03 and 05.

## Verification

In `web/`: `bun run lint`, `bun run typecheck`, `bun run test`; `go test ./internal/auth/... ./internal/connectors/...`; screenshots at 768, 1024, 1440px.

## Files likely touched

- `web/src/Router.tsx`, `web/src/pages/SettingsPage.tsx`, a new `web/src/pages/YourSettingsPage.tsx`, `web/src/pages/MembersPage.tsx`
- `web/src/components/settings/`, `web/src/components/sidebar/`, `web/src/components/AccountMenu*.tsx`
- `internal/auth/handler.go`, `internal/connectors/handler.go`, `web/src/models/Pairing.tsx`
- `website/src/content/docs/docs/guide/`

**Size:** L
