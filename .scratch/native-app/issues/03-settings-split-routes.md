# 03 — Settings split: routes, placement and old links

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

How does the agreed split land in the web app? The routes for Your settings and Configuration, what the old `/settings?section=…` links redirect to (the connector OAuth callback and the pairing setup link both point there today), what Profile contains, how the Members page moves under Configuration, the fix for the Mention chips permission gate, and whether Your settings is a page or an overlay like Discord's.

## Answer

Agreed with the owner 2026-09-28.

- **Routes.** `/settings` becomes Your settings; Configuration lives at
  `/configuration`. Every old `/settings?section=<x>` for a section that moved
  redirects to `/configuration?section=<x>`, keeping the rest of the query
  (`tab`, `setup`, `connector`, `connected`, `error`) and hash; `/members`
  redirects to `/configuration?section=members`. Personal sections keep their
  `/settings` URLs. The two server redirects (connector OAuth callback, the
  sign-in providers callback) point at the new URLs directly. Sixteen web
  links and fourteen website docs pages that mention Settings are updated in
  the same change.
- **A normal page, not an overlay.** Deep links, back button and redirects
  work with no extra state. Ticket 05 owns the look.
- **Profile.** Editable display name and avatar (the API exists; today only
  the first-login wizard reaches it, a one-way door). The sign-in account is
  shown read-only, and every other provider the instance has turned on
  offers "Link via Google" / "Link via Discord" / "Link via GitHub". Linking
  moves provider identity off the `users` row into its own table (one user,
  many identities; sign-in looks up by identity). Unlink is the way back out,
  refused only for the last identity so nobody locks themselves out.
  Linking needs no admission step: the person is already signed in.
- **Mention chips.** The template moves from the instance settings row to the
  workspace, stays gated on `workspaces:write`, and sits under This
  workspace. No data to carry over.
- **Members** moves from the account menu to Configuration → This workspace.
  The account menu keeps Support and Logout.
