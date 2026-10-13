# 02: Every one-way door

Research for `issues/02-one-way-doors.md`. Read-only pass over the tree at commit `0e023490` (branch
`spec/role-aware-ui`). Nothing was run; every entry below comes from reading the code. "Unverified" marks a claim
that rests on a partial read.

## Method

1. Listed every mutating HTTP route (`rg` over `"(POST|PUT|PATCH|DELETE) /…"` in `internal/` and `server/cmd/`,
   495 registrations including duplicates) and every MCP tool name (`mcptool.New`).
2. Listed every `api.post/put/patch/delete` call in `web/src` (hooks, and the few call sites that pass a path
   variable) and every mutation in `native/src`. The phone has about ten: send message, mark thread read, ticket
   status, ticket developer/tester, open ticket thread, redeploy, notification read, sign out (current, others,
   one), and push token.
3. For each create or enter operation, looked for its delete, undo, edit or leave counterpart on web, phone, HTTP
   and MCP. A gap is recorded when a surface has the way in and not the way out, when the server has the way out
   and the web does not, or when a state can be entered and not left.
4. Read the confirmation and empty-state copy for any promise of a way back, and checked each against the server.

The phone is a thin client (read-mostly), so "no way out on the phone" is only noted where the phone itself has
the way in.

## Doors

"Way out today" lists web / phone / HTTP / MCP. "Web-only?" means the server and MCP already do what is needed, so
the fix is a button and a hook.

| Door | Way in (surfaces, file:line) | Way out today | What the server already supports | Web-only? |
|---|---|---|---|---|
| 1. Machine (known) | Created as a side effect of a runner enrolling: `internal/runner/enrollment.go:258-269` (`GetByName`, else `machines.Create`). Web: Add runner dialog, `RunnersPanel.tsx`. HTTP `POST /api/runners/enrollments`. | Web: none. Phone: none. HTTP: none. MCP: none. Removing the last runner (`DELETE /api/runners/{id}`, `RemoveRunnerButton.tsx`) leaves the machine; `MachineGroup.tsx:41-52` keeps drawing it as an offline dashed card. | Only list, rename, set stack root (`PATCH /api/machines/{id}`, `internal/runner/http.go:33`), discover, import. The `MachineRepo` interface (`internal/runner/machine.go:20-30`) has no delete; no `DELETE FROM machines` anywhere in `internal/`. MCP: `machine_list`, `machine_discover`, `machine_import` only. Stacks name a machine by string (`Stack.Machine`, `internal/deploy/model.go:278`), so a delete would need a rule for stacks that still target it. | No. New repo method, use case, route and MCP tool, plus a rule for stacks and runners still on it. |
| 2. Cloudflare tunnel (known) | Web: `useCreateTunnel` `DnsHooks.tsx:60`, `useRouteTunnelHostname` `:73`, `useProvisionTunnelAgent` `:91`, in `TunnelDeployStep.tsx`, `TunnelHostnameStep.tsx`; first-run `SetupTunnelRungs.tsx`. HTTP `POST /api/dns/tunnels` (`internal/dns/handler.go:34`). MCP `dns_tunnel_create`. | Web: none. The web never lists tunnels except as a picker in `CreateGatewayDialog.tsx:75`. Phone: none. HTTP: `DELETE /api/dns/tunnels/{tunnelID}` (`handler.go:42`). MCP: `dns_tunnel_delete` (`internal/dns/tunnel_mcp.go:145`). | `DeleteTunnel` (`internal/dns/tunnel_usecase.go:251-274`) needs `dns:delete`, deletes the tunnel at Cloudflare (a missing one counts as deleted) and the local row. It does not check for gateways or computers whose `tunnel_id` points at it (no FK; `migrations/0001_schema.sql:449`, `0008_computer_tunnel.sql:2`), and it does not remove the CNAME that `RouteTunnelHostname` created (`tunnel_usecase.go:185-199`, record id kept on the tunnel row). Whether Cloudflare refuses to delete a tunnel with live connectors is unverified. | The button and a list are web-only. A guard against deleting a tunnel in use, and removing its CNAME, would be a server change. |
| 3. Stack environment and secrets, and the rest of the stack definition (known) | Web: `WizardEnvStep.tsx:75` (`useUpdateStackEnv`, `StackHooks.tsx:141`), reached only from the project wizard. Compose path, ports, mounts, command, network, build source and machine are set in `WizardServiceStep.tsx` and `CreateStack`. | `useUpdateStackEnv` has one use site: `WizardEnvStep.tsx:47`. `useUpdateStack` is used by branch rules (`BranchDeployRuleRow.tsx`, `AddBranchDeployRuleForm.tsx`, `BranchRulesForm.tsx`) and the wizard attach step (`WizardServiceStep.tsx:65`) only. `StackPage.tsx` has sections overview, logs, exposures, branches, history, danger: no environment or settings section. Even in the wizard a key cannot be removed or blanked: `WizardEnvStep.tsx:73-75` drops empty values and merges over `stack.env`. Phone: none. HTTP: yes. MCP: yes. | `PATCH /api/stacks/{id}` (`internal/deploy/handler.go:43`, full replace of the `Stack` record, so the client sends the whole record). MCP `stack_update` (`internal/deploy/mcp.go:427`) takes `env` as set/unset, plus name, machine, strategy, compose path, network, ports, mounts, command, build source, branch rules. `GET /api/stacks/{id}` returns `env` values in the clear; no masking was found in `internal/deploy` outside logs and events (`events.go:161`). Whether reading them is gated by a narrower permission than `stacks:read` is unverified. | Yes for the form and hook. A set/unset patch (as MCP has) would avoid sending every value back, but the full-replace PATCH works. The product question of who may read secret values is separate. |
| 4. Failed deploy: no retry (known) | Web: Build and deploy form, Redeploy, Rollback on the stack page (`StackDeployActions.tsx`); wizard `WizardEnvStep.tsx:77`. Phone: Redeploy (`native/src/components/deploys/RedeployScreen.tsx`). HTTP `POST /api/deploys`. MCP `stack_deploy` (`internal/deploy/mcp.go:559`). | Web: the deploy page (`DeployPage.tsx`, `DeployProgressSection.tsx:41-43`) shows a cancel button only while running; a failed deploy shows nothing. The stack page can start a new deploy (build form when a build source exists, Redeploy for a run stack with an image, Rollback), but not from the failed deploy and not for a compose stack with no repository (`StackDeployActions.tsx:153-157`). Phone: Redeploy for run stacks only; no cancel of a running deploy, no rollback. HTTP/MCP: re-POST. | `Service.Deploy` (`internal/deploy/usecase.go:168`) rejects only a second *active* deploy, so a failed one can be followed at once. The `Deploy` record (`model.go:64-85`) stores `Image` but not the `Ref` that was built, so a build deploy of a non-default branch cannot be replayed from the record. Triggers record `RuleID`, `TicketID`, `PRNumber` but no branch. | Mostly. A retry button that re-POSTs the stack's image or default build branch is web-only. Replaying the exact ref needs the ref stored on the deploy (server). |
| 5. Ticket delete | Web: Create ticket form (`CreateTicketForm.tsx`), board. Phone: none. HTTP `POST /api/tickets`. MCP `ticket_create`. | Web: none (`rg` finds no ticket-delete call in `web/src`; only labels, blockers, found-in, category are removable). Phone: none. HTTP: `DELETE /api/tickets/{id}` (`internal/tickets/handler.go:120`). MCP: `ticket_delete` (`internal/tickets/mcp.go:32`). | `Service.Delete` (`internal/tickets/usecase.go:832`) needs `tickets:delete`, publishes `ticket.deleted` and unblocks tickets it blocked. The permission `tickets:delete` (`internal/platform/permissions/permissions.go:34`) is in the role catalog, so roles can be granted a right with no button behind it. | Yes. |
| 6. Ticket branch and pull request links | Web: `useLinkBranch`, `useLinkPR` (`TicketHooks.tsx:186-212`) in `DevelopmentSection.tsx`. HTTP `POST /api/tickets/{id}/branches`, `/prs` (`handler.go:122-123`). | Web, phone, HTTP, MCP: none. `TicketLinksSection` removes blockers and found-in only. No unlink route or service method exists (`rg -i unlink internal/tickets` is empty). | Nothing: a link is add-only (`LinkPR` is a no-op on a duplicate). | No. New unlink route and use case first. |
| 7. Workspace | Web: `CreateWorkspaceForm.tsx`, `useCreateWorkspace` (`WorkspaceHooks.tsx:131`). HTTP `POST /api/workspaces` (`internal/tenancy/handler.go:53`). | Web, phone, HTTP, MCP: none to delete or archive. Rename exists (`PATCH`, `handler.go:54`; MCP `workspace_update`). The workspace Danger zone says "No workspace-wide action is destructive yet" (`DangerZoneSection.tsx:5`). | `tenancy` has `RemoveMember` only; no workspace delete or archive method. | No. |
| 8. DNS records the web creates | Web: instance record (`useCreateInstanceRecord`, `DnsHooks.tsx:28`, `InstanceRecordForm.tsx`); tunnel hostname route creates a proxied CNAME (`tunnel_usecase.go:185-199`). | Web: no record list, edit or delete (no hook for `/api/dns/zones/{id}/records` beyond zones). Phone: none. HTTP: `DELETE /api/dns/zones/{zoneID}/records/{recordID}` (`handler.go:27`). MCP: `dns_record_delete` (`internal/dns/mcp.go:211`). Exposure delete (web) removes the record it made; instance-record and route-step records have no web counterpart. | Full record CRUD over HTTP and MCP. `POST/DELETE /api/dns/service-hostnames` (`handler.go:32-33`) are used by no web or phone code (only `web/e2e`). | Yes. |
| 9. Archived doc is invisible | Web: Archive (`DocHooks.tsx:81`, `DocActionsMenu.tsx:52`). MCP `doc_update archived`. | Restore exists on web (`DocActionsMenu.tsx:57`, `DocHooks.tsx:94`), HTTP (`POST /api/docs/{id}/restore`) and MCP. But the list never shows the state: `rg archived web/src` finds no badge, filter or section in the list or header, only the menu toggle and two pickers that hide archived docs (`DocPickerList.tsx:24`, `InterviewSourcePicker.tsx:30`). The HTTP list returns archived docs (`ListDocs` has no archived filter), so they sit in the list looking live. | MCP `doc_list` has `include_archived` (`internal/docs/mcp.go:40`); the HTTP list returns everything with the `archived` flag. | Yes (a marker and a filter). |
| 10. Account disable then reactivate | Web: Disable (`TeamAccountActions.tsx:38`). HTTP `PATCH /api/auth/accounts/{id}`. MCP `account_update`. | Reactivate and Restore exist on all three. The reverse is partial: disabling revokes the person's personal runners (`internal/runner/events.go:121-133`, wired at `server/cmd/subscriptions.go:332-335`), and the code comment says "reactivating the account later brings none back". | See left. | Web copy only; the server behaves as designed. |
| 11. Phone has the way in, not the way out | Send message (`native/src/hooks/ChatHooks.tsx:118`); Redeploy (`StackHooks.tsx:80`). | Phone: no edit or delete of a message, no cancel of a running deploy, no rollback. Web has all three. | `PATCH/DELETE /api/chat/messages/{id}`, `POST /api/deploys/{id}/cancel`, `POST /api/stacks/{id}/rollback` all exist. | Client-only (phone). |
| 12. Enrollment codes | Runner (`POST /api/runners/enrollments`), automations host (`/api/automation-hosts/enrollments`), computer (`/api/pairing/computers/enrollments`); web Add runner and Add host dialogs. | Web, phone, HTTP, MCP: no list and no revoke of an unused code; it dies after an hour (`internal/runner/enrollment.go:21`, `internal/automations/hosts_usecase.go:20`), 24 hours for the instance's own (`internal/runner/enrollment.go:22`, `internal/automations/hosts_usecase.go:21`). | Redeeming consumes it. | No (server list/revoke first). Low severity. |
| 13. Instance upgrade | Web: `UpgradeButton` (`InstanceVersionSection.tsx:110`), `POST /api/instance/upgrade`, MCP `instance_upgrade`. | None: no downgrade on any surface. The confirmation says "downloads the release and restarts its services" and does not say it cannot be undone. | Migrations are forward-only (project memory); "Try again" exists for a failed upgrade. | No; by design. Copy only. |
| 14. Notifications | Server pushes. | Mark read and read-all exist (web, phone). No mark unread, no dismiss, on any surface. | `POST /api/notifications/{id}/read`, `/read-all`. | No. Low severity. |
| 15. DM, ticket, doc and interview threads | Created from web (`useCreateDm`, ticket and doc thread hooks in `ChatHooks.tsx:142-167`) and from the phone (ticket thread, `native/src/hooks/TicketHooks.tsx:82`). | None by design: delete and rename apply to channels only (ADR 0094, `docs/adr/0094-channels-are-managed-with-their-own-permission-bits.md:14`). | See ADR. | No; by design. |

Count: 15 rows. Rows 1 to 4 are the four known doors, all confirmed. Rows 5 to 15 are new. Rows 1, 6, 7, 12, 13, 14
and 15 need a server change; 8 and 9 are web-only; 2, 3, 4 and 5 are mostly web-only (see the caveats in the cells);
10 is copy; 11 is phone-only.

### Confirming the four known doors

- **Machine delete.** Confirmed. No `DELETE` route, no repo method, no MCP tool; machines also survive their last
  runner. Fix needs the server.
- **Cloudflare tunnel delete on web.** Confirmed. `DnsHooks.tsx` has fetch, create, route, agent provisioning and
  verify for tunnels and a delete only for gateways (`:147`) and exposures (`:204`). The route
  (`DELETE /api/dns/tunnels/{id}`) and MCP tool (`dns_tunnel_delete`) exist.
- **Stack env and secrets editable only in the wizard.** Confirmed. `useUpdateStackEnv` is defined at
  `StackHooks.tsx:141` and used only at `WizardEnvStep.tsx:47`. The stack page has no section for it, and the wizard
  step cannot clear a key.
- **Retry or redeploy of a failed deploy.** Confirmed for the deploy page. The stack page does offer a new deploy
  in three shapes, so the missing piece is a retry on the failed deploy and a path for compose stacks with no
  repository.

## Copy that promises a way back

| Copy (file:line) | What the code does |
|---|---|
| "Its credential stops working. Add it again to bring it back." `web/src/components/runner/RemoveRunnerButton.tsx:23` | Adding it again enrols a new runner with a new id and credential. If the same machine name is given, `GetByName` reattaches it to the same machine (`enrollment.go:258`), so the runner returns in effect. Wording "bring it back" is loose; machine and stacks were never lost. |
| "Its automations move to the instance host and its credential stops working. Add it again to bring it back." `web/src/components/automationHost/RemoveAutomationHostButton.tsx:22` | Re-enrolling makes a new host row (`RemoveHost` deletes it, `automation_hosts_repo.go:133-147`). The automations stay on the instance host until someone moves each back (`PATCH /api/automations/{id}/host`). Nothing restores their placement. Unverified: the exact fallback mechanism for placement when the host row is deleted (`placedOn`, `hosts_usecase.go:285`, treats a nil host id as the instance host). |
| "@Agent stops working through this computer, and the Nexul app removes itself from it. Add it again to bring it back." `web/src/components/settings/RunnerComputerRow.tsx:68`, and "Pair it again to bring it back." `ComputerRow.tsx:83` | Delete removes the computer record, its runner, its MCP token and its tunnel (`internal/pairing/tunnel_usecase.go:129-157`). Pairing again creates a new computer. Whether its setup choices and pairing-project mappings come back is unverified; the delete removes the row they hang on. |
| Disable: "lose sign-in but keep their workspace access, and their computers are disconnected." `TeamAccountActions.tsx:38`; toast "Account disabled. Their computers were disconnected." `TeamHooks.tsx:56-61` | True. But Reactivate's confirmation says only "be able to sign in again" (`TeamAccountActions.tsx:44`), and the computers do not return (`internal/runner/events.go:121-122`). The Remove confirmation does say restore does not bring them back; Disable and Reactivate do not. |
| "Rollback re-deploys the deploy from <time>, the last healthy one." `StackDeployActions.tsx:25,139`; button enabled whenever any healthy deploy exists (`StackPage.tsx:53`) | `Service.Rollback` (`internal/deploy/usecase.go:211-225`) refuses a healthy deploy with no image ("has no healthy image to roll back to", 409). A compose or build deploy records no image, so the button is enabled and the click fails. |
| "Attach a repository to build and deploy this stack from here." `StackDeployActions.tsx:156` | The stack page has no control to attach a repository. The only code that sets `build_source` on an existing stack is the wizard attach step (`WizardServiceStep.tsx:104-115`). Unverified: whether another page does. |
| "You can change it later from the DNS page." `web/src/components/setup/SetupDomainStep.tsx:30` | The instance address is edited on the instance settings card (`InstanceUrlSection.tsx`, `useUpdateSettings`, `PUT /api/auth/settings`). The DNS page (`DnsOnboardingPage.tsx`, "Set up DNS") sets up records and tunnels and does not change the address. The first-run endpoint refuses once setup is done (`internal/auth/setup.go:227-229`). |
| "Nothing here can't be undone yet." `web/src/components/settings/DangerZoneSection.tsx:8` | A double negative that reads as "everything can be undone"; the workspace has no destructive action because it has no delete at all (door 7). |
| "…and you can restore it later with a new URL." `web/src/hooks/useBotActions.tsx:29` | True: restore exists (`DeletedBotsFold.tsx`, `PATCH deleted:false`). Listed because it is the pattern the others lack. |
| "…and you can add them back later." `web/src/components/team/TeamRemovedNotice.tsx:16`; "…you can add it again." `web/src/components/project/RepoRow.tsx:26` | True: the Team dialog adds a member back (`TeamHooks.tsx:47`) and repositories can be added to a project. |

## Checked and fine

Each has its way in, its way out and a way to see the state, on the surfaces that have the way in.

- Accounts: disable, reactivate, remove, restore (web Team, HTTP, MCP); the Remove copy states what restore does
  not bring back.
- Workspace members: add, change roles, remove, add back (`TeamHooks.tsx:47-49`).
- Invitations: create, delete (web, HTTP, MCP).
- Roles: create, clone, edit, delete; copy says delete "can't be undone" (`RoleRow.tsx:29-31`).
- Docs: create, lock and unlock, archive and restore (marker gap in door 9), delete (refused while a ticket names the
  doc), folders add and delete, watch and unwatch.
- Memories: create, edit, revert to a version, delete; interview answers skip and answer again; interview sources and
  drafts add and remove.
- Chat channels: create, rename, private and public, add and remove members, leave, delete; messages edit and
  delete; reactions toggle; voice join and leave.
- Bots: create, rename, regenerate URL, delete, restore (Deleted fold).
- Plays and auto plays: create, edit, enable and disable, delete; runs stop and continue; queue cancel and resume.
- Automations: create, enable and disable, delete, token mint and revoke, secrets put and delete, version merge and
  rollback.
- Automation hosts and runners: enroll and remove (copy caveats above).
- Computers: pair, re-pair, remove (tears down runner, token and tunnel), MCP token mint and revoke, setup choices
  editable.
- Projects: create, edit, delete, repositories add and remove; categories, ticket types and statuses create and
  delete; ticket labels, blockers, found-in and category add and remove.
- Stacks: create, delete, deploy, rollback (caveat above), cancel a running deploy, branch rules add and remove.
- Gateways and exposures: create and delete (gateway delete refuses while exposures exist, with the reason).
- Topology nodes: add and remove.
- Connectors: GitHub and Cloudflare connect and disconnect, GitHub App key set and clear.
- Repository installations: attach to and detach from a workspace.
- Auth: personal tokens create and revoke, sessions sign out, identities link and unlink, GitHub link and unlink.
- Attachments: upload and delete on web (`AttachmentRow.tsx`, composer).
- Access grid and permission overwrites: set and discard (not examined in depth).
- Instance templates: save, clone, delete, reset.

## Not examined

- Ticket test pass and fail (`POST /api/tickets/{id}/test/pass|fail`): whether a result can be revised.
- Access grid and permission overwrite clearing, beyond the draft discard.
- Whether deleting an adopted (imported) stack removes containers that existed before Nexul; the confirmation says
  "Removes the stack and its containers".
- Whether disconnecting the Cloudflare connector while tunnels and gateways exist is guarded.
- Native app screens beyond their hooks (every mutation hook was read; no component-level audit).
