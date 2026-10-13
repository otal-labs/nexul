# 01: How the permission rework lands on roles, invitations and computers

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

The Welcome wizard hangs off Roles, and the permission rework
(ADR 0148) was changing them when this question was charted. What will the rework leave
in place for this effort to build on?

- The shape a role takes after the rework: table, columns, `member_roles`, how
  several roles combine, and whether ADR 0148 replaces the Restricted member
  with per-project role rules.
- Invitations after the rework: how many roles one carries and how project
  access is granted.
- How Instance templates are stored and copied in today
  (`0055_instance_templates.sql`, the template use-case, Settings → Templates),
  and what a Template role needs from that machinery.
- Where the personal-runner effort stands: what "Your computer" can rely on,
  and what a tunnel-paired computer looks like to it.
- Which tickets of the rework are still open, and which of them touch the
  files this effort would touch.

## Answer

Details and references in `research/permission-rework-fit.md`.

- The rework leaves the `roles` table as it is. A person holds a set of roles
  through `member_roles` (on master since #596), the set may be empty, and
  roles combine as a union of allows. Nothing on a role can hold wizard steps
  today, so they need a new column or table.
- ADR 0148 replaces the Restricted member: a project is opt-in and lists the
  roles that reach it. A Template role therefore reaches no project until a
  project lists it, and "Your projects" must cover being in none.
- After the rework an invitation carries one or more roles and nothing per
  person.
- Instance templates are rows in `instance_templates` (kind, key, text body),
  and a row exists only once someone edits it. A Template role needs a new
  template kind with five default keys and a JSON body, plus a step at
  workspace creation, next to `SeedWorkspaceDefaults`, that copies them in.
- Personal runners are built for pairing, facts and the Add a computer dialog.
  Removing tunnel pairing, shell jobs and sharing are not. A tunnel-paired
  computer has no runner and no facts until it moves, so "a computer exists"
  holds for both kinds, but the setup checks need a runner.
- The `permission-switch` branch freezes `internal/access`, `internal/tenancy`,
  the chat reads and `web/src/components/{access,team,member,settings}`. That
  covers the role editor, invitations and Team, so the build starts after the
  switch merges.
