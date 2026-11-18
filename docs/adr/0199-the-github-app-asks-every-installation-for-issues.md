# The GitHub App asks every installation for Issues

Supersedes the GitHub issues part of ADR 0123's rejected list.

ADR 0123 rejected a GitHub issues source kind because it would ask every installation for the Issues permission. Two-way
issue sync between a project's tickets and its repository's issues, and the one-shot issues import, both need to receive
issue events and to read and write issues, so the permission now has a use that every attached repository shares.

Decision: the App's manifest asks for Issues: Read and write beside its other permissions, on every installation, not as
an option. The repository webhook subscribes to `issues`, and issue changes and their comments become `git.issue` and
`git.issue_comment`; a pull request's comments stay `git.pr_comment`.

The cost: an App created before this asks each installation's owner to accept the new permission, and until they do the
installation shows "Issues permission missing" and its issues are not read. An instance that never uses issue sync still
asks for the permission.

Rejected: an opt-in Issues permission, which would split every issue feature into a path with it and one without.

Decided 2026-11-18.
