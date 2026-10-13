# 01: Every action on every screen, and the permission it needs

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

List every action a person can take in the web app and the phone app. An action is a button, menu item, row
action, inline edit, drag, or form submit that writes. For each, record:
- where it is (file:line)
- the permission the server checks for it (route → use-case → action)
- what the screen does today for someone without that permission: hidden, shown and refused after the click, or
  shown and silently does nothing

Group by area: projects and board, tickets, docs, memories, plays, stacks and deploys, runners and machines, DNS and
gateways, topology, chat and channels, team and roles, instance settings. Flag every place where the web's check and
the server's check disagree.

## Answer

The full table is in `research/01-every-action-and-its-permission.md`, read at `0e023490`.

- There are 152 actions across 13 areas; the list adds Automations.
- 74 are ungated: shown, then refused after the click. 8 more are gated on a different permission than the server
  checks. The worst of those is the project wizard, which is gated on `projects:create` while its steps need
  `stacks:write`, `deploys:write`, `dns:write` and `projects:write`.
- Board "New category" is gated on `tickets:write`, but the server checks `projects:write`.
- The worst areas:
  - Projects and board: 21 of 27 ungated.
  - Tickets: 12 of 15 ungated, and the phone has the same gaps.
  - Stacks and deploys: 7 ungated and 4 mismatched out of 12.
  - Topology and DNS are close behind.
- Chat and Docs are nearly clean.
