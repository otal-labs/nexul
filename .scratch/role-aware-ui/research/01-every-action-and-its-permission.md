# 01: Every action and the permission it needs

Read statically from commit `0e023490` (branch `spec/role-aware-ui`) on 2026-10-13. Nothing was run. Builds on
`persona-gaps.md`, which was read at an older master; every row below was re-read at this
commit.

## Method

- Server: every non-GET route registered under `internal/*/handler*.go` and `server/cmd/routes.go` (about 267
  registrations, counting the `/api/services` alias of `/api/stacks` and the public auth routes), followed into the
  use-case to the `require*`, `Require*`, `loadStack`, `getChecked`, `HasPermission` or `permissions.*` call.
- Web: every `api.post/put/patch/delete` in `web/src` (231 call sites) mapped to its hook, then to the component that
  calls the hook. A component counts as gated when it, or a parent that decides whether it renders, reads
  `useAreaAccess`, `useHasPermission`, `useHasInstancePermission`, `useCanOpenSection`, `useCanManageAccess`, a
  server-sent flag (`can_close`, `can_detach`, `can_manage_members`), or a section/route gate. Route gates live in
  `web/src/Router.tsx`; section gates in `web/src/hooks/AccessHooks.tsx` (`settingsSectionsFor`) and
  `web/src/models/Access.tsx`.
- Phone: every write in `native/src` (seven hooks) mapped to its screen. The phone has `AreaGate` on three tab routes and
  `hiddenUnless` on the tab bar (`native/src/app/(tabs)/_layout.tsx:31`); both are read gates.
- Line numbers point at the control or the mutation call, whichever is the place a gate would go.

Verdicts in the **Today** column:

- GATED: hidden, or shown disabled/read-only, for someone without the permission.
- UNGATED: shown to people who lack the permission and refused by the server after the click.
- MISMATCH: gated, but on a permission or scope different from the one the server checks.
- N/A: no control exists, the action has no role permission (own account, own computer), or the server needs only read
  access.

"Server permission" uses the catalog names (`area:action`). "Project" means the server resolves it in the target
project; "anywhere" means `RequireAnywhere` (held in any workspace).

## Projects and board

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create a project (wizard) | `Router.tsx:66-67`, `WizardProjectStep.tsx:27`; entry points `NoProjectsState.tsx:16`, `ProjectServices.tsx:16` | no | `projects:create` (`workspace/usecase.go:82`) | GATED |
| Reorder projects | no control | no | `projects:create` | N/A |
| Set the prefix | no control (set at create) | no | `projects:write` | N/A |
| Rename a project | `ProjectName.tsx:19` via `ProjectGeneralSection.tsx:22` | no | `projects:write` | UNGATED |
| Change setup (stack link, env keys, finish) | `ProjectSettingsContent.tsx:26` (`editProjects`) | no | `projects:write` (`setup_usecase.go:63`) | GATED |
| Save the tests-location answer (wizard) | `WizardRepositoryStep.tsx:50` | no | `projects:write` | MISMATCH (route gate is `projects:create`) |
| Delete a project | `ProjectDangerZoneSection.tsx:54,72` | no | `projects:delete` | UNGATED |
| Add a repository | `ProjectRepos.tsx:40`, `AddRepoForm.tsx:19` | no | `projects:write` | UNGATED |
| Remove a repository | `RepoRow.tsx:32,53` | no | `projects:write` | UNGATED |
| Create a category (project settings) | `ProjectCategories.tsx:80`, `AddCategoryForm.tsx:25` | no | `projects:write` | UNGATED |
| Create a category (board "New category") | `BoardFilterBar.tsx:75,137` | no | `projects:write` | MISMATCH (gated on `tickets:write`) |
| Rename or recolor a category | `CategoryEditForm.tsx:22` | no | `projects:write` | UNGATED |
| Reorder categories | `ProjectCategories.tsx:55` | no | `projects:write` | UNGATED |
| Delete a category | `CategoryRow.tsx:60` | no | `projects:delete` | UNGATED |
| Create a status column | `NewStatusForm.tsx:28` | no | `projects:write` | UNGATED |
| Rename a status or change its icon | `StatusRow.tsx:34` | no | `projects:write` | UNGATED |
| Reorder status columns (settings and board drag) | `BoardStatusColumnsSection.tsx:25`, `KanbanColumn.tsx:60`, `KanbanBoard.tsx:119`, `useBoardActions.tsx:101` | no | `projects:write` | UNGATED |
| Delete a status | `StatusRow.tsx:80` | no | `projects:delete` | UNGATED |
| Create a ticket type | `BoardTicketTypesSection.tsx:57` | no | `projects:write` | UNGATED |
| Rename or recolor a ticket type | `TicketTypeRow.tsx:54,83` | no | `projects:write` | UNGATED |
| Set a ticket type's body template | `TicketTypeTemplateForm.tsx:15` | no | `projects:write` | UNGATED |
| Delete a ticket type | `TicketTypeRow.tsx:47` | no | `projects:delete` | UNGATED |
| Set a label color | `LabelColorRow.tsx:22` | no | `tickets:write` | UNGATED |
| Drag a ticket to another column | `KanbanBoard.tsx:147`, `useBoardActions.tsx:66` | `BoardScreen.tsx:51` | `tickets:write` | UNGATED (phone gate is `tickets:read` only, `board/_layout.tsx:11`) |
| Drag a ticket to another category lane | `useBoardActions.tsx:70-75` | no | `tickets:write` | UNGATED |
| Reorder tickets within a column | `useBoardActions.tsx:78` | no | `tickets:write` | UNGATED |
| "+" new ticket in a column | `KanbanColumn.tsx:76`, handler `useBoardActions.tsx:56-57` | no | `tickets:write` | UNGATED, and silent: the button shows, `openCreateTicketDialog` is `undefined` without the permission, the click does nothing |
| New ticket and Report a bug (board menu) | `BoardFilterBar.tsx:75,137` | no | `tickets:write` | GATED |
| Project people and access | `ProjectSettingsContent.tsx:21,27` (`projects:access`) | no | `projects:access` on the project (`access/usecase.go:878`) | GATED |
| Project to computer link | `ProjectLinkForm.tsx:45,61` | no | membership only (`pairing/usecase.go:524`) | N/A |

UNGATED in this area: 21 of 27 actions. The project settings page (`ProjectSettingsContent.tsx:20-34`) gates only
the setup card and the people card; General, Categories, Repositories, Board and Danger zone render for every reader of
the project.

## Tickets

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create a ticket | `useCreateTicketDialog.tsx:19-21` (board, doc page, command palette) | no | `tickets:write` in the project | GATED (hook returns `undefined`, callers hide) |
| Report a bug from a ticket | `TicketBugsSection.tsx:33` | no | `tickets:write` | UNGATED |
| Edit title and body | `TicketDetail.tsx:40-47` | no | `tickets:write` | GATED (readers get static text, no edit session) |
| Change status | `TicketPage.tsx:52`, `TicketStatusRow.tsx:37-47` | `TicketScreen.tsx:74`, `StatusPickerScreen.tsx:44` | `tickets:write` | UNGATED |
| Change type | `TicketPage.tsx:55`, `TicketTypeRow.tsx:53` | no | `tickets:write` | UNGATED |
| Add or remove a label | `TicketPage.tsx:58,61`, `TicketLabelsRow.tsx:105,114` | no | `tickets:write` | UNGATED |
| Set the category | `TicketCategoryRow.tsx:32,35` | no | `tickets:write` | UNGATED |
| Set developer or tester | `TicketPersonRow.tsx:47` | `TicketScreen.tsx:103` (Assign to me) | `tickets:write` | UNGATED |
| Link a branch | `DevelopmentSection.tsx:106` | no | `tickets:write` | UNGATED |
| Link a pull request | `DevelopmentSection.tsx:134` | no | `tickets:write` | UNGATED |
| Add a link (blocked by, blocks, found in, source doc) | `AddTicketLinkMenu.tsx:34-37` | no | `tickets:write` (blocker needs `tickets:read`) | UNGATED |
| Remove a link | `TicketLinksSection.tsx:37,47,58,65,70` | no | `tickets:write` | UNGATED |
| Pass a test | `TicketTestSection.tsx:37` | no | `tickets:write` (`tickets/testing.go:111`) | UNGATED (the file comment at `:16` says anyone who can see the ticket may pass or fail it) |
| Fail a test | `TicketTestSection.tsx:41`, `TestFailForm.tsx:19` | no | `tickets:write` (`testing.go:164`) | UNGATED |
| Attach or delete a file | `AttachmentsSection.tsx:33,43,74` | no | `tickets:write` | GATED |
| Delete a ticket | no control | no | `tickets:delete` | N/A |
| Move a ticket to another project | no control | no | `tickets:write` on the target | N/A |
| Open the ticket thread | `TicketThreadSection` | `TicketScreen.tsx:107` | `tickets:read` | N/A |

UNGATED in this area: 12 of 15 actions. The properties rail has a read-only fallback when a handler is missing
(`TicketPropertiesPanel.tsx:23`) but `TicketPage.tsx:46-62` always passes the handlers, and the category and person rows
take no handler at all.

## Docs

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create a doc | `useCreateDocDialog.tsx:15`, `DocsListPane.tsx:47` | no | `docs:write` in the project | GATED |
| Create a folder | `NewDocFolderButton.tsx:14` | no | `docs:write` | GATED |
| Rename or delete a folder | `useDocFolderActions.tsx:53-54` | no | `docs:write` | GATED |
| Edit title and body | `DocDetail.tsx:59` | no | `docs:write` on the doc | GATED (read-only view, no session) |
| Move a doc to a folder | `useDocRowActions.tsx:65` | no | `docs:write` | GATED |
| Clone a doc | `useDocRowActions.tsx:68` | no | `docs:clone` on the source, `docs:write` on the destination | GATED (destination write not checked client-side) |
| Lock or unlock | `DocActionsMenu.tsx:42-51`, `useDocRowActions.tsx:62` | no | `docs:lock` | GATED |
| Archive or restore | `DocActionsMenu.tsx:52-61`, `DocPage.tsx:46-47` | no | `docs:write` (`docs/usecase.go:412`) | UNGATED |
| Delete a doc | `useDocRowActions.tsx:70` | no | `docs:delete` | GATED |
| Watch or unwatch | `DocWatchersSection` | no | `docs:read` | N/A |
| Answer, clear, "anything else" on clarification questions | `DocQuestionsPanel.tsx:38` | no | `docs:write` | GATED |
| Close a clarification | `DocQuestionsPanel.tsx:32` (server `can_close`) | no | `docs:write` plus the play's permission | GATED |
| Create a ticket from a doc | `DocPage.tsx:42` | no | `tickets:write` | GATED |
| Doc permission rules | `DocPermissionsButton.tsx:12` | no | `permissions:write` on the doc | GATED |
| Attach or delete a file | `AttachmentsSection.tsx:33` | no | `docs:write` | GATED |
| Open the doc thread | `DocThreadButton.tsx:17` | no | `docs:thread` | GATED |
| Import markdown, save a named version | no control | no | `docs:write` | N/A |

UNGATED in this area: 1 of 15 actions.

## Memories

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create a memory | `useCreateMemoryDialog.tsx:10`, `MemoriesListPane.tsx:36` | no | `memories:write` in the project | GATED |
| Edit a memory | `MemoryDetail.tsx:49-101` | no | `memories:write` | GATED |
| Pin ("always included") | `MemoryPinSwitch.tsx:22`, `MemoryListRow.tsx:34` | no | `memories:write` | GATED (switch disabled) |
| Revert to a version | `MemoryDetail.tsx:68,129` | no | `memories:write` | GATED |
| Clone | `MemoryDetail.tsx:53,66`, `MemoryListRow.tsx:31` | no | `memories:clone` on the source, `memories:write` on the destination | GATED (destination write not checked client-side) |
| Delete | `MemoryDetail.tsx:58`, `MemoryListRow.tsx:32` | no | `memories:delete` | GATED |
| Save or skip interview answers | `InterviewChecklist.tsx:39` | no | `memories:write` | GATED |
| Add, restance, remove interview sources | `InterviewSourcesSection.tsx:39` | no | `memories:write` | GATED |
| Accept or dismiss a suggested draft | `InterviewChecklistFeed.tsx:68`, `InterviewSuggestionCard.tsx:23,46` | no | `memories:write` | UNGATED (unverified: the card takes no `readOnly` prop while the question form beside it is hidden for readers, `InterviewChecklistFeed.tsx:16`) |
| Edit the interview template | `InterviewTemplateSection.tsx:12` | no | `memories:template` | GATED (form read-only) |
| Open the interview thread | `InterviewBanner.tsx` | no | `memories:read` | N/A |

UNGATED in this area: 1 of 10 actions (unverified).

## Plays and auto plays

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Run a play | `PlayButton.tsx:46`, `PlaysMenu.tsx:42`, `PlaysRailSection.tsx:22`, `RunItButton.tsx:21` | no | `plays:run` on the play (`plays/run_launch.go:560`) | GATED (hidden, no reason shown) |
| Stop a run | `PlayButton.tsx:50`, `PlaysMenu.tsx:46` | no | starter, or `plays:write` (`run_observer.go:232`) | GATED |
| Answer a waiting question | `TrailQuestionCard.tsx:21` | no | starter, or `plays:write` (`run_launch.go:331`) | UNGATED (shown on state alone) |
| Continue an ended run | `TrailContinueForm.tsx:50`, `TrailDetailBody.tsx:61` | no | starter, or `plays:write` (`run_launch.go:385`) | UNGATED |
| Retry the decisions check | `DecisionsCheckNotice.tsx:29` | no | `plays:run` on the decisions play | UNGATED |
| Create a play | `PlaySettingsSection.tsx:84` | no | `plays:write` | GATED |
| Edit a play or a built-in's instructions | `PlayRow.tsx:50` | no | `plays:write` | GATED |
| Delete a play | `PlayRow.tsx:53` | no | `plays:delete` | GATED |
| Play permission rules | `PlayRow.tsx:52` | no | `plays:write` | GATED |
| Create or edit an auto play | `AutoPlaysList.tsx:54`, `AutoPlayComposer.tsx:31,59` | no | `autoplays:write` | GATED |
| Toggle an auto play | `AutoPlayRow.tsx:62` | no | `autoplays:write` | GATED (switch disabled) |
| Duplicate an auto play | `AutoPlayRow.tsx:46` | no | `autoplays:write` | GATED |
| Delete an auto play | `AutoPlayRow.tsx:47` | no | `autoplays:delete` | GATED |
| Set the daily cap | `AutoPlayCapLine.tsx:25` | no | `autoplays:write` | GATED |
| Cancel a queued run | `QueuedRunRow.tsx:35` | no | the person it runs on, or `autoplays:write` (`queue.go:153`) | GATED |
| Resume paused auto plays | `AutoPlaysPausedRow.tsx:27` | no | the ticket's developer, or `autoplays:write` (`queue.go:235`) | GATED |
| Pick the computer for a queued run | `PickComputerMenu.tsx:40` | no | only the person it lands on (`queue.go:182`) | N/A (unverified whether the menu is shown to others) |

UNGATED in this area: 3 of 16 actions.

## Stacks and deploys

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Deploy a ref | `StackDeployActions.tsx:45` | no | `deploys:write` (`deploy/usecase.go:173`) | UNGATED |
| Redeploy the last image | `StackDeployActions.tsx:86` | `StackScreen.tsx:47-50`, `RedeployScreen.tsx:28` | `deploys:write` | UNGATED (phone `canRedeploy` at `StackScreen.tsx:33` is about data, not permission) |
| Roll back | `StackDeployActions.tsx:111` | no | `deploys:write` (Rollback calls Deploy) | UNGATED |
| Cancel a deploy | `DeployCancelButton.tsx:11` via `DeployProgressSection.tsx:42` | no | `deploys:write` (`usecase.go:763`) | UNGATED |
| Delete a stack | `StackDangerZoneSection.tsx:30,51` | no | `stacks:delete` | UNGATED |
| Delete a branch deployment | `BranchDeploymentRow.tsx:21,32` | no | `stacks:delete` | UNGATED |
| Add, edit, remove a branch-deploy rule | `AddBranchDeployRuleForm.tsx:36`, `BranchDeployRuleRow.tsx:26,39,57,61`, `StackBranchDeploySection.tsx:36` | no | `stacks:write` | UNGATED |
| Create a stack (wizard) | `WizardServiceStep.tsx:161` | no | `stacks:write` | MISMATCH (route gate is `projects:create`) |
| Edit stack env, then first deploy (wizard) | `WizardServiceStep.tsx:110,119`, `WizardEnvStep.tsx:75,77` | no | `stacks:write`, `deploys:write` | MISMATCH |
| Branch rules in the wizard | `BranchRulesForm.tsx:63` | no | `stacks:write` | MISMATCH |
| Scan a repository (wizard) | `WizardRepositoryStep.tsx:58` | no | `projects:create`, anywhere (`repository/usecase.go:29`) | GATED |
| Attach a repository to an unmanaged stack | `StackHeaderSection.tsx:66` | no | `stacks:write` and `projects:write` | MISMATCH (gated on `projects:create`) |

UNGATED in this area: 7 of 12 actions; with MISMATCH, 11 of 12 miss the server's permission. `web/src/hooks/ServiceHooks.tsx`
(`/api/services`) and `CreateServiceForm` have no mounted caller, so they are not counted.

## Runners and machines

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Add a runner (enrollment code) | `RunnersPage.tsx:15`, `MachineGroup.tsx:79` | no | `runners:write`, anywhere (`runner/enrollment.go:56`) | UNGATED |
| Remove a runner | `RunnerRow.tsx:40`, `RemoveRunnerButton.tsx` | no | `runners:delete`, anywhere | UNGATED |
| Edit a machine's stack root | `MachineGroup.tsx:91`, `EditableStackRoot.tsx` | no | `machines:write`, anywhere (`usecase.go:260`) | UNGATED |
| Import stacks from a machine | `MachineGroup.tsx:34,52`, `ProjectWizardImportPage.tsx:47,58` | no | `machines:read` (discover), `stacks:write` in the project, plus `stacks:write` anywhere with gateways (`deploy/import.go:156-162`) | MISMATCH (gated on `projects:create`) |
| Delete a machine | no route, no control | no | none | N/A |
| Pair, repair, rename, delete a computer; computer MCP token; setup run | `ComputerRow.tsx`, `RunnerComputerRow.tsx`, `PairComputerForm.tsx` | no | own computers only (`pairing/usecase.go:1005`), no role permission | N/A |

Reading the page also needs `machines:read` (`runner/usecase.go:226`) in addition to the `runners:read` route gate.
UNGATED in this area: 3 of 4 actions.

## DNS, gateways, exposures

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create a gateway | `GatewaysSection.tsx:27`, `CreateGatewayDialog.tsx:96` | no | `dns:write`, anywhere (`dns/gateway_usecase.go:20`) | UNGATED |
| Delete a gateway | `GatewayRow.tsx:50` | no | `dns:delete`, anywhere | UNGATED |
| Expose a service (create an exposure) | `ServiceHostnameSection.tsx:49`, `ExposeServiceDialog.tsx:53` | no | `dns:write`, anywhere | UNGATED |
| Remove an exposure | `ExposureRow.tsx:54` | no | `dns:delete`, anywhere | UNGATED |
| Set up DNS stepper (tunnel, agent, route hostname, instance record, reverse proxy) | `DnsOnboardingPage.tsx:25`, `TunnelDeployStep.tsx:56-57`, `TunnelHostnameStep.tsx:67`, `InstanceRecordForm.tsx:45`, `ProxyEntryPathForm.tsx:64,68`; entry link `GatewaysSection.tsx:25` | no | `dns:write`, anywhere | UNGATED (the route has no area gate, `Router.tsx` outside `workspaceRoutes`) |
| Expose a service in the wizard | `WizardReachStep.tsx:125` | no | `dns:write` | MISMATCH (route gate is `projects:create`) |
| Connect Cloudflare | `ConnectCloudflareAction.tsx:17-28` | no | `connectors:write`, anywhere | GATED (button, or "Ask the instance owner to connect Cloudflare.") |
| Delete a tunnel | no control | no | `dns:delete` | N/A |

UNGATED in this area: 5 of 7 actions.

## Topology

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Move a node, pan or zoom (the canvas saves) | `TopologyFlow.tsx:69,95`, `useTopologyView.tsx:81` | no | `topology:write` (`topology/usecase.go:70`) | UNGATED (a reader's pan or zoom sends the PUT after 600 ms with no deliberate click) |
| Add a node | `TopologyPalette.tsx:14`, `CanvasEmptyHint.tsx:30`, `AddNodeDialog.tsx:91` | no | `topology:write` | UNGATED |
| Edit a hand-drawn node | `FreeNodeDialog.tsx:81` | no | `topology:write` | UNGATED |
| Remove a hand-drawn node | `FreeNodeDialog.tsx:91` | no | `topology:delete` | UNGATED |
| Draw an edge | `flowActions.ts:24` then the canvas save | no | `topology:write` | UNGATED |
| Delete an edge | `flowActions.ts:19` then the canvas save | no | `topology:delete` | UNGATED |
| "Add a service" link | `CanvasEmptyHint.tsx:25` | no | leads to the project wizard | GATED |

UNGATED in this area: 6 of 7 actions. `TopologyFlow.tsx:47` reads `can` only for opening stacks and logs.

## Chat and channels

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| New channel or voice channel | `ChatSidebarSection.tsx:39`, `useCommandActionItems.tsx:48` | no | `channels:write` (`chat/usecase.go:244`) | GATED |
| New direct message | `ChatSidebarSection.tsx:40`, `useOpenDirectMessage.tsx:20` | no | `chat:write`, or membership when every other person is an Owner (`conversation_usecase.go:48-58`) | GATED (web is stricter in the all-Owners case) |
| Rename a channel | `useChannelRowActions.tsx:42` | no | `channels:write` | GATED |
| Delete a channel | `useChannelRowActions.tsx:43` | no | `channels:delete` | GATED |
| Make private, who stays, back to public | `ChannelSettingsCard.tsx:17`, `WhoStaysForm.tsx:20`, `useChannelSettingsActions.tsx:38` | no | `channels:write` | GATED |
| Remove someone from a private channel | `ChannelMemberRow.tsx:27` | no | `channels:write` (self is free) | GATED |
| Add people to a private channel | `AddChannelPeopleForm.tsx:20` | no | read access to the channel | N/A |
| Leave a channel | `useChannelSettingsActions.tsx:52` | no | own membership | N/A |
| Create, edit, regenerate, delete, restore a bot | `BotsSection.tsx:26`, `BotDetailFooter.tsx:38`, `useBotsDialog.tsx:13` | no | `botwebhook:write` / `botwebhook:delete` | GATED |
| Open a doc thread | `DocThreadButton.tsx:17` | no | `docs:thread` | GATED |
| Open a ticket thread | `TicketThreadSection` | `TicketScreen.tsx:107` | `tickets:read` | N/A |
| Post, edit, delete, react to a message | `ConversationThread.tsx:124` | `ChatComposer.tsx:19` | read access to the conversation; edit and delete are author-only (`message_usecase.go:416`) | N/A |
| Answer or interrupt an agent turn in chat | `ChatQuestionCard.tsx:19`, `ConversationThread.tsx:116` | no | no role check in `agent/handler.go`; `GetConversation` only | N/A (unverified) |
| Join a voice channel | `VoiceCallSection` | no | `voice/usecase.go:49` checks only that it is a voice channel | N/A (unverified) |

UNGATED in this area: 0 of 8 actions with a role permission.

## Team, roles, invitations, workspaces

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create an invitation | `TeamSection.tsx:53`, `CreateInvitationForm.tsx:34` | no | `members:write` in every invited workspace (`invitations_repo.go:215,422`) | UNGATED (Team opens for `accounts:read` alone, `AccessHooks.tsx:60`) |
| Revoke an invitation | `InvitationsFeed.tsx:42` | no | `members:write` in every invited workspace | UNGATED |
| Change someone's roles in a workspace | `TeamWorkspacePanel.tsx:22`, `TeamPersonDialog.tsx` | no | `members:write` (`tenancy/usecase.go:376`) | GATED (server `can_manage_members`) |
| Add a person to a workspace | `TeamAddWorkspacePopover.tsx:22` | no | `members:write` | GATED |
| Disable, reactivate, restore an account | `TeamAccountActions.tsx:13,38-52` | no | `accounts:write`, anywhere | GATED |
| Remove an account | `TeamAccountActions.tsx:14,53` | no | `accounts:delete`, anywhere | GATED |
| Create, edit, delete a role | `SettingsPageContent.tsx:19` | no | `roles:write` | GATED (section hidden) |
| Clone a role | `RoleRow.tsx:39` | no | `roles:clone` on the source, `roles:write` on the target | GATED |
| Rename the workspace; mention chip layout | `SettingsPageContent.tsx:15,33` | no | `workspaces:write` | GATED (section hidden) |
| Create a workspace | `WorkspaceSwitcher.tsx:20` | no | `workspaces:create`, anywhere | GATED |
| Doc, play and project permission rules | `useCanManageAccess.tsx:10`, `PanelAddTarget.tsx:25`, `RowActionItems.tsx:135` | no | `permissions:write` on the doc, `plays:write`, `projects:access` | GATED |
| Access page: project lists, public-channel hiding | `Router.tsx` `access` route, `AccessProjectsTab.tsx:24`, `AccessPage.tsx:26` | no | projects the actor manages; channels and finish are Owner-only (`access/grid.go:118`) | GATED |
| Detach a GitHub installation from a workspace | `GitHubInstallationWorkspaceChip.tsx:30` | no | `projects:create` in that workspace (`repository/installations.go:337`) | GATED (server `can_detach`) |

UNGATED in this area: 2 of 13 actions.

## Instance settings, connectors, templates

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Set the instance URL | `InstanceUrlSection.tsx:30` | no | `instance:write`, anywhere (`auth/bootstrap.go:232`) | UNGATED (section opens on `instance:read`) |
| Request an instance upgrade | `InstanceVersionSection.tsx:123` | no | `instance:write` (`runner/usecase.go:495`) | UNGATED |
| Set or clear a sign-in provider | `OAuthProviderForm.tsx:27`, `OAuthProviderActions.tsx:36` | no | `instance:write` (`auth/service.go:354`) | UNGATED |
| Connect a connector with OAuth | `ConnectorCard.tsx:41,90` | no | `connectors:write`, anywhere (`connectors/usecase.go:137`) | UNGATED (section opens on `connectors:read`) |
| Connect a connector with manual credentials | `ManualConnectorDialog.tsx:62`, `ConnectorCard.tsx:82` | no | `connectors:write` | UNGATED |
| Disconnect a connector | `ConnectorCard.tsx:61,95` | no | `connectors:write` | UNGATED |
| Set up a connector app, GitHub app key | `ConnectorsSettingsPanel.tsx:27`, `ConnectorCard.tsx:83-90` | no | `connectors:write` | GATED (disabled button with the reason as its title) |
| Edit, reset, clone instance templates | `models/Access.tsx:12` (section opens on `templates:write`), `CloneTemplateForm.tsx:27` | no | `templates:write`, anywhere (`templates/usecase.go:257`) | GATED |
| Integrations, dead letters, audit | no web control | no | `integrations:*`, `audit:read` | N/A |
| Own tokens, sessions, GitHub link, devices, connect code, notifications | `PersonalAccessTokensSection.tsx`, `DevicesFeed.tsx`, `GitHubAccessRow.tsx`, `ConnectCodePanel.tsx`, `InboxPage.tsx` | `SessionHooks.tsx`, `NotificationHooks.tsx`, `ConnectHooks.tsx` | the signed-in user's own records | N/A |

UNGATED in this area: 6 of 8 actions.

## Automations (not in the ticket's list)

| Action | Web | Phone | Server permission | Today |
|---|---|---|---|---|
| Create an automation | `AutomationsPage.tsx:23`, `NewAutomationDialog.tsx:36` | no | `automations:write` (`automations/usecase.go:99`) | UNGATED |
| Enable or disable | `AutomationRow.tsx:40`, `AutomationDetailHeader.tsx:48` | no | `automations:write` | UNGATED |
| Edit config values | `AutomationConfigForm.tsx:29` | no | `automations:write` | UNGATED |
| Choose the host | `AutomationPage.tsx:58` | no | `automations:write` | GATED |
| Mint or revoke the token | `AutomationTokenSection.tsx:36,45` | no | `automations:write` | UNGATED |
| Set, replace, delete a secret | `AutomationSecretForm.tsx:24`, `AutomationSecretRow.tsx:44` | no | `automations:write` | UNGATED |
| Merge or roll back a version | `AutomationVersionDiff.tsx:38`, `AutomationVersionRow.tsx:25` | no | `automations:write` | GATED |
| Delete an automation | `AutomationPage.tsx:46` (tab hidden) | no | `automations:delete` | GATED |
| Add an automation host | `AutomationHostsSection.tsx:18` | no | `automations:write` | UNGATED |
| Remove an automation host | `AutomationHostRow.tsx:24` | no | `automations:delete` | UNGATED |

UNGATED in this area: 7 of 10 actions.

## Phone app writes, in one place

The phone has seven mutation hooks (`native/src/hooks/{Ticket,Stack,Chat,Notification,Session,Connect}Hooks.tsx`). The
board and deploys routes are guarded by `AreaGate` for reads (`app/(tabs)/board/_layout.tsx:11`,
`app/(tabs)/deploys/_layout.tsx:11`); nothing gates a write. The sheet routes `app/(sheets)/board/status-picker` and
`app/(sheets)/deploys/redeploy` sit outside those layouts (unverified whether a deep link into them is gated).

| Write | Where | Server permission | Today |
|---|---|---|---|
| Change a ticket's status (board drag, ticket screen, picker sheet) | `BoardScreen.tsx:51`, `TicketScreen.tsx:74`, `StatusPickerScreen.tsx:44` | `tickets:write` | UNGATED |
| Assign a ticket to me | `TicketScreen.tsx:103` | `tickets:write` | UNGATED |
| Redeploy a stack | `StackScreen.tsx:50`, `RedeployScreen.tsx:28` | `deploys:write` | UNGATED |
| Open a ticket thread | `TicketScreen.tsx:107` | `tickets:read` | N/A |
| Post a chat message | `ChatComposer.tsx:19` | conversation read | N/A |
| Notifications, sessions, push token, connect code | `NotificationHooks.tsx`, `SessionHooks.tsx`, `ConnectHooks.tsx` | own records | N/A |

## Mismatches

1. **Board "New category"** (`BoardFilterBar.tsx:75,137`) shows on `tickets:write`; the server needs `projects:write`
   (`workspace/category_usecase.go:17`). Someone who writes tickets but not projects sees it and is refused.
2. **The project wizard** (`Router.tsx:66-67`) is gated on `projects:create` (`newProject`). Its steps need more:
   tests location, setup and repository attach need `projects:write` (`WizardRepositoryStep.tsx:50`,
   `WizardDoneStep.tsx:49,61`); create and update stack need `stacks:write` (`WizardServiceStep.tsx:110,161`,
   `WizardEnvStep.tsx:75`, `BranchRulesForm.tsx:63`); the first deploy needs `deploys:write`
   (`WizardServiceStep.tsx:119`, `WizardEnvStep.tsx:77`); the reach step needs `dns:write` (`WizardReachStep.tsx:125`);
   the import page needs `machines:read` and `stacks:write` (`ProjectWizardImportPage.tsx:47,58`).
3. **"Import from this machine"** (`MachineGroup.tsx:34`) gates on `projects:create`; the server needs `machines:read` to
   discover and `stacks:write` in the target project to import (`deploy/import.go:156`).
4. **"Attach repository"** on an unmanaged stack (`StackHeaderSection.tsx:66`) gates on `projects:create`; the wizard it
   opens ends in `stacks:write` and `projects:write` calls.
5. **New direct message** (`AREA_PERMISSION.newConversation` is `chat:write`): the server accepts plain membership when
   every other participant is an Owner (`chat/conversation_usecase.go:48-58`), so the web hides one case the server allows
   (`useOpenDirectMessage.tsx:20`).
6. **Ticket Pass and Fail** (`TicketTestSection.tsx:16`) is written as "anyone who can see the ticket", the server needs
   `tickets:write` (`tickets/testing.go:111,164`). Listed as UNGATED above; the comment documents the wrong rule.
7. **Role-wide check against per-resource rules.** `useHasPermission` answers from the role for the project in view
   (`WorkspaceHooks.tsx:57`), but the server resolves docs per doc (`docs/usecase.go:657-676`) and plays per play
   (`plays/run_launch.go:560`) with overwrites. Affected controls: `useDocRowActions.tsx:23-26`, `DocActionsMenu.tsx:26`,
   `DocDetail.tsx:53-54`, `PlayButton.tsx:36-37`, `PlaysMenu.tsx:30-32`, `RunItButton.tsx:15`, `PlayRow.tsx` (via
   `PlaySettingsSection.tsx`). A role-wide yes can still be refused on one doc or play. This is where stack rules (permission
   rework ticket 19) and instance-area rules (23, 24) will widen the gap.
8. **Anywhere versus one workspace.** DNS, runner, machine, connector and template actions are checked with
   `RequireAnywhere` / `HoldsAnywhere` on the server (`dns/usecase.go:146`, `runner/usecase.go:163`,
   `connectors/usecase.go:78`, `templates/usecase.go:257`). The Settings sections follow that through
   `instance_permissions` (`AccessHooks.tsx:42`), but the stack page's Exposures tab reads `useAreaAccess(project)`
   (`StackPage.tsx:40-41`), which is scoped to the project, so a role that holds `dns:read` elsewhere does not see the tab.
9. **Clone destinations.** Doc and memory clone are gated on the source permission only (`useDocRowActions.tsx:68`,
   `MemoryDetail.tsx:53`); the server also needs write on the destination project (`docs/usecase.go:155`,
   `memories/usecase.go:387`), which the dialog picks later.

## UNGATED counts per area

| Area | UNGATED | MISMATCH | GATED | N/A | Actions (not N/A) |
|---|---|---|---|---|---|
| Projects and board | 21 | 2 | 4 | 3 | 27 |
| Tickets | 12 | 0 | 3 | 3 | 15 |
| Docs | 1 | 0 | 14 | 2 | 15 |
| Memories | 1 (unverified) | 0 | 9 | 1 | 10 |
| Plays and auto plays | 3 | 0 | 13 | 1 | 16 |
| Stacks and deploys | 7 | 4 | 1 | 0 | 12 |
| Runners and machines | 3 | 1 | 0 | 2 | 4 |
| DNS, gateways, exposures | 5 | 1 | 1 | 1 | 7 |
| Topology | 6 | 0 | 1 | 0 | 7 |
| Chat and channels | 0 | 0 | 8 | 6 | 8 |
| Team, roles, invitations, workspaces | 2 | 0 | 11 | 0 | 13 |
| Instance settings, connectors, templates | 6 | 0 | 2 | 2 | 8 |
| Automations | 7 | 0 | 3 | 0 | 10 |
| **Total** | **74** | **8** | **70** | **21** | **152** |

Counts are per row of the tables above (the phone's rows are folded into their web rows; the phone table repeats
three of them). The totals include one unverified UNGATED (memories draft card).

## Things seen along the way

- Patterns for a shown-but-unavailable control already exist and are inconsistent: a disabled button with the reason as a
  title (`ConnectorCard.tsx:84-90`), a sentence naming who to ask (`ConnectCloudflareAction.tsx:28`), a disabled switch
  with no reason (`MemoryPinSwitch.tsx:22`, `AutoPlayRow.tsx:62`), and a read-only reason component
  (`components/team/TeamReadOnlyReason.tsx`). Everything else that is gated is simply hidden or returns `null`
  (`PlayButton.tsx:46`, `PlaysMenu.tsx:42`).
- The rule-8 source sentence reaches the user only after a refused call; no client reads it before the click.
- `useAreaAccess` with no project answers for "the project in view" on the web (`AccessHooks.tsx:12`) and for "any project
  they are in" on the phone (`native/src/hooks/WorkspaceHooks.tsx:50`).
- Unverified items: the memories draft card, `PickComputerMenu` visibility, the agent interrupt and answer routes, voice
  join, whether the phone sheet routes are gated, and whether `Show locked actions` can be answered from the data already
  loaded (ticket 03).
