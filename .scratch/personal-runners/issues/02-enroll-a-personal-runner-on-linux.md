# 02 — Add a computer: enroll a personal runner on Linux with one command

**Status:** ready-for-agent

**Blocked by:** 01

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md` (sections 4 to 8, 11),
`practices/testing.md`, ADRs 0073, 0074, 0087, 0102, the spec (Installing a personal runner, Data model,
Access and privacy).

## What to build

- Migration: `runner_enrollments.owner_user_id` and `.computer_id` (defaults `''`).
- Pairing use-case `AddComputer(ctx, userID, name)`: creates a computer row waiting for its runner and
  mints a personal enrollment code through a runner seam bound to the caller and that computer (1 hour,
  runner name `computer-<8 random>`), returning the install commands. No permission bit; a Restricted
  member may add one. `POST /api/pairing/computers/enrollments`; `computer_create` now returns the
  install commands (its `port` is ignored, removed in ticket 14).
- Runner enrollment for a personal code: records owner and computer, makes no machine row.
- `nexul install computer --server --code` and `website/public/computer.sh`: as the user, refuses root,
  binary in `~/.local/bin`, systemd **user** unit `nexul-computer`, lingering as the spec says, no Docker
  check, `Restart=always` so the service manager restarts a crashed runner. `install.sh` installs into `~/.local/bin` without sudo for this subcommand. `nexul uninstall
  computer` removes it.
- Personal runners publish `runner.personal_changed` (owner-only live audience, members-only) and never
  `runner.connected`/`runner.disconnected`. Every runner list, the machine list and the import wizard read
  `WHERE owner_user_id = ''`.
- Every existing `computer.*` payload gains `members_only: true`, then `make event-schemas` and
  `make live-topics`.
- `PATCH /api/pairing/computers/{id}` renames a computer; the row's name defaults to the hostname the
  runner reports at enrollment.

## Acceptance criteria

- [ ] On nexul-box: the command from `POST /api/pairing/computers/enrollments`, run as a normal user, leaves
      a running `nexul-computer` user unit whose runner connects; the computer row shows it connected.
- [ ] `TestRunners_ListNeverShowsAPersonalRunner` (HTTP, MCP and the machine list, for a caller holding every
      bit and for a workspace Owner).
- [ ] `TestComputers_AnotherPersonsComputerIsNotFound` across list, get, rename and enrollment, over HTTP and
      MCP, for a workspace Owner and an every-bit role: 404, never 403.
- [ ] `TestLiveAudience_PersonalRunnerChangesReachOnlyTheOwner` and `TestComputerEvents_AreMembersOnly`
      (no `computer.*` or `runner.personal_changed` event reaches an integration webhook or an automation).
- [ ] A personal code cannot enroll a deploy runner and a deploy code cannot enroll a personal runner.
- [ ] `nexul install computer` refuses root and installs nothing as root; its unit tests follow
      `internal/install`'s fakes.
- [ ] `website/src/content/docs/docs/guide/mcp-server.md` describes the new `computer_create` result.
