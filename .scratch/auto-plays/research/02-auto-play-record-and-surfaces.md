# 02: The auto play record, its permissions, and its HTTP and MCP surface

Answer to `issues/02-auto-play-record-and-surfaces.md`. Evidence first, then the recommendation, then open
risks.

## What the code does today

- A play is one row in `plays` (`migrations/0001_schema.sql:631`), workspace scoped, `ON DELETE CASCADE` from
  `workspaces`. Its one list-valued field, `excluded_project_ids`, is a JSON text column
  (`0001_schema.sql:640`) that the repo encodes and decodes. Foreign keys are on for every connection
  (`storage/db.go:12`), so cascades work.
- The plays domain already owns two repos, `Repo` and `TrailRepo` (`internal/plays/repo.go:11-41`), and splits
  its second entity across `run.go`, `run_handler.go`, `run_mcp.go`. A third entity fits the same split.
- The decisions check's switch is a column on `workspaces` (`migrations/0047_automations_workspace.sql:31`)
  read and written by the plays repo (`queries/plays.sql`, `GetDecisionsCheckEnabled`,
  `SetDecisionsCheckEnabled`; `repo.go:19-21`). That is the precedent for a workspace-level number the plays
  domain owns.
- HTTP: `handler.go:33-44` mounts every play route under `/api/workspaces/{workspaceID}/plays`, with literal
  segments (`applicable`, `decisions-check`) beside `{playID}`. `PATCH` is a full replace of the edited fields
  (`handler.go:132-137`); the MCP tool overlays a patch on top (`mcp.go:175-199`).
- MCP: 110 tools against a ceiling of 111 (`internal/mcp/surface_test.go:21`, ADR 0131). Plays have
  `play_list`, `play_create`, `play_update`, `play_delete` and `play_run`. `play_list` already appends an
  extra record only when the caller holds a second permission, and drops it silently on forbidden
  (`mcp.go:145-157`). A child entity carried on its parent's update has a precedent: `memory_update`'s
  `add_sources`, `update_sources`, `remove_sources` (`internal/memories/mcp.go:50-52`, types at 57-69).
- Permissions: one row per domain in `domainTable` (`internal/platform/permissions/permissions.go:136` for
  plays, `:160` for bots); the catalog, roles grid and token scopes derive from it, so a new row is the whole
  server-side vocabulary change. Backfilling a new bit into existing grants has a template:
  `migrations/0057_docs_lock_permission.sql` (roles, overwrites allow and deny, invitation grants including
  nested project access, integration installs, automation scopes).
- Access in the web: `web/src/models/Access.tsx:2-24` is the area map; bots, which also live inside another
  entity's settings rather than a route of their own, are three entries (`bots`, `editBots`, `deleteBots`,
  lines 17-19). The phone reads the shared table in `client-core/permissions.ts`. Plays are gated in `ConfigurationPage.tsx:14-16`
  and `AccessHooks.tsx:47`.
- Live push: a topic reaches a browser only if it is bridged in `livePushTopics` (`server/cmd/main.go`) and
  has a rule in `liveRules` (`server/cmd/live_audience.go`); `make live-topics` copies the rule names to the web, and a domain's live follower decides what a frame
  refreshes. `play.created`, `play.updated` and `play.deleted` are pushed under `plays:read`.
- Every catalogued topic is declared in its domain's `Topics()` with its payload type, and `make event-schemas`
  generates its schema and the SDK types (enforced by `internal/eventcatalog/contract_test.go`). It also needs
  an automation scope rule in `server/cmd/automation_scope.go` (enforced by `automation_scope_test.go`); play
  topics use `nestedWorkspaceScope("play")` and `workspaceScope`.
- What the conditions point at:
  - Ticket types, categories and status columns all belong to one project (`0001_schema.sql:128,342,359`),
    while a play belongs to the workspace. Templates already match ticket types across projects by name
    ignoring case (ADR 0103, "Clone is one operation").
  - Labels are free strings on the ticket (`ticket_labels`, `0001_schema.sql:138`), with no table of their own
    and no rename or delete operation (`tickets/usecase.go:581-660`).
  - Developer and tester are member logins on the ticket (`tickets/model.go:40-42`), resolved to a user id
    where a run needs one (`plays/decisions_check.go:219`).
- What deleting a referenced thing does today:
  - A project delete is refused while it has tickets (`workspace/usecase.go:317-333`) and publishes nothing; a
    play's `excluded_project_ids` keeps the dead id and nothing cleans it.
  - Status column and ticket type deletes are refused while tickets use them (`workspace/usecase.go:983-1005`,
    `:829`).
  - A category delete uncategorizes its tickets (`:606`).
  - A doc folder delete moves its docs to Main (ADR 0096).
  - So an id naming a deleted project, column, type, category or folder can never match a live ticket or doc
    again.
- One run per ticket at a time already holds for every run, manual or not: `refuseIfActive`
  (`plays/run.go:809-819`) refuses a start while any trail on the target is active (`model.go`,
  `TrailState.Active`).

## Recommendation

### Code home

Auto plays are a third entity of `internal/plays`, not a new domain package: the matcher (ticket 10) has to
start runs through `plays.Runner`, and domains never import each other. Files: `autoplay.go` (model and
validation), `autoplay_usecase.go`, `autoplay_handler.go`, `autoplay_mcp.go`, with `AutoPlayRepo` in `repo.go`
and `storage/autoplays_repo.go` over `queries/auto_plays.sql`. Copy the shape of the play CRUD
(`usecase.go:180-251`) and the trail repo wiring.

### Table: columns for what is queried, JSON for the trees

The matcher looks auto plays up by workspace and moment on every event, so those fields are columns with an
index. Conditions and priority are variable-shaped trees that SQL never filters on, so each is one JSON text
column, the way `excluded_project_ids` and grant sets already are.

```sql
-- 0078_auto_plays.sql (take the next free number when 09 is built)
CREATE TABLE auto_plays (
    id                  TEXT PRIMARY KEY,
    play_id             TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    workspace_id        TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled             INTEGER NOT NULL DEFAULT 0,
    moment              TEXT NOT NULL,
    moment_stage        TEXT,
    conditions          TEXT NOT NULL DEFAULT '{"match":"all","groups":[]}',
    priority            TEXT NOT NULL DEFAULT '{"rules":[],"otherwise":"normal"}',
    once_within_minutes INTEGER NOT NULL DEFAULT 0,
    run_on              TEXT NOT NULL DEFAULT 'developer',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL
);
-- Serves a play's settings page: its auto plays, oldest first.
CREATE INDEX idx_auto_plays_play ON auto_plays(play_id, created_at);
-- Serves the matcher: a workspace's live auto plays for one moment.
CREATE INDEX idx_auto_plays_moment ON auto_plays(workspace_id, moment) WHERE enabled = 1;
ALTER TABLE workspaces ADD COLUMN auto_play_daily_cap INTEGER NOT NULL DEFAULT 5;
```

The same migration backfills the new bits (see Permissions). No name column: the ticket line reads "Fix with
AI skipped: no longer unblocked", play label plus moment. Order is creation order, so there is no position
column.

### Go shape

```go
type Moment string   // values owned by ticket 01, e.g. "ticket.unblocked", "ticket.entered_stage", "doc.changed"
type RunOn string    // "developer", "tester", "causer"
type Level string    // "high", "normal", "low"
type Match string    // "all", "any"
type Field string    // type, project, stage, status, category, label, developer, tester,
                     // source_doc, linked_pr, blocked; doc plays: project, folder
type Op string       // "is", "is_not" (any of values), "set", "unset" (no values)

// Conditions is a stack, nested one level: the top matches all or any of its groups,
// each group all or any of its rules. A plain "all of these" is one group.
type Conditions struct {
	Match  Match   `json:"match"`
	Groups []Group `json:"groups"`
}
type Group struct {
	Match Match  `json:"match"`
	Rules []Rule `json:"rules"`
}
type Rule struct {
	Field  Field    `json:"field"`
	Op     Op       `json:"op"`
	Values []string `json:"values,omitempty"`
}

// Priority: the first rule whose group matches sets the level, else Otherwise.
type Priority struct {
	Rules     []PriorityRule `json:"rules"`
	Otherwise Level          `json:"otherwise"`
}
type PriorityRule struct {
	Level Level `json:"level"`
	When  Group `json:"when"`
}

type AutoPlay struct {
	ID                string     `json:"id"`
	PlayID            string     `json:"play_id"`
	WorkspaceID       string     `json:"workspace_id"`
	Enabled           bool       `json:"enabled"`
	Moment            Moment     `json:"moment"`
	MomentStage       *Stage     `json:"moment_stage"` // set only for ticket.entered_stage
	Conditions        Conditions `json:"conditions"`
	Priority          Priority   `json:"priority"`
	OnceWithinMinutes int        `json:"once_within_minutes"` // 0: no limit; else one run per target per window
	RunOn             RunOn      `json:"run_on"`
	CreatedBy         string     `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
```

Two levels always, rather than a recursive union: the "one level at most" rule then holds by the type, and
validation is two loops. `Validate` checks the enums, that a field fits the play's type (a doc play takes only
project and folder, an interview play takes no auto play), that `set`/`unset` carry no values and
`is`/`is_not` carry at least one, that `moment_stage` is set exactly for the stage moment, and a bound of 20
rules in total so a pasted tree cannot grow without limit. It does not check that referenced ids still exist,
since they can go stale at any time anyway (next section).

What each value holds:

- **project, status, folder:** ids. A status column is one project's, and stage is the cross-project version
  of it (CONTEXT, Stage).
- **type, category:** names, matched ignoring case. A workspace play spans projects, and "type is Bug" must
  hold in every project, including ones made later; ADR 0103 already matches types by name.
- **label:** the typed name.
- **developer, tester:** user ids, matched after resolving the ticket's login through the users seam the
  decisions check uses.

### When something it names is deleted

- **The play:** `ON DELETE CASCADE`, so its auto plays go with it. Ticket 10's queue table should reference
  `auto_plays(id) ON DELETE CASCADE` too, so a removed auto play never starts a run already waiting. Trails
  keep their loose `play_id`, as today. No `auto_play.deleted` events for the cascade; `play.deleted` tells
  the page.
- **A disabled play:** its auto plays stay but never fire; the matcher joins on `plays.enabled` and honours
  `excluded_project_ids`.
- **A project, status column, type, category, folder, or label:** nothing happens to the auto play. As shown
  above, a deleted id can never match a live ticket or doc again, so `is` on it is dead and `is_not` on it is
  always true, which is the honest reading. This is the same as `excluded_project_ids` today. The composer
  shows a stale value as "Deleted column" and the like, so the editor can remove it. No cleanup consumer.
- **A renamed type or category:** stops matching under its old name. The composer flags a name no project
  carries.

### Permissions

- `domainTable`: `{"autoplays", "auto plays", []string{read, write, delete}, AreaWorkspace}` right after
  plays, plus constants `AutoplaysRead`, `AutoplaysWrite`, `AutoplaysDelete`. Labels come out as "Read auto
  plays", "Create and update auto plays", "Delete auto plays" with no `verbLabel` entry.
- Use-case checks through `s.require` (`usecase.go:415`): list `autoplays:read`, create and update
  (including on/off) `autoplays:write`, delete `autoplays:delete`. The person a run lands on needs `plays:run`
  on the play, checked at run time (ticket 10), not at save.
- Backfill, in the 0057 style:
  - `autoplays:read` wherever `plays:read` is held.
  - `autoplays:read`, `write` and `delete` wherever `automations:write` is held, so whoever could switch the
    decisions check today still can once ticket 13 moves it onto an auto play.
  - Nobody gets write from `plays:write`: editing a prompt never used to make anything fire by itself. The
    Owner holds every bit through the bypass.
  - Ticket 07's ADR records this as an amendment to ADR 0057's list.
- Web: add `autoPlays: "autoplays:read"`, `editAutoPlays: "autoplays:write"` and `deleteAutoPlays:
  "autoplays:delete"` to `AREA_PERMISSION`, copying the bots trio, and to the native mirror. The Auto plays
  section on the play's settings page renders on `autoPlays`. The roles grid needs no web change; it renders
  the catalog.

### HTTP routes

On the existing plays handler (`handler.go:33`), full-replace bodies as play `PATCH` does today:

- `GET    /api/workspaces/{workspaceID}/plays/{playID}/auto-plays`
- `POST   /api/workspaces/{workspaceID}/plays/{playID}/auto-plays`
- `PATCH  /api/workspaces/{workspaceID}/plays/{playID}/auto-plays/{autoPlayID}`
- `DELETE /api/workspaces/{workspaceID}/plays/{playID}/auto-plays/{autoPlayID}`
- `GET|PATCH /api/workspaces/{workspaceID}/plays/auto-play-limits` with body `{"daily_cap_per_ticket": 5}`.
  This literal sits beside `{playID}`, the way `decisions-check` does, and replaces that route once ticket 13
  removes it. Reading takes `autoplays:read`, changing it `autoplays:write`, and the cap is bounded 1 to 50.

### MCP: extend `play_*`, no new tool

Extend `play_*` and add no tool. The server is at 110 of 111. An `autoplay_*` family would be at least three
tools (list, update with add and remove, and a delete) and would need an ADR raising the ceiling. That ADR
could not say "no existing tool could carry it": an auto play belongs to exactly one play, which is the
child-collection case in `practices/mcp.md` §4 and has a working precedent in `memory_update`'s sources.

- **`play_list`**, without `type`: each play carries `auto_plays` when the caller holds `autoplays:read`, left
  out otherwise. Copy the forbidden-drop in `mcp.go:150`. One extra query per call,
  `ListAutoPlaysByWorkspace`, grouped by play.
- **`play_update`** gains `add_auto_plays []autoPlayIn`, `update_auto_plays []autoPlayChangeIn`, and
  `remove_auto_plays []string`. `autoPlayChangeIn` takes an id plus pointer fields; conditions and priority
  are replaced whole when passed. Turning one on or off is `enabled` inside `update_auto_plays`, so the way
  back is as visible as the way in.
  - Today the tool always calls `s.Update`, which needs `plays:write` (`mcp.go:113-125`). It has to call it
    only when a play field was passed, so someone holding only `autoplays:write` can compose.
  - Order: play fields, then adds, updates, removes. It stops at the first failure and says what took effect
    (§8).
  - Removing a whole auto play rides on `update`. `play_update` already carries the conservative "may
    overwrite or delete" hint (only `Idempotent` and `Local` set, `mcp.go:112`), so no annotation changes.
    Removal is gated by `autoplays:delete` in the use-case, the same split `botwebhook_update` makes with
    `deleted` (ADR 0131).
- **The daily cap:** `workspace_update` gains `auto_play_daily_cap`, and `workspace_list` shows it. Both live
  in `internal/mcp/composite/workspaces.go`, which may import plays.
- The descriptions of `play_list` and `play_update` name the moments, fields and ops. The server instructions
  line about the decisions check (`play_update sets its decisions check`) changes with ticket 13.

### Events and live push

- Topics in `events.go`: `auto_play.created` and `auto_play.updated` with payload `{"auto_play": AutoPlay}`,
  and `auto_play.deleted` with `{"id", "play_id", "workspace_id"}`. Copy `CreatedEvent`, `UpdatedEvent` and
  `DeletedEvent` (`events.go:31-46`); write them through the outbox in the same transaction, as plays do.
- Each is declared in `Topics()` with its payload type, then `make event-schemas` (ADR 0137), and gets its
  automation scope (`nestedWorkspaceScope("auto_play")` for created and updated, `workspaceScope` for deleted).
- Live: bridge the three in `livePushTopics`, with one rule in `liveRules` that reads `workspace_id` from the top level or under `auto_play` and
  requires `autoplays:read` in that workspace, then `make live-topics`. In the web, the plays domain's live
  follower refreshes the auto plays query for them, and `play.deleted` does too
  (`practices/react-guide.md`, The live topic contract). The play topics themselves are already pushed under
  `plays:read`.

### Where the two caps live

- **Daily per-ticket cap, default 5:** a workspace setting, the `workspaces.auto_play_daily_cap` column above,
  read and written through the plays repo exactly as `decisions_check_enabled` is. A ticket lives in one
  workspace, so the workspace is the right owner. It is not an instance template: templates are layered texts
  that start a workspace off (ADR 0103), copied or followed, and a number needs neither. If owners later want
  one default for every new workspace, that is a separate decision.
- **Per-person concurrency:** a constant in `internal/plays` with ticket 03's default, no storage for now. The
  queue is one per person across every workspace, since a person has one computer, so a workspace column would
  let two workspaces disagree about the same person. If it ever becomes editable, it is the person's own
  setting beside their computer, never a workspace setting or a template.
- **Per-ticket concurrency (one auto run at a time):** already enforced for every run by `refuseIfActive`. No
  new storage.

## Open risks

- **Migration numbers:** 0078 is next, but tickets 08 and 10 (the queue table, maybe a moment column) and
  other open branches also add migrations. The number is taken at build time, and the backfill must run after
  the permission row exists in Go, never be edited afterwards.
- **Type and category by name** is a choice the composer has to make visible: one dropdown entry stands for
  "Bug" in every project. If the owner expects per-project picks, ticket 05's look settles it before 09 is
  built.
- **`show_when_stage`:** a ticket play shows its button in one stage only, while auto plays can fire in any
  (Fix with AI shows in progress, but "becomes unblocked" may happen in backlog). Recommended: auto plays
  ignore `show_when_stage` and their own conditions decide. Ticket 10 confirms this, or the composer must hint
  at it.
- **Unseen projects in conditions:** an editor may receive project ids they cannot open. With full-replace
  `PATCH`, the composer must round-trip values it cannot label, or a save silently drops them; that is the bug
  `unseenExclusions` (`usecase.go:405`) patches for plays. Prefer the round-trip in the web over another
  server-side merge.
- **Backfill reach:** giving `autoplays:write` and `delete` to everyone who holds `automations:write` may be
  wider than the owner wants. It keeps the decisions check switch working for those people, and ticket 07's
  ADR should say so explicitly.
- **`play_update`'s schema grows** by three nested arrays; walk "add an auto play" end to end before merging
  (§10).
- **Moment names:** stored strings are a contract once production rows hold them. They come from ticket 01 and
  must be fixed before 09's migration ships.
