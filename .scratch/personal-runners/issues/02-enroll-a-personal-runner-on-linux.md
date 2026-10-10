# 02 — Add a computer: enroll a personal runner on Linux with one command

**Status:** resolved

**Blocked by:** 01

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md` (sections 4 to 8, 11),
`practices/testing.md`, ADRs 0073, 0074, 0087, 0102, the spec (Installing a personal runner, Data model,
Access and privacy).

## What to build

- Migration: `runner_enrollment_codes.owner_user_id` and `.computer_id` (defaults `''`).
- Pairing use-case `AddComputer(ctx, userID, name)`: creates a computer row waiting for its runner and
  mints a personal enrollment code through a runner seam bound to the caller and that computer (1 hour,
  runner name `computer-<8 random>`), inside a signed token (server, code, computer, `exp`), returning the
  one-line install command `curl -fsSL https://nexul.io/computer.sh | sh -s -- <token>`. No permission bit; a Restricted
  member may add one. `POST /api/pairing/computers/enrollments`; `computer_create` now returns the
  install commands (its `port` is ignored, removed in ticket 14).
- Runner enrollment for a personal code: records owner and computer, makes no machine row.
- `website/public/computer.sh <token>` (the only command a person runs) and the engine it reaches,
  `nexul install computer --token`: as the user, refuses root,
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

- [x] On nexul-box: the command from `POST /api/pairing/computers/enrollments`, run as a normal user, leaves
      a running `nexul-computer` user unit whose runner connects; the computer row shows it connected.
- [x] `TestRunners_ListNeverShowsAPersonalRunner` (HTTP, MCP and the machine list, for a caller holding every
      bit and for a workspace Owner).
- [x] `TestComputers_AnotherPersonsComputerIsNotFound` across list, get, rename and enrollment, over HTTP and
      MCP, for a workspace Owner and an every-bit role: 404, never 403.
- [x] `TestLiveAudience_PersonalRunnerChangesReachOnlyTheOwner` and `TestComputerEvents_AreMembersOnly`
      (no `computer.*` or `runner.personal_changed` event reaches an integration webhook or an automation).
- [x] A personal code cannot enroll a deploy runner and a deploy code cannot enroll a personal runner.
- [x] `nexul install computer` refuses root and installs nothing as root; its unit tests follow
      `internal/install`'s fakes.
- [x] `website/src/content/docs/docs/guide/mcp-server.md` describes the new `computer_create` result.

## Comments

Built: migration 0093 (`runner_enrollment_codes.owner_user_id` and `.computer_id`, and `idx_runners_computer` made
unique, so a computer has one runner), `AddComputer` / `EnrollComputer` / `RenameComputer` in
`internal/pairing/computer_usecase.go`, `CreatePersonalEnrollment` in `internal/runner/enrollment.go`,
`nexul install computer` and `nexul uninstall computer` in `internal/install/computer.go`, `website/public/computer.sh`.

- The command is one argument (owner, 2026-10-10): `curl -fsSL https://nexul.io/computer.sh | sh -s -- <token>`. The
  token is an HS256 JWT signed with a key derived from the instance's auth secret (`server`, `code`, `computer`,
  `exp`); the enroll endpoint takes `token` in place of `code` and checks signature, expiry, the computer it names
  and the code's single use. A personal code is accepted only inside a token (`computer_code` otherwise), and a token
  holding a deploy code is refused (`runner_code`). The script reads only `server`, to say where it connects.
- The server renders the command against `NEXUL_SITE_URL` and `NEXUL_RELEASE_URL` when set (both optional), carried
  as `NEXUL_INSTALL_URL` / `NEXUL_RELEASE_URL` for the scripts; that is how the box test ran the command unchanged.
- A computer added without a name takes the hostname the installer reports: the runner publishes
  `runner.personal_changed` with `state: enrolled` and `hostname`, and a pairing consumer names an unnamed row.
- `GET /api/pairing/computers` and `computer_list` gain `runner: {connected, last_seen}`; the web pairing follower
  refetches computers on `runner.personal_changed` (the dialog itself is ticket 06).
- Rename over MCP is `computer_pair` with `id` and `name` and no token (tool budget is full); ticket 03 reshapes
  `computer_pair` anyway. `computer_create` takes an optional `id` for a fresh command while no runner is enrolled.
- `runners:delete` (`DELETE /api/runners/{id}`, `host_delete`) answers not found for a personal runner.
- A personal runner publishes no `runner.heartbeat` either, since that topic reaches automations and webhooks.
- A personal runner removes itself with `nexul uninstall computer --detach`, a transient unit of the person's own
  systemd user manager (`systemd-run --user`).
- Lingering: `loginctl enable-linger` as the person, then `sudo -n`; never prompts. On Ubuntu 24.04 a person on
  `su -` or SSH is refused by polkit, so without passwordless sudo it stays off and the installer says how to turn
  it on; the runner then runs only while they are logged in. Open question 3 (asking for the sudo password once)
  is still the owner's.
- Box check (nexul-box-enroll, v0.3.99-beta.1 built from this branch): the command from the endpoint, run as
  `alice` (no sudo) through `su - alice`, installed `~/.local/bin/nexul`, the `nexul-computer` user unit and the
  runner; the row read `name: nexul-box-enroll, runner: {connected: true}`; after `loginctl enable-linger alice`
  the unit started on its own, and a `kill -9` was restarted by systemd (`NRestarts=1`). A second workspace Owner
  got 404 / not found on list-by-id, setup, rename and a fresh code over HTTP and MCP, an empty computer list, and
  no `computer-` runner in `/api/runners` or `machine_list`. The same command as root stopped before downloading.
- For 03: `Computer.ServerURL` is empty on a runner computer until it pairs; `Computer.Session()` and `pair` need
  the relay address there. For 04: `RemoveRunner` hides personal runners, so retiring one goes through a new seam.
