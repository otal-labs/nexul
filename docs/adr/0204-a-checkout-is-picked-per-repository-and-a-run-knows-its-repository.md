# A checkout is picked per repository, and a run knows its repository

Amends ADR 0165 and ADR 0197's checkout tie-break.

A project link kept one checkout per computer, but a project can hold an app repository and a tests repository. A
computer with both either asked which one every time or, once picked, sent every run there, and a computer holding only
the tests repository looked ready for app work, because its checkout matched one of the project's remotes. A play that
works on the tests repository, such as a review an automation starts on `git.pr_opened` from it, could not be sent to a
tests checkout without also catching app work.

Decision: a run names the repository it works on, and placement matches a checkout of that repository alone.

- **The run's repository.** A run an automation queues while handling a `git.*` event works on that event's repository;
  the SDK's `runPlay` sends it. Otherwise the play says which, by role: `app`, the default, or `tests`, since a play is
  workspace-wide and cannot name one project's repository. A project with no tests repository, or a repository the run
  names that is not the project's, falls back to the app's.
- **Picks per repository.** A person's picked checkouts are kept per computer and repository
  (`pairing_project_checkouts`). A computer with both repositories takes each run in its own checkout without asking;
  only two clones of the same repository are ambiguous. A computer without the run's repository is passed over as
  `no_checkout`, naming the repository when the project has several.
- **Picks from before.** A pick saved with no repository counts for the repository its T3 project is a clone of, or,
  when it clones none of the project's repositories (an empty folder), for the app's, which is what it served before.

ADR 0197's tie-break falls out of this rather than being a rule of its own: a run that names nothing works on the app,
so the app's clone is the one it matches.

The trade-off: a play that should work in the tests repository has to say so, and an automation reacting to a git event
in a repository outside the project still runs in the app's checkout. Rejected: inferring the repository from the
play's label or instructions, which no one can see or correct; a repository per auto play, which leaves pressed runs and
`runPlay` without one; and keeping one checkout per computer with a second "tests checkout" field, which breaks again
at a third repository.

Decided 2026-11-19.
