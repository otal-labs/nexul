# An instance upgrades itself through a helper container the bundled runner starts

Upgrading a compose install from the UI means restarting the very containers that would run the upgrade. The
server holds no credentials to its host (the runner model exists so it never has to), and the bundled
`instance` runner is itself a container in the project being upgraded, so neither can run `docker compose up`
directly: the runner's own container is recreated mid-command and the job dies with it. Instead the runner
starts a one-shot helper container, `nexul-upgrade`, from its own image (which already ships the docker CLI,
the compose plugin, and git). The helper is not part of the compose project, so it survives the restart. It
pulls the release's images and runs `docker compose up -d` against the same project name, working directory,
and compose files the running stack was started from, all read from the runner container's compose labels, so
no install directory has to be configured anywhere. The server writes an upgrade record before dispatching and
resolves it on boot: the booted version matching the target completes it; the old version still running after
fifteen minutes fails it with a pointer to `docker logs nexul-upgrade`.

Rejected: a host runner (the standalone binary on the instance host) running the install script. It works, but
it asks every operator to install a second runner on the instance host before the button does anything, and the
stock install already has a runner with the Docker socket. Also rejected: a detached shell (`nohup … &`) started
by the runner; inside a container that shell dies with the container, so it only works for the host-binary
shape. Also rejected: `git pull` in the helper. The helper runs as root, and a root-owned object in the
operator's checkout breaks their next manual pull; a release that changes the compose file says so in its notes
and keeps the manual path.

The trade-off: there is no live log after the restart begins. The runner reports `started` and then goes away,
so the UI shows progress by polling the version until it changes, and a failure is inferred from a timeout
rather than observed. The helper is deliberately not removed after it exits, so one stopped container per host
holds the last attempt's output; the next upgrade removes it first.

Decided: 2026-09-15
