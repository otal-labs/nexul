# 01: Every action on every screen, and the permission it needs

Type: research
Status: open
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
