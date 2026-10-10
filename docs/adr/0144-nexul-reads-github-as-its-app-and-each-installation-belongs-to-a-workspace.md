# Nexul reads GitHub as its App, and each installation belongs to a workspace

Nexul read GitHub with one user token for the whole instance, the one stored when someone clicked Connect on the
GitHub connector. GitHub answers a user token with the intersection of what the App is installed on and what that user
can open, so when a client installed the App on their own account, their repositories never listed, and a scan, a
clone, a webhook or a pull request read failed for them too. The fix the guide offered, making the connected account a
collaborator on the client's repositories, put the instance owner's personal account inside every client's
organisation. And the one list was the same for every workspace, so a client workspace saw every other client's
repositories.

Decision: Nexul reads GitHub as the App. A JWT signed with the App's private key, issued by its client ID, is traded
for an installation access token per account (`POST /app/installations/{id}/access_tokens`), and that token lists the
installations and their repositories, scans, clones, registers webhooks and reads pull requests and commits. Installing
the App on an account is enough for its repositories to be readable.

- **The key.** New setup can create an App through GitHub's manifest flow: GitHub supplies its client ID, secret,
  slug and PEM directly to the server. The setup pass that started it alone can finish it. New setup passes carry a
  random nonce, the state is signed and stored only as a hash beside that pass's hash, and it expires after fifteen
  minutes. The callback consumes it before exchanging the code, including failed exchanges. Sign-in and connector
  credentials are stored encrypted in one transaction, refused if someone has already signed in. A no-referrer
  redirect moves GitHub's query parameters into a fragment before the browser loads setup. The browser removes them
  from its address bar and sends the code back in a JSON body; it never sees credentials.
  For an existing App, the owner pastes the PEM into Settings → Connectors → GitHub App. GitHub checks it through
  `GET /app`, and the stored status says only `private_key_set`. Removing it is the way back. No environment variable
  is involved.
- **Without a key, nothing changes.** Every read keeps the connector's token, every workspace lists the connected
  account's view, and the repository list and the GitHub App card say that only the connected account's repositories
  are visible. An upgrade breaks nothing; it signals.
- **Tokens are cached per installation** until five minutes before GitHub expires them, and the installation of a
  repository per account, in one App object that lives as long as the key, client ID and server stay the same. These
  are the server's own. A runner gets a token minted per clone, never cached, for the one repository with
  `contents: read` alone (`repositories` and `permissions` in the mint's body). Saving another client ID clears the
  stored key, a key belonging to one App.
- **Each installation belongs to a workspace.** With a key, a workspace's repository list holds only the installations
  assigned to it, so one client never sees another's repositories. The assignment is keyed by GitHub's numeric account
  id, an App having one installation per account, and lives in `github_installation_workspaces` with the login last
  seen. An account may be assigned to more than one workspace. One installation GitHub refuses, a suspended one
  included, is skipped and logged and shows its problem on the GitHub App card; the others keep listing.
- **How an installation gets its workspace.** The wizard's install link carries a random state, stored only as a hash
  with the workspace, the person who asked for the link, who must hold `projects:write` there, and a day's expiry.
  GitHub returns the installer to the sign-in callback with the state, the installation id and a code. The claim
  deletes the state before anything else, so a link works once, and goes on only while the link's maker still holds
  `projects:write` in the workspace. The code is traded for the installer's own token; the installation is assigned
  only if that token sees it, as GitHub advises, since the installation id in the URL can be forged, and only if the
  installer is the account itself (`GET /user`) or an active admin of the organisation (`GET
  /user/memberships/orgs/{org}`, which needs the App's organisation Members: read permission). GitHub lists an
  installation to anyone with access to one of its repositories, so seeing it proves nothing about the rest. An
  account already assigned elsewhere is never claimed; sharing it stays a connector manager's act. An installation
  made any other way, or whose claim fails, is unassigned: the GitHub App card lists it, and someone holding
  `connectors:write` assigns or unassigns it there, or with `workspace_update`'s `add_github_accounts` and
  `remove_github_accounts`.
- **Assignments are announced.** Assigning and unassigning publish `repository.installation.assigned` and
  `repository.installation.unassigned` through the outbox, pushed live to who holds `connectors:read` or
  `projects:write` in that workspace, so other connector managers' cards and open project wizards refresh.
- **An uninstall is noticed when listing as the App.** The App's webhook stays off, so no `installation` event
  arrives. Each read of the installations as the App records the ids of assignments made before ids were kept, follows
  a renamed login, and marks gone every assignment whose account `/app/installations` no longer lists, publishing its
  unassigned event. A gone account stays on the card, with its workspaces, until they are cleared; installing the App
  there again drops the gone rows, so it lands unassigned. A GitHub push for a repository whose installation is not
  assigned to its project's workspace, or that no project links, is ignored rather than queued to fail.
- **The upgrade.** Migration 0088 assigns each account to every workspace whose projects already attach a repository
  from it. An account no project uses stays unassigned. Migration 0090 keys the assignments by account id; its rows
  carry only the login until the first read as the App records the id, since a migration cannot ask GitHub, and a
  login GitHub no longer lists is marked gone.
- **Permissions.** Listing and scanning a workspace's repositories take `projects:write` in that workspace, and without one named,
  the union of the workspaces where the caller holds it. The installations list shows those assigned to the workspaces
  where the caller holds `connectors:read`, each with only those workspaces, filtered in SQL, and the unassigned ones
  to a holder of the instance-level `connectors:write`. A workspace's Owner therefore sees its own workspace's
  installations and, holding every instance-level permission (ADR 0088), the unassigned ones, but never one assigned
  only to workspaces they do not belong to. Assigning takes `connectors:write` in the target workspace, plus the
  instance-level `connectors:write` for an unassigned installation or `connectors:write` in a workspace already holding
  it; unassigning takes `connectors:write` in that workspace. Repository attachment, pull requests, webhooks and clone credentials also require the installation
  to be assigned to the linked project's workspace, even for background work. Owner and repository name are validated
  as single path segments before a lookup or cache access, so URL normalization cannot select another installation.
  App-mode clone authorization is mandatory: a denial fails the build before dispatch, including queued work. Only
  pre-key mode explicitly permits the runner's own credential fallback.

The trade-offs: a read of a repository's owner as the App costs a lookup of its installation, cached per owner. One
instance-wide repository walk is cached for a minute and
filtered per workspace, so a workspace's search still walks every installation once a minute. A client who installs
from a plain GitHub link, or with "Request user authorization during installation" off, lands unassigned and waits for
a connector manager, and so does an organisation installed from a link while the App lacks Members: read. An
uninstall is noticed only when someone next lists the installations as the App.

Rejected: requiring a key, which would break every instance on upgrade; a workspace picked by the
installer on GitHub, which GitHub has no field for; trusting the installation id in the callback URL; keying the
assignment by login, which a rename breaks and a reused login hands to someone else; keying it by installation id,
which changes on a reinstall; and a signed, reusable state, which let one person bind an account many times for a week.

Amends ADR 0039: a GitHub repository resolves to the App's installation token once a key is set, not to the
connector's token. Amends ADR 0087: the wizard's repository list is checked in the workspace it lists for, and the
installations list takes `connectors:read`. Decided 2026-10-10.

Amended 2026-10-10, before release, after review: the assignment is keyed by account id (migration 0090), the install
link works once for its maker and claims only for the account or an organisation admin and never an account assigned
elsewhere, the installations list and assigning are confined to the caller's workspaces, assignments publish events,
an uninstall is marked when listing as the App, a runner's token covers one repository's contents, one refused
installation no longer fails every list, a push for an unassigned installation queues nothing, and another client ID
clears the key.
