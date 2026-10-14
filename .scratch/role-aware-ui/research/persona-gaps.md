# Persona gaps: what each kind of newcomer hits today

Read from the code on 2026-10-11 (master at 5c700fe7), statically, nothing run. Five walks: client, junior
developer, senior developer, devops, administrator. Each gap is tagged with the kind of fix it needs.

## Shared

- Only the Owner role is seeded per workspace (`internal/roles/usecase.go:49`); every other role is hand-built.
  The invite form's role select is a dead end on a fresh workspace (`InvitationGrantRow.tsx:60`).
- First sign-in is one flag per user (`first_login_done`); a new role or a second workspace never re-triggers it.
  `FirstLoginWizardPage` is profile only, says "leave the GitHub defaults" to every provider, shows "1 / 2" for
  one step, and Back goes to `/login` while signed in.
- `WorkspaceHome.tsx:31` is developer marketing ("Runners build and deploy to servers you own") with Board and
  Topology buttons only; nobody gets a next step.
- The owner wizard's "Finish setup" is disabled until a computer is paired (`SetupT3CodeStep.tsx:10-19`).

## Client

- Sign-in may need GitHub: Google and Discord show only if the owner enabled them (`internal/auth/handler.go:72-73`),
  and no setup step offers that.
- The invite page doesn't name the projects: the server drops project access from the acceptance
  (`server/cmd/wire_invitations.go:68-72`); the client sees a role name and raw "Allow: tickets:read".
- No project yet: Board, Docs, Memories say "Create one to start", with no button for them; create forms say
  "Create a project first".
- Chat: a Restricted member reads no public channel (`internal/chat/usecase.go:56,155-159`); there are no
  project channels; empty Channels / Voice / DM headers.
- Developer UI on the ticket page: "+ Link branch/PR" ungated (`DevelopmentSection.tsx:154`), Trail always shown,
  Developer/Tester pills on create. Interview banner talks about agent rules; Memories rides the same gate.
- Settings shows Computers, a GitHub access card and Tokens.
- Inbox empty with no hint; the doc Questions panel exists but nothing points to it. Phone app needs a connect
  code from web Settings, no download link, no ticket creation.

## Junior developer

- "Fix with AI" only applies in the progress stage (`internal/plays/usecase.go:288`); the rail renders nothing
  otherwise, with no hint to move the card.
- Computer setup isn't part of readiness (`PairingHooks.tsx:188-215`): the person learns of it after pressing Run.
  The auto-play queue says "run it in Settings → Computers" as plain text (`PlayQueueUtility.tsx:31`).
- No computer: PlayButton disabled with "Add a computer in Settings to run plays." and no link (`Pairing.tsx:485`).
- "T3 project" is unexplained, unfiltered, and has no clone/create action (`HarnessProjectField.tsx:43`), though
  the backend can clone.
- Offline computer: "Offline" / "T3 Code on your computer is offline." with no next step.
- Bug: `plays:run` without `memories:read` breaks the run dialog (`PlayRunDialog.tsx:33-60`).
- No "My tickets" view; the Inbox is the only signal.
- Connect GitHub isn't needed for this role: the agent pushes from the person's computer.

## Senior developer

- Project settings show edit and delete controls to readers (`ProjectSettingsContent.tsx:21-33`).
- Project access is read-only in the project, "Change access from Team." with no link, and hidden without
  `members:write`; UI still says "Restricted members".
- `PlayButton`, `PlaysMenu`, `HarnessReadinessNote` return null without `plays:run`, with no reason shown.
- "Create one to start" shown to people without `projects:write`; MCP `project_create` is owners only, stricter
  than the UI.
- The ticket prefix is permanent and the wizard doesn't say so (`WizardProjectStep.tsx:44-50`).
- The wizard is deploy-first ("Pick the repository to deploy"); Settings → Repositories is typed owner/name only.
- The interview is offered to people who can't run it; readiness strings have no links.
- "To tickets via AI" in the Questions panel waits for clarification to close, while the Plays menu offers it now.
- Plays and auto plays are hard to find (Configuration → Plays; auto plays a tab in the dialog).
- No PR list page (MCP only). Memories empty state doesn't say what a memory is.

## Devops

- No way to delete a machine (no DELETE route); removed runners leave a dashed offline card forever.
- Stack env and secrets editable only in the creation wizard.
- Cloudflare is instance-level (`connectors:write`); members get "Connect a DNS provider…" with no button and no
  name of who can fix it.
- A failed deploy has no Retry / Redeploy; dead letters are MCP only.
- Many write buttons render without the permission (runners, deploy, rollback, delete stack, gateways, exposures,
  topology "Add a node").
- The Runners page also needs `machines:read`; a role without it gets "Couldn't load runners."
- No web delete for a Cloudflare tunnel.
- Add runner: no "waiting for connect" or success state; closing the dialog drops the code.
- The DNS wizard is about the instance's own hostname and defaults to the "instance" tunnel; `/wizard/onboarding/dns`
  has no permission gate.
- The version chip says "updating" for offline runners.

## Administrator

- Team shows no setup progress, though `first_login_done`, `pairing_computers` and `github_user_links` exist.
- A redeemed invitation vanishes from the list; no "accepted by".
- Silent invitation failures: the creator's authority changed (deleted at redeem, invitee sees "expired"), or the
  invitee was already in the workspace (role never applied).
- One role per invitation while members can hold several.
- Invite copy uses "Restricted" and "Every project".
- Audit has no web page. Upgrade blocked by "instance runner is not connected" has no next step.
- Disable copy says "their computers are disconnected"; only Remove deletes computers.

## Where a per-role field would live

`roles` table (next migration 0099), `queries/roles.sql` CreateRole/UpdateRole, `roles_repo.go`, `internal/roles/model.go`,
`roles/handler.go` roleRequest, `roles/mcp.go` role_update, `web/src/models/Role.ts`, `RoleEditorDialog.tsx` (draft
and dirty cover name and actions only), `RoleSectionTabs.tsx` (tabs are permission areas; Getting started would be
a fourth, non-area tab).
