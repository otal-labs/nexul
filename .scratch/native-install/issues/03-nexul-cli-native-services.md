# 03 — `nexul` as its own binary, installing native services on all three OSes

**Status:** resolved
**Type:** task
**Blocked by:** None — can start immediately (builds against the enrollment contract)

## Scope

- Split the binary: `./cmd/nexul` is the install and management command
  (`internal/install`), `./server/cmd` is `nexul-server` and only serves (no
  subcommands besides `version`). Downloads use the component asset names.
- Native service manager seam in `internal/install` with systemd, launchd
  (LaunchAgents for the user) and Windows SCM implementations: write, enable,
  start, stop, remove, status. `nexul service-host <unit>` implements the SCM
  protocol with `golang.org/x/sys/windows/svc` and supervises the unit's
  binary with its env.
- `nexul install [server]`: no compose. Download `nexul-server` and the pinned
  OpenObserve open-source build (`downloads.openobserve.ai/releases/openobserve/<v>/openobserve-<v>-<os>-<arch>.tar.gz`,
  `.zip` on Windows, sha256 pinned in code per target), create the `nexul`
  system user on Linux, write both units with the spec's env, keep the port
  prompt and check, pick a free localhost logs port (HTTP and gRPC both bound to
  127.0.0.1; verify OpenObserve's env names against its docs), wait for the
  server, then enroll and install the bundled runner and automations host named
  `instance` from the `data/enroll/*` code files. Docker (and Colima on macOS)
  is still ensured, for the runner.
- `nexul install runner|automations`: POST the enroll endpoint, write the unit
  directory (binary copy, `0600` env, credential file), install and start the
  unit. On Linux also the sudoers drop-in for the automations host.
- `nexul uninstall runner|automations <name> [--detach]`: best-effort
  self-remove call to the instance with the credential, stop and remove the
  unit and its directory. `nexul uninstall`: every unit, the server, OpenObserve,
  the command itself; `--purge` also the install directory.
- `nexul upgrade [--detach]`: swap itself (existing), then every unit's binary
  (and OpenObserve when its pin changed), restarting each.
- `nexul status`: every unit with kind, name, state, version.
- Delete `docker-compose.yml`, `docker-compose.desktop.yml`, `compose.go` and
  the compose code paths in `internal/install`. Keep `docker-compose.debug.yml`
  and `docker-compose.e2e.yml` working for development (they build from source).

## Acceptance

- Unit tests per OS through the fake `Exec` and path seams: exact unit file,
  plist and `sc.exe`/SCM calls; install, uninstall and upgrade sequences;
  detach per OS; port re-prompt; refusing an old compose install directory.
- `make lint`, `make coverage` green. Real-systemd run happens in ticket 06.
