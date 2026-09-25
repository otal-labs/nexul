# The runner is a host binary, not a container

ADR 0069 removes the exception below: the instance runner is a host systemd service too.

Everything else Nexul ships runs in the compose stack, so a
containerized runner would be the consistent choice — but a runner's whole
job is to drive the host's Docker daemon and check repositories out onto the
host filesystem, both of which a container only reaches by mounting the
socket and the stack root through from outside. Go cross-compiles to a single
static binary per OS/arch, so the host binary is a `curl` and a `chmod` with
no runtime to install. The bundled `instance` runner still runs as a
container for convenience; that is the exception, and it only works because
its stack root is bind-mounted at the *same path* it reports to the server —
the host daemon resolves a checkout's compose bind mounts, not the runner's
own filesystem.

Decided: 2026-07-25
