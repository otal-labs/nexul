# macOS and Windows installs run the runner in a container, with data in Docker volumes

Amends ADR 0069, which covered Linux servers only; brings back ADR 0032's container exception for these hosts.

`nexul install` also runs on a Mac and on Windows, for trying Nexul on a laptop before it goes on a server. There,
Docker runs in a VM and there is no systemd, so the install writes `docker-compose.desktop.yml` next to the Linux
compose file and lists both in `COMPOSE_FILE`. The overlay runs the instance runner as the `nexul-runner` image, and
keeps the database and logs in Docker volumes instead of the install directory. A stack's checkout lives at a path
that the runner container and the Docker VM both see, which the runner now reports as its stack root when its
machine is first created: the install directory under the home folder on macOS, which Docker shares into its VM,
and a path inside the VM on Windows. On macOS the installer runs as the user and, when no Docker engine is present,
installs Colima, the Docker CLI and Compose with Homebrew (installing Homebrew itself if needed) and starts Colima
at login; an engine that is already there, Docker Desktop included, is used or started as it is. On Windows it only
checks for Docker Desktop and Compose and says what to install or start.

Why Docker volumes: SQLite's write-ahead log relies on shared memory mapping, which is unreliable on a folder the
host shares into Docker's VM. Why Colima and not Docker Desktop on macOS: Docker Desktop's install needs the app
opened and its licence accepted, which a script cannot do, while Colima installs and starts headless. Why not
install Docker Desktop on Windows: its installer wants WSL, a reboot and a licence acceptance.

The trade-off: a runner in a container cannot run `nexul upgrade` on its host, so the UI's upgrade on these
installs points at the terminal instead, and the Windows stack root lives inside Docker Desktop's VM rather than
on the Windows disk. Releases carry a third image, `nexul-runner`, for these installs.

Decided: 2026-09-25
