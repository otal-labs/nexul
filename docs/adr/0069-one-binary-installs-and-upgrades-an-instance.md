# One binary installs, upgrades and removes an instance

ADR 0072 extends this to macOS and Windows, where the runner is a container and data lives in Docker volumes.

Supersedes ADR 0054 and the bundled-runner exception in ADR 0032; supersedes ADR 0015 in part.

An instance is installed by the `nexul` binary itself: `curl -fsSL https://nexul.io/install.sh | sh` downloads the
binary for the host's CPU, checks it against the release's `checksums.txt`, and runs `nexul install`. That command
installs Docker Engine through Docker's own script when it is missing, installs the Compose plugin from the host's
package manager (falling back to Docker's documented manual install) when only Compose is missing, writes the
compose file embedded in the binary and a generated `.env` into one directory (`/data/nexul` by default, asked for
with its port), starts the stack, and installs the instance runner as the `nexul-runner` systemd service. The
same binary is the server (`nexul serve`, the container's command), with the web UI embedded, so the stack serves
the UI, the API, both WebSockets and MCP on one port and there is no nginx image. `nexul upgrade` swaps itself for
the target release's binary and re-runs under it, so the new release's compose file is the one written;
`nexul upgrade --version` rolls back the same way; `nexul uninstall` removes everything but the directory, and
`--purge` removes that too. The UI's Upgrade button asks the instance runner to start `nexul upgrade` in a
transient systemd unit, `nexul-upgrade`, which outlives the server restarting underneath it.

Why the runner moved to the host: it drives the host's Docker daemon and checks stacks out onto the host
filesystem, which ADR 0032 already said a container only reaches by mounting both through. The container runner
was a convenience of the clone-and-compose install; with an installer that can write a systemd unit, the
convenience is gone and so is the reason for the exception. The instance runner now updates itself like any
other runner (ADR 0052).

Rejected: keeping `git clone` plus a shell `install.sh`. It needs git on a bare server, clones the whole
repository to read one compose file, and grows a branch per distribution for Docker and Compose; a Go binary
carries that logic testably and knows its own OS and CPU. Also rejected: one binary for the server and the
runner. Remote machines only need the runner, and downloading the server and the web UI to every deploy target
is weight and attack surface for nothing. Also rejected: keeping the helper container for upgrades (ADR 0054).
It existed because the runner was a container in the project it was upgrading; a host runner has no such
problem, and `nexul upgrade` gives the UI path and the shell path one implementation.

The trade-off: the installer supports Linux with systemd. On macOS or Windows the documented path is the
single binary, `nexul serve`, with runners added separately. The `.env` keeps the logs credentials because
OpenObserve reads them only at its first boot, so a reinstall into a kept directory must, and does, reuse them.

Decided: 2026-09-25
