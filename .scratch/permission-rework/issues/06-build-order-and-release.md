# 06 — Build order and the release that wipes

**Type:** grilling
**Status:** resolved
**Blocked by:** 02, 03, 04, 05, 08

## Question

How is the build sliced, given that the wipe happens once, at upgrade? The release that wipes has to carry
opt-in projects, the setup grid and several roles together, or users land in a broken state. What ships
before it (for example the evaluator that names sources), and what after it (folders, stacks, channels,
computers, instance areas)? Which of the original 14 tickets survive, merge or go?

## Answer

Decided on 2026-10-11 as a technical call the owner delegated. It reads ticket 08 as "the role editor opens in a
dialog with a Save button". One question changes what a user sees, and it closes this answer. The build doesn't
wait on it, because the default is stated.

### 1. The release that wipes, and how master stays releasable

**What lands in one commit on master.** Each piece below, without the others, leaves an upgraded install either
broken or reading more than it does today:

- **The wipe migration.**
- **The flip to opt-in projects**, with the "In this project", "Create projects" and "Manage this project's access"
  permissions.
- **Several roles and zero roles**, in Team, invitations and `/me`.
- **The Access page**, with the one-time setup grid, the redirect and the waiting state.
- **On projects:** People in this project, the project's panel and the wizard's "Who's in it" step.
- **Role targets** in the doc and play panels.
- **Channel hiding** through the setup grid.
- **The role editor's new sections.**
- **Removals:** Team overrides, Project access, Every project and Restricted, in Go, web, client-core and the phone.
- **Invitations** that give one or more roles and nothing per person.
- **HTTP and MCP field changes.**
- **Pills locked across the project line.** These follow from the flip, and only the tooltip ships earlier.

What breaks without the rest:

- Opt-in without the grid leaves members with no projects and no fast way back.
- Removing Team overrides before several roles exist leaves no way to give one person more.
- Dropping Restricted without channel hiding lets a client read public channels.
- Project creation that doesn't list the creator makes a new project not found to whoever made it.

**The strategy: a stack whose last PR switches it on, built in two halves.** Of the three options it is the stack.
The one change is that the part of the stack that can't be neutral waits on a short branch instead of on master.

- **First half, on master (slices 1 to 8).** Every piece that changes nobody's access lands as a normal PR and
  reaches the daily betas. Most of the risk lives here: the new evaluator, role targets, role sets, and the batched
  reads for project and channel rules. Production runs these for days before anything is wiped.
- **Second half, on the branch `permission-switch` (slices 9 to 17).** It holds everything that changes access or
  removes something, one session per slice. It lands on master as **one squash-merged PR**, the last of the stack.
  Betas are cut only from master, so the wipe migration is nowhere a beta can see until that merge, and the merge
  brings all of it at once. No partial wipe can reach a beta.

Rules for a first-half PR. A PR that breaks one of them is not merged.

1. **No answer changes on real data.** A PR that touches the evaluator or a permission table states an empty
   `access.Matrix` diff from a copy of the production database, taking master against the branch, with the row
   counts compared. The copy is a read-only `.backup` kept outside the repo.
2. **Migrations only add and copy.** A role set of one, and person rows copied as `target = 'person'`. Nothing is
   deleted and no meaning changes.
3. **The second half can't be reached.** Before the switch:
   - no HTTP route or MCP field writes a role rule, several roles, a project rule for a member who isn't
     Restricted, or a channel rule;
   - no screen from the release that wipes is mounted;
   - the reads that serve all of these are built, used and guarded.
4. **Every beta stays whole.** A beta cut after any first-half PR is a working Nexul.

Rules for the switch branch.

- **When it starts.** It is cut from master once slices 1 to 5 have merged, and it lives for days, not weeks.
- **Each slice is a green commit.** The local CI mirror passes on it, and the orchestrator checks it against the
  practices before the next slice starts. Parallel sessions work in their own worktrees, branched from it, and
  merge back.
- **It stays current.** Every session starts by rebasing onto master, so conflicts stay small. While the branch is
  open, no other effort is sent into these places:
  - `internal/access`, `internal/tenancy`, and the channel reads in `internal/chat`;
  - `web/src/components/{access,team,member,settings}`.
- **Migration numbers come last.** The branch's migrations get their numbers at the final rebase, as the next free
  numbers on master. Their tests refer to them by file.
- **The final PR.** It passes the box test (section 5), and its title is the release note: "Rebuild permissions on
  roles: projects are opt-in, people hold several roles, and an Owner sets who's in each project again after
  upgrading".

Rejected:

- **One long-lived branch for everything.** The neutral half would sit unmerged and untested by betas for the whole
  effort. It would conflict with every domain that calls the access service and land as one huge diff. The switch
  branch holds only what can't be neutral, and only for days.
- **A feature flag.** A flag can't gate a forward-only migration that deletes rows. Both evaluators would run on
  production, which doubles every parity and statement guard. An install setting breaks "install needs no env
  vars", and a per-workspace flag is a second backend in all but name.

### 2. What ships before the wipe

Slices 1 to 8 change nobody's access.

- **Live:**
  - forbidden errors name their source;
  - the explain path, and Check access;
  - the three-state permissions panel on docs and plays, with today's person targets;
  - hold-what-you-give on doc and play rules;
  - the row menu, and the "You don't have access" tooltip on locked pills;
  - the role editor as a dialog, with today's sections.
- **Built and guarded, not reachable yet:**
  - role-targeted rules in storage and the evaluator;
  - each member's role set, always a set of one until the switch;
  - project rules read for every member, still applied only to Restricted members;
  - channel rules read on every channel path.

Rules saved through the new doc and play panels before the switch are person rules, and the wipe deletes them like
any other sharing. Each creator's own row survives. Production has no such rows today.

### 3. What ships after it

Each of these is a normal PR on master once the switch has merged. All of them are new abilities, so none changes
anyone's access on upgrade:

- See on each play;
- doc folder rules;
- stack rules;
- channel rules beyond the setup grid, with a posting permission;
- computer sharing;
- instance-area rules, and their panel.

Not sliced here, because the map hasn't specified them yet:

- how the phone shows a project or play that disappeared, beyond empty lists;
- Request access on a locked pill.

### 4. Slices

Each slice is one working session, except 3 and 9, which may need two sessions on one branch. Every slice keeps the
MCP tool count unchanged and updates its own guide page and `CONTEXT.md` lines, as AGENTS.md's "Hit every surface"
asks. "Replaces" names the original permission-overrides ticket it stands in for.

**Before the switch: master, live or built but unreachable**

1. **Every answer names its source.**
   - **Scope:**
     - `decide` returns whether the action is held, plus the source as ids. Every caller still reads only whether it
       is held.
     - `Service.Explain` covers doc, play and project, as today's rules mean them. For each action it returns the
       answer, plus the answer this entity would give the chosen target if its own rule were Fallback, which feeds
       the panel's Fallback line.
     - `GET /api/permissions/explain`, and an optional `user_id` on `permission_overwrite_list` that returns
       `effective`.
     - Forbidden errors carry the source sentence. Not found carries none.
     - `access.Matrix`, run on a database copy named by `NEXUL_ACCESS_MATRIX_DB`, covering the workspace, projects,
       docs, plays and channels.
   - **Blocks:** 2 and 6.
   - **Guards:** the new parity test `TestExplain_AnswersWhatTheCheckAnswers`, over the `list_paging_test.go`
     fixture's viewers and every catalog action. Every `TestStatements_*` count stays the same.
   - **Accept:**
     - table-driven `TestDecide_Source`;
     - explaining another person's access takes the entity's manage permission, over HTTP and MCP;
     - an unreadable doc is not found;
     - the Matrix golden on the fixture.
   - **Replaces:** original 01. Its Check access UI moves to 6.

2. **Rules can name a role.**
   - **Scope:**
     - A migration rebuilds `permission_overwrites` with `target` (`person`, `role`; no Everyone), a nullable
       `user_id` and `role_id`, each cascading, the check that exactly one is set, and the unique index. Every row
       is copied as `person`.
     - The evaluator reads person and held-role rows in one memoised statement. Allow wins between role rules, a
       person rule beats the roles, and the nearer entity beats the farther one.
     - `DocsWith` stays bounded by the number of rules, and `ListUsers` counts role rows.
     - `access.grant.changed` gains `target` and `role_id` beside `user_id`, and a role row's audience is the
       role's holders. The web and phone followers refetch `/me` on a frame that names no person, the way
       `role.updated` does.
     - No writer creates a role row.
   - **Blocks:** 3, 4, 5 and 9.
   - **Guards:**
     - `list_paging_test.go` gains role rows on docs;
     - new `TestStatements_RoleRulesCostTheSameAtAnyLength`, with rules on every row against none;
     - `liveRequests.test.tsx` and the phone's `requestCounts.test.tsx` gain a frame that names no person.
   - **Accept:**
     - an empty Matrix diff on the production copy;
     - `TestDecide_RoleRules`;
     - deleting a role or an account deletes its rows;
     - `make event-schemas` and `make live-topics` are clean, and the contract only grows;
     - `doc_list`, `ticket_list` and one live frame, statements and median time, before and after on the heavy copy.
   - **Replaces:** original 02, without Everyone and without the trim migration.

3. **A member holds a set of roles (still one each).**
   - **Scope:**
     - A migration adds `member_roles` and backfills it from `workspace_members.role_id`, then rebuilds
       `workspace_members` without `role_id`. `restricted` stays until the switch.
     - These all read the set, as a union:
       - the 13 sqlc queries, and the tenancy use-cases;
       - `memberRole` and the memo;
       - `roles.requireAction`, through `access.WorkspacePermissions`;
       - the invitation SQL (`creatorAccess`, `ListManageableInvitationHeaders`);
       - `ListActiveOwnerIDs`.
     - Every writer still gives exactly one role, and deleting a role someone holds is still refused, now in the
       use-case.
     - `/me`, Team, `account_list` and `workspace_list` gain a `roles` list beside the single fields.
   - **Blocks:** 4 and 9.
   - **Guards:** `TestStatements_LiveFramesCostOneReadPerPersonOnceWarm` keeps its count. A member's roles cost two
     statements or fewer, joined.
   - **Accept:**
     - an empty Matrix diff;
     - `TestRoleSet_EveryEvaluatorAgrees`: a member given two roles in a test gets the same union from
       `WorkspacePermissions`, `requireAction`, the invitation SQL and the owner list;
     - the existing tenancy and invitation suites stay green with only their fixtures changed.
   - **Replaces:** nothing. It is new.

4. **Project rules are read for every member.**
   - **Scope:**
     - `layers()`, `ProjectsWith` and the memo read the project rules for the person and their roles, for every
       member, in one statement per workspace. They still apply only to Restricted members.
     - `ListProjectPeople` becomes one SQL query, and the workspace project list uses the `ProjectsWith` batch.
   - **Blocks:** 9.
   - **Guards:**
     - The fixtures of `access_memo_test.go`, `list_paging_test.go` (`listParityFixture`) and
       `permissions_integration_test.go` (`seedPermFixture`) list every non-Owner reader in their projects. The
       answers don't change yet, so after the flip the guards can't pass for the wrong reason.
     - `TestProjectsWith_ReadsARestrictedMembersAccessOnce` becomes `…ReadsEveryMembersProjectRulesOnce`.
     - New `TestStatements_ProjectRulesCostTheSameAtAnyLength`.
     - The project list and the project-people query join `TestStatements_ListsCostTheSameAtAnyLength`.
     - `TestRows_ListToolsReadOnlyTheirPage` gains a non-Owner viewer for `doc_list`.
   - **Accept:** an empty Matrix diff, and `ticket_list` and `doc_list` numbers on the heavy copy.
   - **Replaces:** original 04, the read half.

5. **Channels read their rules.**
   - **Scope:**
     - `readsChannel` consults channel rules for the person and their roles, batched across the conversation
       list, unread counts, search, voice join and live frames, under the memo fence.
     - The Owner is never denied. Private channels and Restricted behave as today. No writer.
   - **Blocks:** 9 and 13.
   - **Guards:** the "chat conversation list" and "chat unread counts" rows of
     `TestStatements_ListsCostTheSameAtAnyLength` with a rule on every channel, and the live-frames guard, keep their
     counts.
   - **Accept:**
     - an empty Matrix diff;
     - a channel denied in a test is not found in lists, unread counts, search and live frames;
     - the Owner still sees it.
   - **Replaces:** original 11, the read half.

6. **The permissions panel on docs and plays.**
   - **Scope:**
     - The panel as locked in ticket 03, with the owner's three-button control, Check access and the Fallback
       source lines. People targets only for now.
     - It replaces `PermissionsForm` on docs, and "Exclude users" on plays, where the Run row shows today's
       exclusions as Deny.
     - `SetGrants` and `SetPlayGrants` take a `state` (Allow, Fallback, Deny). HTTP `PUT /api/permissions` and
       `permission_overwrite_update` gain `state` beside `grant`, which keeps its meaning.
     - Hold-what-you-give applies to doc and play rules.
   - **Blocks:** 7 and 14.
   - **Guards:** `PermissionsForm.test.tsx` is replaced by the panel's tests.
   - **Accept:**
     - a person set to Deny and another to Allow on one doc works end to end, and Check access shows both sources;
     - another open page updates live;
     - `TestGrants_NobodyAllowsWhatTheyLack` for docs and plays, and a Deny is always accepted;
     - verified at 768, 1024 and 1440px with screenshots, and the react-guide self-review passes.
   - **Replaces:** original 03, and the person part of 06.

7. **The row menu, and locked pills.**
   - **Scope:**
     - `components/ui/context-menu.tsx` through the shadcn CLI, with the `cn` import fixed. It goes on doc, folder,
       project, channel and play rows in the sidebar and lists, with `RowActions` ⋯ wherever a row lacks one.
     - "Permissions…" shows on docs and plays only. Projects get it in 14, folders in 18, channels in 20.
     - The "You don't have access" tooltip on locked doc and ticket pills, on the web. Check the phone's rendering.
   - **Accept:**
     - right-click opens at the pointer, and the row keeps `bg-accent` while the menu is open;
     - ⋯ shows the same items, and the keyboard works;
     - "Permissions…" shows only to people who can manage the entity's access;
     - verified at 768, 1024 and 1440px.
   - **Replaces:** nothing. It is new.

8. **The role editor as a dialog.**
   - **Scope:** the dialog locked in ticket 08, carrying today's sections (Workspace, Every project). The roles list
     on Configuration → Roles is unchanged.
   - **Blocked by:** ticket 08's lock.
   - **Blocks:** 15.
   - **Accept:** a role is created, edited and cloned through the dialog. Save stays in view as the body scrolls.
     Verified at 768, 1024 and 1440px.
   - **Replaces:** nothing. It is new.

**The switch: on `permission-switch`, landing as one PR**

9. **The flip to opt-in.**
   - **Scope:**
     - A non-Owner sees a project only when its rules list them or a role they hold. Zero roles is allowed.
     - Restricted branches leave `internal/access`, `internal/chat` (`refuseRestricted`, `IsRestricted`) and
       `internal/memories`.
     - New permissions: "Create projects" (workspace level), "Manage this project's access", and "In this project"
       (a rule-only permission). A migration grants "Create projects" to roles holding `projects:write`, and
       "Manage this project's access" to roles holding `members:write`, the way migration 0046 did.
     - Project create lists its creator and their roles, in the same write, over HTTP and MCP.
     - Writers now accept role targets on docs, plays and projects.
     - The "Kept out" rule: a person denied See on a project.
   - **Blocks:** 10.
   - **Guards:**
     - `TestIntegration_PermissionSuitesAnswerTheSameWithAMemo` and `…LiveAudienceMemo_FollowsEveryChange`: the
       fixture's `uClient` becomes a client listed in one project.
     - `TestListTools_ShowWhatTheRowByRowCheckShowed`: `restrictedClient` becomes an unlisted member and a member
       with zero roles. `doc_list` parity becomes "listed equals openable", because the no-query path filters by
       `DocsWith`.
     - The "docs list" and "a play's queued runs" rows of `TestStatements_ListsCostTheSameAtAnyLength` keep their
       counts.
     - `TestCanDocs_ReadsEachLayerOnceForTheLot` moves out of `restricted_test.go`.
     - The parity test from slice 1 keeps running over the new fixture.
   - **Accept:**
     - the access and chat tests of research section 10 are deleted or rewritten;
     - a zero-role member reads People, public channels, DMs and the inbox, and nothing in a project;
     - pills from an unlisted project show their full content, locked;
     - a creator's new project opens for them.
   - **Replaces:** the rest of original 04. Original 08 is dropped, because opt-in hides projects.

10. **The wipe migration, and the tenancy and invitation removals.**
    - **Scope, the migration:**
      - It deletes `workspace` rows, `project` rows, `play` person rows, doc person rows other than the creator's,
        and doc rows whose doc is gone (research 4.1).
      - It keeps the creator's doc row as stored (option a in research 4.2).
      - It drops `workspace_members.restricted`.
      - It moves `invitation_grants` to a role set, with the role copied across. It strips the per-person columns
        and drops them; open links keep their role and lose the rest.
      - It writes the per-workspace setup marker (unset where the workspace has projects) and the channel
        carry-over (see the question below).
      - `workspace_invites` keeps one role, a set of one on joining.
    - **Scope, the code:**
      - `SetMemberOverrides`, `project_access.go`, Every project, and the invitation per-person validation are
        removed;
      - a held role may be deleted (the cascade leaves zero roles);
      - removing a role you couldn't give is refused.
    - **Blocks:** 11.
    - **Accept:**
      - `TestMigration_TheWipe`: a fixture at the previous schema with every row kind (Owner, Team allow and deny,
        Restricted with Project access, dormant project rows, creator and non-creator doc rows, a doc row of a
        deleted doc, play exclusions, and the four shapes of open invitation), upgraded, asserting exactly which
        rows survive;
      - the tenancy and invitation tests of research section 10 are deleted or rewritten.
    - **Replaces:** nothing. Original 04's trim migration is dropped.

11. **The HTTP and MCP contract.**
    - **Scope, HTTP.** The API only grows (ADR 0082), so every old field keeps answering:
      - `/me` keeps `restricted`, now true for every non-Owner, with `projects` sent to everyone. A phone build that
        predates the switch then reads per-project permissions correctly with no update.
      - `role_name` is the roles joined, beside `roles`.
      - The member routes take `role_ids` beside `role_id`.
      - `allow`, `deny`, `every_project` and `project_access` read as empty. Writing a non-empty value is refused
        with a message naming what replaced it.
    - **Scope, MCP.** Its schemas change:
      - `account_update` gets `add_roles` and `remove_roles` (the practice's child-collection fields), and loses
        `role_id`, `allow`, `deny`, `every_project` and `project_access`;
      - `account_list` and `workspace_list` carry `roles`;
      - `invitation_create` takes `roles` (at least one) and loses its per-person fields;
      - `permission_overwrite_list` and `_update` accept `project` and `channel`, `target` and `role_ids`;
      - descriptions are rewritten, and `rg -w` finds the old fields in play instructions, the `nexul-memory` skill
        (its version moves) and `api-and-tokens.md`.
    - **Blocks:** 12 to 16.
    - **Guards:** the MCP surface test keeps 110 tools. `liveRequests.test.tsx`'s member-updated and `role.updated`
      counts are updated.
    - **Accept:** a `/me` that predates the switch, fed to the current client-core in a test, gives the same answers
      as the new shape. Each removed MCP field fails with a schema error.
    - **Replaces:** nothing. It is new.

12. **Several roles on screen.**
    - **Scope:** the ticket 04 look:
      - Team rows with role chips;
      - the person dialog's roles checklist with "Can't give" and the zero-roles warning, replacing the role
        select, `TeamOverridesForm` and `TeamProjectAccess`;
      - the invitation checklist (at least one role) and the invitation page's chips;
      - the zero-roles empty state and sidebar;
      - Your roles, and the account row.
    - **Accept:** a role ticked or unticked updates the other person's sidebar live. The removed components and their
      tests are gone. Verified at 768, 1024 and 1440px.
    - **Replaces:** the Team half of original 05. Original 07 is dropped, because Team's workspace rules are gone.

13. **The Access page and the one-time setup.**
    - **Scope:** the ticket 05 look:
      - `/:workspace/access` and its sidebar row;
      - the first-visit redirect and the "Set up access" row;
      - the "Projects being set up" waiting state for non-Owners;
      - both grids;
      - one save request that writes project rules and channel deny rules in one transaction, with the marker set on
        Finish;
      - the live refresh for members;
      - the Access page's tabs after setup.
    - **Guards:** new `TestStatements_AccessGridCostsTheSameAtAnyLength`, covering the grid's read and the Finish
      write at 3 projects and at 30.
    - **Accept:**
      - Finish with every project left empty still works;
      - a member's projects appear without a refresh;
      - leaving with unsaved ticks asks first;
      - verified at 768, 1024 and 1440px, including the 30 × 10 case.
    - **Replaces:** the hiding part of original 11.

14. **People in this project, the project panel, and "Who's in it".**
    - **Scope:**
      - The "People in this project" card, with its "Kept out" heading, replaces `ProjectPeopleAccessSection`.
      - The project's panel, and "Permissions…" on project rows.
      - The doc and play panels gain role targets.
      - The doc panel refuses anyone outside the project, with "Add <name> to <project> first".
      - The danger zone's "N people lose this project" replaces `RestrictedMembersLoseAccess`.
      - The wizard's "Who's in it" step, pre-ticked with the creator's roles, in `SetupSteps` and the web.
    - **Accept:**
      - a role added on the card opens the project for its holders live;
      - a person kept out disappears from their own sidebar;
      - projects made before the step count it as done;
      - verified at 768, 1024 and 1440px.
    - **Replaces:** the project half of original 05, and the role part of 06.

15. **The role editor's new sections, and the web's last Restricted reads.**
    - **Scope:**
      - The dialog from slice 8 gets a workspace section and an "In projects" section in place of "Every project",
        with "Create projects" and "Manage this project's access".
      - `useNewConversationDialogs` and `ChannelSettingsCard` stop reading `restricted`.
      - `RoleSettingsSection.test.tsx` and `ChannelSettings.test.tsx` are rewritten.
    - **Accept:** verified at 768, 1024 and 1440px. `rg -w restricted web/src` finds nothing.
    - **Replaces:** nothing. It is new.

16. **client-core and the phone.**
    - **Scope:**
      - `MyWorkspaceInfo` carries `roles`, and `projectPermissions` reads `projects` for every member;
      - the phone's locked docs leave lists;
      - `useLiveEvents` handles grant frames that name no person;
      - the Restricted cases in `AreaGate.test.tsx` and `BoardScreen.test.tsx` are rewritten.
    - **Guards:** `native/src/hooks/requestCounts.test.tsx` and `useLiveEvents.test.tsx`.
    - **Accept:** checked through the T3 Device panel. An unlisted member sees no projects, and chat works.
    - **Replaces:** nothing. It is new.

17. **The upgrade test, and the PR.**
    - **Scope:**
      - rebase onto master, and number the migrations;
      - run the local CI mirror and the box test in section 5;
      - ADR 0148 moves to accepted as ticket 07 rewrote it;
      - check the guide pages (`people-and-access`, `projects-and-repositories`, `plays`, `api-and-tokens`) against
        what was built;
      - open the PR with the release-note title above, and its screenshots;
      - squash-merge it.
    - **Accept:** every box-test assertion holds and is recorded in the PR body.

**After the switch: master, normal PRs**

18. **Doc folder rules.**
    - **Scope:**
      - original 09, without Everyone or Restricted;
      - "Permissions…" on folder rows;
      - a folder panel limited to doc actions, which refuses anyone outside the project.
    - **Guards:** new `TestStatements_FolderRulesCostTheSameAtAnyLength`, and `list_paging_test.go` gains folder
      rules.
    - **Replaces:** original 09.
19. **Stack rules.**
    - **Scope:** original 10, without Everyone.
    - **Guards:** new `TestStatements_StackRulesCostTheSameAtAnyLength`.
    - **Replaces:** original 10.
20. **Channel rules and posting.**
    - **Scope:**
      - the channel panel, with role and person targets, in the channel's settings and its row menu;
      - `channels:post`, with a migration granting it to `chat:write` holders.
    - **Accept:** a production-copy Matrix diff that is empty for posting.
    - **Replaces:** the rest of original 11.
21. **Computer sharing.**
    - **Scope:** original 12 as written: owner-only, named people, never memoised. It doesn't depend on the switch,
      and comes after it so the wipe never touches a computer row.
    - **Replaces:** original 12.
22. **See on each play.**
    - **Scope:**
      - `plays:read` set per play;
      - every path in research section 8 answers not found for a play you can't see;
      - a run's starter reads their own trail;
      - `playFrame`, `playRunFrame` and `autoPlayFrame` follow See;
      - the play panel gains a "See this play" row.
    - **Guards:** "a play's queued runs" and the plays list keep their counts with rules on every play. New
      `TestStatements_PlayRulesCostTheSameAtAnyLength`.
    - **Replaces:** the rest of original 06.
23. **Instance-area rules.**
    - **Scope:** original 13, with Restricted lines removed. Several roles mean Allow wins between role rules.
    - **Accept:** `/api/auth/me` costs the same statements with rules on every area as with none.
    - **Replaces:** original 13.
24. **The access panel on instance areas.**
    - **Scope:** original 14, with role and person targets only.
    - **Replaces:** original 14.

**Order:**

- Before the switch: 1 → 2 → 3 → (4, 5). Slice 6 follows 1, 7 follows 6, and 8 follows ticket 08.
- The switch: 9 → 10 → 11 → (12, 13, 14, 15, 16 in parallel) → 17.
- After the switch: 18 to 24 in any order. Slice 20 needs 13, and 22 needs 14.

### 5. Upgrade testing for the wipe

**In CI:** `TestMigration_TheWipe` (slice 10), plus every rewritten guard and parity test.

**On `nexul-box`, before the switch PR merges.** The wipe is forward-only, and a fix-up PR after a beta would be too
late for any install that already upgraded.

1. **Start from the previous release.** Restore the box to its `clean` snapshot, then run
   `nexul install --version <the newest beta on master>`.
2. **Load production-shaped data, in two parts.**
   - A copy of production's database, taken with `sqlite3 .backup` over a read-only connection. It goes onto the box
     in place of the fresh database, so the previous release's own migrations bring it up to date. It never enters
     the repo or leaves the owner's machines, and it is deleted afterwards.
   - A synthetic workspace, Acme (alice, bob, lena, sam, and roles including Client), seeded through the previous
     release's own HTTP and MCP, so the rows have the shapes that release writes. It carries every case production
     lacks:
     - Team allow and deny;
     - a Restricted client with Project access on one of two projects, and a private channel;
     - dormant project rows;
     - a doc shared with someone other than its creator, and a doc whose creator left;
     - a play exclusion;
     - open invitations that are role-only, carry allow and deny, and carry Restricted with Project access;
     - a Developer's access token and automation;
     - an auto play switched on for a Developer;
     - a play run in progress.
3. **Take the "before" picture.** Copy the database off and run `access.Matrix` on it at the previous release's
   commit. Snapshot the box.
4. **Upgrade.** Stop the services, put the switch branch's binaries (a local build, since no release is cut) in
   place of the installed ones, and start. Restart once more to show the migration doesn't run twice.
5. **Before any Owner acts, check:**
   - **Rows.** They match `TestMigration_TheWipe`'s expectations. On the production copy: the 2 project rows and the
     Restricted flag are gone, and the 32 creator rows remain.
   - **Nothing widens.** In the after-Matrix, every answer a non-Owner holds was also held before. The only exception
     is the channel answer in the question below, if it goes that way. Owners are unchanged.
   - **Owners.** They are redirected once to the setup page. Every project reads "Owners only", and "Set up access"
     shows in the sidebar.
   - **Members.** They see "Your projects are on their way", and chat, DMs and the inbox work.
   - **Things acting as a Developer.** The token and the automation list no projects and return no 500s. The auto
     play records "may not run here". The T3 link is hidden. The running play reattaches, or ends cleanly.
   - **Old invitation links.** Redeeming one gives its role and no projects.
   - **Pills.** A pill from an unlisted project shows its content, locked, with the tooltip.
   - **The phone.** Through the T3 Device panel, both the current phone release and a build that predates it open
     with no crash, no projects, and working chat.
6. **Finish setup** on both workspaces. Members' projects appear live. The after-Finish Matrix matches what the
   grid says.
7. **Rollback drill.** Restore the backup from step 3 and reinstall the previous release. It boots on its old data.
   That is the only way back from a forward-only wipe, so it gets proven once.

**After the merge.** Once the daily beta that carries the switch is cut, repeat steps 1 to 6 from a fresh `clean`
snapshot, on the production copy only. This time upgrade through `nexul upgrade --version <that beta>`, the real
path. Then take a `.backup` of production's database right before nexul.otal.dev is upgraded.

### Settled by the owner's earlier answers

1. **The one client who is kept out of public channels today:** wiped like everything else. She sees the public
   channels until an Owner hides them on the Access page. The owner decided twice that per-person access is wiped
   and that hiding public channels is the Owner's job. The upgrade writes no per-person channel rule.
