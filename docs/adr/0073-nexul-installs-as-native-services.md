# Nexul installs as native services; no part of Nexul runs in a container

Supersedes ADR 0072, the compose stack in ADR 0069, and what was left of ADR 0015.

`nexul install` puts every component on the machine as a process under the OS service manager: systemd on Linux,
launchd on macOS, the Service Control Manager on Windows. A release is four binaries named by component: `nexul`
(the small install and management command), `nexul-server` (the server with the web UI embedded), `nexul-runner`,
and `nexul-automations` (the automations host compiled with `bun build --compile`). OpenObserve's open-source binary
is downloaded from its vendor at a version and sha256 pinned in `nexul`. The server is the only listener on the
port the owner picks; OpenObserve listens on localhost and the server proxies it at `/openobserve/`. Each runner
and automations host is its own named service (`nexul-runner-<name>`, `nexul-automations-<name>`) with its own
directory, binary copy and credential, so a machine can run several, and `runner.sh` and `automations.sh` install
one on another machine. `nexul upgrade` swaps every unit's binary on the machine; `nexul uninstall` removes one
host or all of them. Docker stays a prerequisite only for runners, because runners deploy stacks with it.

Why: every component but OpenObserve was already one cross-compiled binary, so the images were release weight with
no payoff. The containers also published ports on the owner's Docker (`80`, `5080`), which collided with whatever
else ran there, and they forced a second install shape on macOS and Windows (ADR 0072) in which the instance
runner could not upgrade its own host.

Rejected: keeping the compose stack and only moving the runner out, which is what ADR 0069 did; it left two
shapes to install, upgrade and debug, and the port collisions. Also rejected: one binary for the command and the
server. The command is what a remote machine downloads to install a runner, and it has no reason to carry the
server and the web UI. Also rejected: a process supervisor Nexul ships
itself, to get one shape on every OS. The OS service managers already restart, log and start at boot, and a
supervisor of our own is one more process to install and keep alive.

The trade-offs:

- Three service managers to write for and test: systemd units, LaunchAgent plists, and Windows services whose
  binary path is `nexul service-host <unit>`, because a Windows service must answer the Service Control Manager
  and the component binaries do not.
- On Linux the server, OpenObserve and automations hosts run as a `nexul` system user, so the install creates one;
  runners run as root because they drive Docker and run `nexul upgrade`. An automations host removes itself
  through a sudoers drop-in that allows exactly its own `nexul uninstall automations <name> --detach`.
- On macOS the services are LaunchAgents of the installing user, so they run while that user is logged in. That
  suits a laptop trying Nexul and is the reason a Mac is not the documented server.
- A runner or automations host that removes or upgrades itself must not be killed halfway by its own service
  stopping, so `nexul … --detach` re-runs the command outside the caller's process tree (`systemd-run` on Linux,
  a detached process elsewhere).
- The debug compose stack stays, for development only; nothing released is an image.
- A directory that still holds a compose install is refused with the command that removes it. There is no
  migration, because there is no production data yet.

Decided: 2026-09-27
