# 01 — What a container's logs show

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

What does someone opening a container's logs expect to see, and how far back? The choices are a live tail that follows new lines (like `docker logs -f`), the last N lines as a snapshot, or searchable history across restarts and redeploys. Also: whether stdout and stderr are told apart, whether timestamps show, and whether the output of the container a redeploy replaced is still reachable.

This picks the source of truth. Docker's own log file (`docker logs`) needs nothing new on the host, but it goes with the container and has no search. Shipping container output into the OpenObserve that every install already runs gives search and retention, at the cost of a collector on every runner host and disk on the server.

## Answer

Decided 2026-09-30 by the owner.

- **A live tail read from Docker.** Opening a container's logs shows its last
  lines, then follows new ones as the container prints them. The source is
  Docker's own log (`docker logs`), so the host needs nothing new.
- **Every line carries its timestamp and its stream.** Stderr is marked so
  it stands apart from stdout.
- **Nothing is stored in Nexul.** The output of a container that a redeploy
  replaced goes with it, and there is no search across containers.
- **Searchable history is deferred.** That means shipping container output
  into the install's OpenObserve. The trigger is someone needing to search
  across restarts or redeploys.
