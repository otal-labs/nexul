# A computer's runner clones with a token for the one repository

Amends ADR 0162.

Cloning onto a computer was T3 Code's `projectClone.start`, a plain `git clone` of the HTTPS URL that leans on the
computer's own git sign-in. A computer never signed in to GitHub could not clone a private repository: git, with
prompts off, failed with "could not read Username for 'https://github.com'", after T3 had already made the project.

Decision: the computer's personal runner clones, with a token the server mints for that repository alone, then T3
Code's `projects.mutate` registers the checkout as a project.

- **One repository, read only, never stored.** The token is ADR 0144's per-clone mint, `contents: read` on the one
  repository. The runner hands it to git as an `http.extraheader` through git's environment, so it is in neither the
  process list nor the checkout's config; the remote stays the plain HTTPS URL.
- **Only what the person may open.** The server mints it only for a repository of a project the person can open, and
  never hands a computer the connector token an instance without the App key falls back to.
- **T3 Code still clones when the runner cannot.** A runner older than this, a repository with no such token, or an
  instance without the App key leaves the clone to T3 Code and the computer's own sign-in, as before.
- **The clone outlives the request.** It is synchronous, bounded at ten minutes, and finishes even when the browser
  stops waiting, so a slow clone still becomes a project.

The trade-off: a clone through the runner shows no progress and cannot be cancelled, which T3's own clone can. Fetching
and pushing later still need the computer's own sign-in; the token covers the first clone only.

Rejected: putting the token in the clone URL for T3 Code, since git keeps the URL in the checkout's config and every
later fetch would carry an expired token; a token for the person's own GitHub sign-in, which reaches every repository
they can, not just this one.

Decided 2026-10-22.
