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

- **The key.** The setup wizard collects a client ID, secret and slug, not a manifest, so no key arrives with it. The
  owner pastes the `.pem` GitHub generates (App settings → General → Private keys) into Settings → Connectors → GitHub
  App. It is checked with GitHub (`GET /app` must answer as this client ID), stored encrypted beside the client secret
  in `connector_app_config.private_key`, and never sent back: the status says only `private_key_set`. Removing it is the
  way back. No environment variable is involved.
- **Without a key, nothing changes.** Every read keeps the connector's token, every workspace lists the connected
  account's view, and the repository list and the GitHub App card say that only the connected account's repositories
  are visible. An upgrade breaks nothing; it signals.
- **Tokens are cached per installation** until five minutes before GitHub expires them, and the installation of a
  repository per account, in one App object that lives as long as the key, client ID and server stay the same.
- **Each installation belongs to a workspace.** With a key, a workspace's repository list holds only the installations
  assigned to it, so one client never sees another's repositories. The assignment is keyed by the account's login, an
  App having one installation per account, and lives in `github_installation_workspaces`. An account may be assigned to
  more than one workspace.
- **How an installation gets its workspace.** The wizard's install link carries a state the server signs with the
  workspace and a week's expiry, minted for someone holding `projects:write` there. GitHub returns the installer to the
  sign-in callback with the state, the installation id and a code; the code is traded for the installer's own token,
  and the installation is assigned only if that token can see it, as GitHub advises, since the installation id in the
  URL can be forged. An installation made any other way, or whose claim fails, is unassigned: the GitHub App card lists
  every installation with its workspaces, and someone holding `connectors:write` assigns or unassigns it there, or with
  `workspace_update`'s `add_github_accounts` and `remove_github_accounts`.
- **The upgrade.** Migration 0088 assigns each account to every workspace whose projects already attach a repository
  from it. An account no project uses stays unassigned.
- **Permissions.** Listing a workspace's repositories takes `projects:write` in that workspace, and without one named,
  the union of the workspaces where the caller holds it. Listing installations is an instance-level read,
  `connectors:read` anywhere. Assigning takes `connectors:write` and membership of the workspace; unassigning
  `connectors:write`.

The trade-offs: an account renamed on GitHub loses its assignment until it is assigned again, the price of a key the
migration could compute from what projects already store. One instance-wide repository walk is cached for a minute and
filtered per workspace, so a workspace's search still walks every installation once a minute. A client who installs
from a plain GitHub link, or with "Request user authorization during installation" off, lands unassigned and waits for
a connector manager. The App's own webhook stays off, so an uninstalled App is noticed when GitHub stops listing the
installation rather than by an `installation` event.

Rejected: generating the App from a manifest to receive its key, which would rebuild the setup wizard's GitHub step for
an instance already running; requiring a key, which would break every instance on upgrade; a workspace picked by the
installer on GitHub, which GitHub has no field for; trusting the installation id in the callback URL; and keying the
assignment by installation id, which the migration cannot know without calling GitHub.

Amends ADR 0039: a GitHub repository resolves to the App's installation token once a key is set, not to the
connector's token. Amends ADR 0087: the wizard's repository list is checked in the workspace it lists for, and the
installations list takes `connectors:read`. Decided 2026-10-10.
