# 01 — What a container's logs show

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

What does someone opening a container's logs expect to see, and how far back? The choices are a live tail that follows new lines (like `docker logs -f`), the last N lines as a snapshot, or searchable history across restarts and redeploys. Also: whether stdout and stderr are told apart, whether timestamps show, and whether the output of the container a redeploy replaced is still reachable.

This picks the source of truth. Docker's own log file (`docker logs`) needs nothing new on the host, but it goes with the container and has no search. Shipping container output into the OpenObserve that every install already runs gives search and retention, at the cost of a collector on every runner host and disk on the server.
