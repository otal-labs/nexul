# Native install: every component a service, no Nexul containers

**Status:** ready-for-agent

## Problem statement

`nexul install` still runs the server, OpenObserve and the automations host as
Docker Compose containers, and on macOS and Windows the instance runner too.
Every one of those except OpenObserve is already a single cross-compiled binary,
so the images are release weight with no payoff, and they publish ports on the
host (`80`, `5080`) that collide with whatever else the owner runs on the same
Docker. Runners and automations hosts on other machines have no installer at
all: the Add runner dialog prints a foreground `./nexul-runner` command, the
automations host exists only as the bundled container, and nothing can remove
either once it is running. Every runner shares one instance secret, so a runner
removed in the UI would re-register on its next connect.

## Solution

Every component runs as a native process under the OS service manager
(systemd on Linux, launchd on macOS, the Service Control Manager on Windows).
Docker stays a prerequisite only for runners, because runners deploy stacks
with it; Nexul itself never runs in a container.

- **Four release binaries, named by component:** `nexul` (the small install and
  management command), `nexul-server` (the server with the web UI embedded),
  `nexul-runner`, and `nexul-automations` (the automations host compiled with
  `bun build --compile`). OpenObserve's open-source binary is downloaded from
  its vendor at a version and sha256 pinned in `nexul`.
- **One public port.** The server is the only listener on the chosen port.
  OpenObserve listens on localhost only and the server reverse-proxies it at
  `/openobserve/`. The install asks for the port and re-asks while it is taken
  (already true today; it stays).
- **One-line installers per component.** `install.sh` installs the server;
  `runner.sh` and `automations.sh` (and their `.ps1` twins) hand over to it
  with the component filled in:
  `curl -fsSL https://nexul.io/runner.sh | sh -s -- --server <url> --name <name> --code <code>`.
  The instance renders these commands for the Add runner and Add automations
  host dialogs.
- **Named runners and automations hosts, several per machine.** Each installs as
  its own service (`nexul-runner-<name>`, `nexul-automations-<name>`) with its
  own directory, binary copy and credential, so two on one machine never share
  state or race on a self-update.
- **Each host has its own credential.** The install command carries a one-time
  enrollment code; `nexul install` trades it for the host's own credential
  before the service first starts. The shared runner secret is removed.
- **Removal from the UI, MCP or the CLI.** Removing a runner or automations host
  revokes its credential. A connected host receives an uninstall instruction
  and removes its own service through `nexul uninstall … --detach`; a host that
  was offline is refused as removed when it reconnects and does the same.
  `nexul uninstall runner <name>` on the machine does the local cleanup and
  tells the instance.
- **Automations are placed on a host.** Each automation runs on exactly one
  automations host; new automations default to the instance's bundled host
  (named `instance`) and can be moved from the automation page or MCP.

## User stories

1. As an owner installing on my own server, I want Nexul to run as native
   services on one port I choose, so it does not add containers or ports to the
   Docker my other apps use.
2. As an owner, I want the logs UI at `/openobserve/` on the same address, so I
   do not open a second port.
3. As an owner, I want the Add runner dialog to give me one line that installs a
   named runner as a service on another machine, so adding capacity is a paste.
4. As an owner, I want to run two runners on one machine with different names,
   so the machine takes two jobs at a time.
5. As an owner, I want Remove on a runner to uninstall it from its machine when
   it is online, and refuse it forever when it is not, so a removed runner never
   comes back.
6. As an owner on the machine, I want `nexul uninstall runner <name>` to remove
   that runner and drop it from the instance, so I can clean up from the shell.
7. As an owner, I want the same install, list, remove and uninstall for
   automations hosts, and to choose which host runs each automation.
8. As an agent, I want MCP tools for enrolling, listing and removing runners and
   automations hosts and for moving an automation, so the agent path matches the
   browser.
9. As an owner, I want `nexul upgrade` and the Upgrade button to move every
   component on the host to the new release, on Linux, macOS and Windows.
10. As an owner, I want `nexul status` to list every Nexul service on the machine
    with its kind, name, state and version.

## Contracts

These are fixed so tickets can be built in parallel. Change one only by
updating this section in the same change.

### Release assets

Raw binaries `<binary>-<os>-<arch>[.exe]` for linux/darwin × amd64/arm64 and
windows/amd64, all listed in `checksums.txt`:
`nexul-*` (CLI, main `./cmd/nexul`), `nexul-server-*` (main `./server/cmd`,
`-tags embed`), `nexul-runner-*` (main `./runner/cmd`), `nexul-automations-*`
(built by `bun run --cwd automations build:binaries`, which writes
`automations/dist/nexul-automations-<os>-<arch>[.exe]` for all five targets).
No container images are released.

### `nexul` command surface

```
nexul install [server] [--dir D] [--port P] [--version V] [--yes]
nexul install runner --server URL --name N --code C [--stack-root DIR] [--git-token T] [--version V] [--yes]
nexul install automations --server URL --name N --code C [--version V] [--yes]
nexul uninstall [--purge] [--yes]                 # the server and every unit on this machine
nexul uninstall runner <name> [--detach] [--yes]
nexul uninstall automations <name> [--detach] [--yes]
nexul upgrade [--version V] [--detach]
nexul status
nexul service-host <unit>                          # Windows only: runs a unit under the SCM
```

`--detach` starts the same command outside the calling service's process tree
(Linux: `systemd-run --unit <unique> --collect`; macOS and Windows: a detached
process that survives the caller's service stopping) and returns at once. It
is what a runner or automations host calls to remove or upgrade itself. On
Linux a non-root caller re-runs through `sudo -n`; the install writes a
sudoers drop-in allowing exactly `nexul uninstall automations <name> --detach`
for the automations service user.

### Services and layout

| Unit | Linux (systemd) | Runs as (Linux) |
|---|---|---|
| server | `nexul-server` | `nexul` system user, `AmbientCapabilities=CAP_NET_BIND_SERVICE` |
| logs | `nexul-openobserve` | `nexul`, `MemoryMax=1G` plus OpenObserve's own cache caps |
| runner | `nexul-runner-<name>` | root (drives Docker, runs `nexul upgrade`) |
| automations host | `nexul-automations-<name>` | `nexul` |

macOS uses LaunchAgents for the installing user (label `io.nexul.<unit>`);
the installer already runs as the user there. Windows uses services registered
with `nexul service-host <unit>` as the binary path; `install.ps1` elevates.
The server install keeps its directory (`/data/nexul` on Linux, `~/nexul`
elsewhere) with `data/`, `logs/` and `stacks/`. Each runner and automations
host gets its own unit directory holding its binary copy, a `0600` env file and
its credential file: `/opt/nexul/<kind>-<name>/` on Linux,
`~/Library/Application Support/nexul/<kind>-<name>/` on macOS,
`%ProgramData%\Nexul\<kind>-<name>\` on Windows. Names match
`^[a-z0-9][a-z0-9-]{0,31}$`.

### Environment

- Server: `NEXUL_HTTP_ADDR=:<port>`, `NEXUL_DB_PATH`, `NEXUL_LOGS_URL`
  (`http://127.0.0.1:<logs-port>`; when set the server proxies `/openobserve/`
  to it, path unchanged, since OpenObserve runs with `ZO_BASE_URI=/openobserve`),
  and the existing `NEXUL_OTLP_*` pointed at the local OpenObserve. The logs
  port is picked free on localhost by the installer and is not a flag.
- Runner: `NEXUL_SERVER_URL` (the instance's `http(s)://` base; the runner
  derives `ws(s)://…/ws/runner`), `NEXUL_CREDENTIAL_FILE`, `NEXUL_RUNNER_NAME`
  (its unit name, used only to remove itself), `NEXUL_STACK_ROOT`,
  `NEXUL_GIT_TOKEN`, `NEXUL_CTL` (path to `nexul`). `NEXUL_SERVER_WS`,
  `NEXUL_RUNNER_SECRET*`, `NEXUL_RUNNER_ID` and `NEXUL_MACHINE` are removed;
  identity, name and machine come from the credential's server record.
- Automations host: `NEXUL_SERVER_URL`, `NEXUL_CREDENTIAL_FILE`,
  `NEXUL_AUTOMATIONS_HOST_NAME`, `NEXUL_CTL`. The tokens file and
  `NEXUL_AUTOMATIONS_HOST_TOKENS_PATH` / `NEXUL_AUTOMATIONS_TOKENS_PATH` are
  removed.

### Enrollment and credentials

- Enrollment code: `nxe_` + 32 random bytes base64url, stored as a sha256 hash,
  single use, one hour, bound to a kind and a name (and optionally a machine
  for runners).
- Credential: `nxr_…` (runner) or `nxa_…` (automations host), 32 random bytes,
  stored as a sha256 hash, no expiry, revoked by removal. Revoked credentials
  are kept as tombstones so a returning host is told it was removed rather than
  that its credential is unknown.
- Shared helpers (mint, hash, compare) live in `internal/platform/hostcred`;
  each domain owns its own tables.
- Bundled hosts: while no runner (or automations host) named `instance` is
  enrolled, the server writes a fresh code at boot to
  `<data dir>/enroll/runner-instance` (or `automations-instance`), `0600`, valid
  24 hours, and deletes the file once that host enrolls. `nexul install` reads
  it to enroll the bundled pair.

### HTTP (in the OpenAPI spec) and MCP

Runners:
- `POST /api/runners/enrollments` (instance admin) `{name, machine?}` →
  `{code, expires_at, commands: {unix, windows}}`. MCP `host_create` with
  `kind: "runner"`.
- `POST /api/runners/enroll` (public) `{code, name, os, arch, version, stack_root}`
  → `{id, name, machine, credential}`; `401 invalid_code` for unknown, used or
  expired codes, `409 name_mismatch` when `name` is not the code's (the
  standard error body, `{"message", "code"}`).
- `DELETE /api/runners/{id}` (instance admin): revoke, delete the record, send
  the uninstall frame if connected. MCP `host_delete` with `kind: "runner"`.
- `POST /api/runners/self/remove` (runner credential as Bearer): the same, used
  by `nexul uninstall runner`.
- `/ws/runner` authenticates with `Authorization: Bearer <credential>`. A
  revoked credential is refused `401` with body `{"error":"runner_removed"}`;
  the runner then calls `nexul uninstall runner <name> --detach` and exits.
- `GET /api/runners/download/{target}` accepts a runner credential.

Automations hosts: the same five shapes under `/api/automation-hosts` (MCP
`host_create` and `host_delete` with `kind: "automations"`; hosts are listed
per machine by `machine_list`, so each host reports its machine),
plus `GET /api/automation-hosts/self/assignments` (host credential) returning
the enabled automations placed on that host, each with the token its worker
uses. That token is host-scoped: the automation auth path accepts it for that
automation only while the automation is placed on that host and the host's
credential is live. The host's refusal body is `{"error":"automations_host_removed"}`.
An automation's host is set through the existing automation update endpoint and
MCP tool (`host_id`; null means the bundled `instance` host).

The MCP surface stays under its 100-tool budget (ADR 0068):
`automation_token_create` folds into `automation_update` as `rotate_token`.

Rendered commands (instance URL from settings, version pinned to the server's
own release; a dev build omits the pin):

```
curl -fsSL https://nexul.io/runner.sh | NEXUL_VERSION=<v> sh -s -- --server <url> --name <name> --code <code>
$env:NEXUL_VERSION='<v>'; & ([scriptblock]::Create((irm https://nexul.io/runner.ps1))) --server <url> --name <name> --code <code>
```

Automations hosts use `automations.sh` and `automations.ps1` the same way.

## Out of scope

- Nexul sign-in in front of `/openobserve/` (OpenObserve keeps its own login).
- Per-runner authorisation (which stacks a runner may deploy).
- Renaming the SDK's own `nexul` bin (`init|dev|push|pull`), which clashes with
  the install command's name. Tracked as its own follow-up.
- Migrating existing compose installs. There is no production data yet; the new
  installer refuses a directory that still holds a compose install and says to
  run the old `nexul uninstall` first.

## Testing decisions

Test what an operator sees: the unit files and commands written for each OS
(asserted as argument lists and exact file content through the existing fake
`Exec` and path seams in `internal/install`), a connection accepted or refused,
which hosts are affected by a removal. Enrollment and revocation are covered at
the HTTP and WebSocket handler seams with real SQLite. The end-to-end check is a
full install from a locally built release on the `nexul-box` test machine:
server, bundled runner and automations host come up on one chosen port, a second
named runner installs from the rendered command, and Remove uninstalls it.
