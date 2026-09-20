# Each service gets its own Dockerfile even though compose runs them collapsed

Server, runner, and web each have a Dockerfile with `release` and `debug`
targets, while the compose stack runs everything on one host. The boundaries
are drawn in the build now precisely because they are cheap to draw now:
pulling a service onto its own machine later becomes a compose edit rather
than a packaging project. The cost is three build files to maintain for a
deployment shape that currently needs one.

Decided: 2026-07-24
