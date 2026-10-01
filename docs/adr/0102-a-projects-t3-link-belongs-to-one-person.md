# A project's T3 link belongs to one person

A project's T3 link (the paired computer, T3 project, provider, model, and model options `@Agent` runs with inside
that project) was one row per project, set from Project settings. It pointed at the computer of whoever saved it, so
every teammate's `@Agent` in that project ran on that one person's machine. That broke the promise in ADR 0029 that a
turn runs on the mentioning user's own environment, and it put a personal choice, which of my computers and which
model, in a shared settings page that anyone with the project open could overwrite.

Decision: a project link is per person. Each person sets their own for each project they can open, from Your
settings → T3 pairing → Projects, and the link's computer is always one of their own. Someone with no link for a
project falls back to their own pairing defaults, exactly as an unlinked project did before.

- **Storage.** `pairing_project_links` is keyed by `(user_id, project_id)`, with `user_id` cascading from the user,
  `project_id` from the project, and `computer_id` from the computer, every other column kept.
- **Migration 0054** rebuilds the table and gives each existing link to the owner of the computer it pointed at, so
  the person who set it keeps it unchanged. Nobody else gets a copy; their turns in that project move to their own
  defaults on upgrade.
- **Resolution.** A mention, a play, and the run dialog's preselected harness read the mentioning user's own link,
  then their defaults. Pairing no longer looks up a computer regardless of its owner.
- **Access.** Reading, setting, and clearing a link take being able to open the project (`RequireProject` with the
  empty action), so a project hidden from a Restricted member reads as not found (ADR 0097). `GET
  /api/pairing/projects` lists the caller's links on every project they can still open; a link on a project that
  has since been hidden stays stored but is not listed or offered.
- **HTTP.** `GET`, `PUT`, and `DELETE /api/pairing/projects/{id}` keep their paths and now mean the caller's own link
  (ADR 0082). No MCP tool exposes a project link, and no event is published for one.
- **Web.** Project settings loses its T3 pairing section; an old link to it lands on the first section like any
  unknown one.

The trade-off: a team can no longer point everyone at one machine with a single setting. Each person's `@Agent` runs
on their own computer, so a teammate without a paired computer cannot borrow someone else's, and a team that wants
the same T3 project and model everywhere has each person set it once. Accepted, because a shared machine meant one
person's laptop carried everyone's turns with that person's skills and MCP setup, which is the failure ADR 0063's
setup gate exists to stop.

Rejected: keeping the shared link as a project default under each person's own, which keeps the borrowed-computer
path alive; copying the shared link to every member on upgrade, which would point their turns at a computer they
don't own.

Amends ADR 0058: a run's resolved target comes from the starter's own project link or defaults. Restores ADR 0029 for
project-linked mentions. Decided 2026-10-01.
