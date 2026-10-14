# An action a person can't take is hidden unless a role of theirs shows it locked

Of the actions the web and phone apps offer, many were shown to people who couldn't take them and refused only after
the click. Hiding every one fixes that, but leaves a person with no way to learn that a deploy or a delete exists, or
which role would let them do it.

Decision: an action the viewer can't take is hidden. A viewer holding any role with **Show locked actions** on sees it
as a Locked action instead, dimmed, with the reason: "Your role can't <verb>", or the source sentence of the rule that
refused it on that one thing (ADR 0148). A thing they can't read stays not found either way.

- **The switch is a field on the role**, `roles.show_locked_actions`, off by default. It is not a permission: it
  grants nothing and reveals no data, only that an action exists, so anyone who can edit a role sets it without
  holding anything extra, like a Welcome wizard step (ADR 0149).
- **One role with it on is enough.** `/api/workspaces/{id}/me` returns `show_locked_actions`, true when any role the
  viewer holds in that workspace has it on. Holding a role without it never turns it off.
- **Owner can do everything**, so it never has locked actions; the Owner role carries the switch off and can't be
  edited.
- **Template roles carry it** (ADR 0149). Junior Developer, Senior Developer, DevOps and Administrator ship with it
  on and Client off, so a new workspace starts with developers seeing what they can't do and clients seeing only what
  they can. The upgrade turns it on in the instance's stored copy of those four, matched by their stable key, and
  never touches a role inside an existing workspace.
- **Clone and duplicate copy it**; it rides `role.updated` so a holder's open pages redraw when it flips; `role_update`
  and `workspace_list` carry it for agents.

The trade-offs: a role with the switch on shows its holders the shape of what they can't do, so it stays off for a
role handed to people outside the team; existing roles upgrade off, so nobody sees a locked action until an Owner
switches it on; and the switch is per role, so a person holding a developer role and a client role sees the locked
actions of both.

Rejected: showing locked actions to everyone, which tells a client everything they can't do; a workspace-wide setting,
which can't be on for developers and off for clients; a per-person setting, which is one more thing to maintain on
every member; and a permission bit, which would need hold-what-you-give (ADR 0088) for something that grants nothing.

Builds on ADR 0148 and ADR 0149. Decided 2026-10-14.
