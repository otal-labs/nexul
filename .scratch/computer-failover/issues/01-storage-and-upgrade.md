# 01 — Storage for the computer order, and the upgrade

**Status:** ready-for-agent
**Blocked by:** None — can start immediately

Spec: `../spec.md`, Storage. ADRs 0165, 0166.

Migrations for `pairing_computers` (position, runs at once, chat folder), `pairing_user_defaults` (placement, clone
automatically; drop default computer and fallback project), `pairing_project_links` (placement; drop computer and T3
project), the new `pairing_project_computers`, and dropping `workspaces.auto_play_runs_at_once`, with the data moves the
spec lists. Queries, `make sqlc`, repo methods.

Done when: a migration test on a seeded database shows the old default first, each computer's runs at once at its
owner's highest workspace value, the fallback project on the old default computer, and each link's computer and T3
project kept as a remembered project checkout.
