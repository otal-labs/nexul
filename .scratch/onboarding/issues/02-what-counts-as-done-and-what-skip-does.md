# 02: What counts as done for each step, and what skipping does

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A step is done when what it sets up exists. For each built-in step, what exactly
is that, read from data that exists today or from a new record?

- Profile: `first_login_done`, or a name and avatar set?
- Connect GitHub: a row in `github_user_links`?
- Your computer: paired, or paired and the setup checks confirmed? A
  tunnel-paired computer that has no runner?
- Your projects: "opened a project" is not recorded anywhere today. Record a
  first visit, or count it done when the person is in at least one project?
- Add a deploy server: any runner connected, or one this person enrolled?
- Get the phone app: a signed-in device?

And skipping: the dot shows while any step is left. Does skipping a step in the
wizard keep the dot, silence it until something changes, or silence it for good
for that step? Is a skip stored per person per step?

## Answer

Owner's calls, 2026-10-13.

- **Profile**: `first_login_done` changes meaning to "the Welcome wizard has had
  its first run". It is set when the person finishes or closes the wizard the
  first time, which is also what stops the wizard opening by itself again.
  Profile is the first screen, so it counts as done once the flag is set and
  never lights the dot.
- **Connect GitHub**: done while the person has a working row in
  `github_user_links`. If GitHub revokes the link, the step is not done again.
- **Your computer**: done while at least one of the person's computers is paired
  and its setup confirmed. A tunnel-paired computer counts once its setup is
  confirmed. Removing the last such computer makes the step not done again.
- **Your projects**: done while the person is in at least one project. It does
  not light the dot on its own, because a person can't add themselves to a
  project. It lights only when someone gives the person the step (see Skip),
  and stays lit until they open a project or skip it. When they are in no
  project, the step says the team hasn't added them yet.
- **Add a deploy server**: done once a deploy server *this person added* is
  connected. That is new data: record who enrolled each deploy runner (personal
  runners already record an owner, migration 0091). Runners can show
  "added by" too.
- **Get the phone app**: stays done once a phone has connected. Signing the
  phone out later does not bring the step back.
- **Skip**: stored per person per step. A skipped step stops lighting the dot
  and shows as Skipped in the account menu's Welcome list, where it can be done
  at any time. The skip lasts until the person is given the step again, either
  because a role they hold gains the tick or because they are given a role that
  has it. The step is then not done, lights the dot, and only another skip
  silences it. This is so a person struggling with something can be handed
  the step by an admin and see exactly what to do.
- **New records**:
  - per person per step: when it was skipped, and, for the phone, when it
    was first done
  - on each deploy runner: who enrolled it

  Everything else is worked out from data that already exists.
