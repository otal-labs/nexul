# Repositories are discovered with each person's own GitHub sign-in

Supersedes the discovery and assignment parts of ADR 0144. Amends ADR 0040 (a GitHub sign-in now keeps its token)
and ADR 0039.

ADR 0144 listed a workspace's repositories as the App, for the installations assigned to that workspace, and before
a private key through the instance's GitHub connector token, the account of whoever clicked Connect. Both read GitHub
as someone other than the person asking. A client saw only what the owner's account could open; a workspace member
saw every repository of every installation assigned to their workspace, private ones included; and a holder of
`connectors:write` could assign a client's unassigned installation to their own workspace and read it.

Decision: a person finds repositories through their own GitHub sign-in, and the App's own access is only for
background work on repositories already attached to a project.

- **The person's link.** Signing in with GitHub, or Connect GitHub for someone who signed in another way, keeps the
  GitHub App user token and its refresh token, encrypted at rest beside the person (`github_user_links`). Connect
  GitHub is the sign-in's link mode (ADR 0040), so it joins the existing account and never makes a second one; a
  GitHub account already linked to someone else is refused and nobody's token changes. A token is refreshed shortly
  before it lapses, under one lock, since a refresh token works once; a refresh GitHub refuses clears the tokens and
  marks the person to reconnect, and a network failure is only retried. Disconnect forgets the token; unlinking the
  GitHub sign-in forgets it too. Nothing is backfilled: a token arrives on the next GitHub sign-in or Connect GitHub.
- **Discovery is the person's view.** The project wizard's list and search, its scan, the attach pickers, the
  installations card and `repository_list`/`repository_scan` read `GET /user/installations` and their repositories
  with that token, which GitHub answers with exactly what the person can open where the App is installed. Without a
  live token there is no list: the answer is a `github_not_connected` or `github_reconnect` refusal saying how to
  connect, and no other credential is tried in its place. An MCP turn acts as the person who started it.
- **Attaching links the account.** Attaching a GitHub repository to a project requires the attacher's own list to
  hold it, and records its installation's account against the project's workspace in `github_installation_workspaces`,
  publishing `repository.installation.assigned` as before. That row now means "this workspace's projects attach this
  account's repositories". The server's own attaches, which have no person, keep the background check.
- **Background work stays where it was.** Clones with the one-repository token, webhooks, scans of attached
  repositories, and pull request and commit reads use the App's installation tokens, only for a repository attached
  to a project whose workspace the account is linked to. Without a key they keep using the connector's token, the
  legacy path for already-attached repositories; discovery never does.
- **No assignment by hand.** The assign route stays registered for older clients (ADR 0082) and refuses;
  `workspace_update` loses `add_github_accounts`. The install link carries no state and links nothing, so the claim
  and its `github_install_states` table go (migration 0092). The installations card lists only what the viewer's own
  token sees, each with the viewer's workspaces whose projects attach one of its repositories; detaching one takes
  `projects:write` there, and an existing assignment with nothing attached grants nothing.

The trade-offs: every person who lists repositories needs their own GitHub link, so after the upgrade everyone signs
in with GitHub again or connects it; Nexul now holds a user token per person, refreshed every eight hours while
used; a repository a teammate can open but you cannot is not attachable by you; and the first list after an App
install may need a refresh, since the person's view is cached for a minute.

Rejected: keeping App-side listing filtered by assignment, which still showed every member a client's private
repositories and kept an admin able to link someone else's account; falling back to the connector token when a
person has no link, which is the bug; and a second OAuth app for repository access, when the sign-in already is the
App's user authorization.

Decided 2026-10-10.
