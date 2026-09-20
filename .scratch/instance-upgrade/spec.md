# Instance upgrade from the UI

**Status:** built; end-to-end run on a real install pending public GHCR packages (see spec "Out of scope" and TODO)

## Problem statement

Upgrading a running Nexul instance means logging into the host and running
`git pull && ./install.sh`. The web UI already knows a newer release exists
(the version badge in the sidebar) but can only link to the docs. Every
comparable self-hosted platform offers an "Upgrade" button; ours sends the
operator to a shell.

## Solution

An **Upgrade** action on the instance settings page, with MCP parity, that
brings the compose stack up on the newest release of the running channel
without anyone touching the host.

The mechanism goes through the bundled `instance` runner, which already
mounts the Docker socket and runs inside the compose project it will
upgrade. It cannot run `docker compose up` itself, because that recreates
its own container mid-command. Instead it starts a **one-shot helper
container** (the runner's own image, which ships `docker-cli`, the compose
plugin, and `git`) that is not part of the compose project and therefore
survives the restart. The helper pulls the release's images and runs
`docker compose up -d` against the same project, working directory, and
compose files the running stack was started from. Those come from the
runner container's own compose labels (`com.docker.compose.project`,
`...project.working_dir`, `...project.config_files`), so no install
directory has to be configured anywhere.

The server persists an upgrade record before dispatching, and resolves it
when it comes back up: if the booted version equals the target, the record
completes; if the same old version is still running after a timeout, it
fails with a hint to read the helper container's logs.

This replaces the earlier idea of running the upgrade through a host
runner. The helper container works on every stock install with nothing
extra installed, and it keeps the runner model's rule that the server never
holds credentials to the host.

## Out of scope

- Single-binary installs: the action reports "not a compose install" and the
  UI shows the manual instructions. A re-exec self-update for the binary is
  a later effort.
- Private registries: the helper's Docker client has no registry
  credentials, so images must be pullable without login. GHCR packages go
  public with the repository.
- Pulling compose-file changes (`git pull`). The UI upgrade updates images
  only. A release whose compose file changed says so in its notes, and
  `git pull && ./install.sh` remains the full path.
- Downgrades and pinning. The target is always the channel's newest release.
- Automatic upgrades on a schedule.

## User stories

1. As an instance admin, I want to see the running version, the channel,
   and the newest release side by side on the instance settings page, so I
   know whether an upgrade is waiting.
2. As an instance admin, I want an **Upgrade to vX** button with a
   confirmation, so one click moves the instance to the newest release.
3. As an instance admin, I want the page to show that the upgrade is in
   progress, then that the app is back on the new version, so I am not
   guessing whether it worked.
4. As an instance admin, I want a failed upgrade to say what to check
   (`docker logs nexul-upgrade` on the host), so I can recover without
   reading server code.
5. As an agent working through MCP, I want an `instance_upgrade` tool that
   does exactly what the button does, recorded with `:mcp` provenance (ADR
   0049), so the agent path is not weaker than the browser path.
6. As an operator on a single-binary install, I want the button replaced by
   the manual instructions rather than an error after clicking.

## Contracts

### HTTP (registered in the OpenAPI spec, ADR 0045)

`GET /api/instance/upgrade` (instance admin only)

```json
{
  "version": "v0.2.0-beta-003",
  "channel": "beta",
  "latest": { "version": "v0.2.0-beta-004", "url": "https://github.com/..." },
  "update_available": true,
  "can_upgrade": true,
  "reason": "",
  "upgrade": null
}
```

`reason` is non-empty when `can_upgrade` is false, one of:
`"dev build"`, `"already on the newest release"`, `"release lookup failed"`,
`"instance runner is not connected"`, `"instance runner is busy"`,
`"an upgrade is already in progress"`.

`upgrade` is the most recent record or null:

```json
{
  "id": "…", "from_version": "v0.2.0-beta-003", "to_version": "v0.2.0-beta-004",
  "status": "pending", "error": "", "requested_by": "user-id",
  "created_at": "…", "updated_at": "…"
}
```

`status` is `pending` (record written, frame sent), `started` (helper
container running, stack restart imminent), `completed`, or `failed`.

`POST /api/instance/upgrade` (instance admin only), empty body. Returns
`202` with the new record, or `409` `{ "reason": "…" }` with the same
reasons as above.

### Runner protocol (internal/runner/protocol.go)

- Server → runner `assign_upgrade`: `id` (upgrade id), `version` (release
  tag, e.g. `v0.2.0-beta-004`). Occupies the runner's single job slot like
  `assign_deploy`.
- Runner → server `upgrade_progress`: `id`, `log` (one line).
- Runner → server `upgrade_result`: `id`, `status` (`started` or `failed`),
  `error`, `log`. `started` means the helper container is running; the
  runner is about to be recreated, so no further frames follow.

### Event topic

`instance.upgrade_changed` with the record above as payload, published on
every status change, registered in the event catalog (ADR 0044). The web
client invalidates the upgrade and version queries on it.

### Storage

Table `instance_upgrades` (id, from_version, to_version, status, error,
requested_by, runner_id, created_at, updated_at), owned by the runner
domain, queries in `queries/runner.sql` via sqlc (ADR 0009).

## Runner behaviour (`Executor.Upgrade`)

1. `docker inspect <own container>` (the container ID is the hostname
   inside a container). Missing compose labels → `failed` with
   `"instance runner is not managed by docker compose"`.
2. Read project name, working dir, config files, and image from the
   inspect output.
3. `docker rm -f nexul-upgrade` (ignore errors) so one stopped helper per
   host keeps the last attempt's logs.
4. `docker run -d --name nexul-upgrade -v /var/run/docker.sock:/var/run/docker.sock -v <wd>:<wd> -w <wd> -e NEXUL_VERSION=<tag without v> --entrypoint sh <image> -c "<script>"`
   where the script is `set -e; docker compose -p <project> -f <f1> [-f <f2>…] pull; docker compose -p <project> -f … up -d --remove-orphans`.
5. Send `upgrade_progress` for each step, then `upgrade_result started`
   with the helper container id in `log`.

The helper is deliberately not `--rm`: its logs are the only trace if the
stack does not come back on the new version.

## Server behaviour

- `RequestUpgrade(ctx, actor)`: checks the reasons above in order, writes
  the record as `pending`, publishes `instance.upgrade_changed`, and sends
  `assign_upgrade` to the connected runner whose id is `instance`.
- Inbound `upgrade_progress` appends to the server log; `upgrade_result`
  moves the record to `started` or `failed`.
- A runner disconnect while the record is `pending` fails it
  ("runner disconnected"). A disconnect while `started` is expected and
  changes nothing.
- On boot (after migrations), `ResolvePendingUpgrade()`: any `pending` or
  `started` record whose `to_version` equals the running version becomes
  `completed`; any other unresolved record older than 15 minutes becomes
  `failed` with `"instance is still on <version>; run docker logs nexul-upgrade on the host"`.
  The same check runs lazily inside `UpgradeStatus()` so a stuck record
  fails even without a reboot.

## Web behaviour

An **Instance version** section at the top of the instance settings panel
(`InstanceSettingsPanel.tsx`), built from the settings kit (`SettingsCard`):

- Facts row: running version, channel, newest release (link to the release).
- Button `Upgrade to vX`, disabled with the `reason` as helper text when
  `can_upgrade` is false. Confirmation dialog before `POST`.
- While `pending`/`started`: the button becomes a status line ("Upgrading
  to vX… the app will reconnect on its own") and the section polls
  `GET /api/instance/upgrade` every 5 seconds. The existing reconnect toast
  (`notifyIfServerUpdated`) prompts the reload once the new server is up.
- `completed`: "Upgraded to vX at <time>". `failed`: the error, plus the
  `docker logs nexul-upgrade` hint.
- The sidebar version badge links to this section instead of the docs.

Mono Console rules apply: no new hues, status through the existing badge
language.

## Acceptance criteria

- On the debug stack, clicking Upgrade with a newer beta available starts
  the helper, the stack comes back on that beta, and the record reads
  `completed` with no manual step.
- `POST` on a dev build returns 409 `dev build`; on a stack whose instance
  runner is disconnected returns 409 with that reason.
- The MCP tool `instance_upgrade` produces the same record with
  `requested_by` suffixed `:mcp`.
- A record left `started` on a server that boots on the old version fails
  after 15 minutes with the log hint.
- Go coverage gate and web lint/tests stay green; OpenAPI lists both routes.
