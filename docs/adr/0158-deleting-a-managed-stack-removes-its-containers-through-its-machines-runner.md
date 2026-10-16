# Deleting a managed stack removes its containers through its machine's runner; an adopted stack's stay

Deleting a stack removed its rows, its exposures and its services, and the confirm promised its containers were removed.
Nothing told the runner, so a deleted cloudflared gateway kept running and its tunnel then refused to delete because it
still had active connections.

Decision: `service.deleted` carries the stack's machine, slug, strategy and whether it is managed, and the runner domain
consumes it. For a managed stack it sends the machine's runner a `stack_teardown` frame: `docker compose -p <slug> down
--remove-orphans` for a compose stack, `docker rm -f <slug>` for a run stack. The compose call names only the project, so it
needs neither the checkout nor the compose file; Docker finds the project's containers by label. Volumes, networks and the
checkout stay. The result frame is logged, and a failure is not retried because the stack record that would carry the retry
is gone.

- **An adopted (unmanaged) stack's containers are left running.** An import adopts what an owner already ran, with no
  repository and nothing Nexul created; removing it would delete someone else's workload on the strength of a button
  labelled for Nexul's own. The confirm says so. Attaching a repository makes the stack managed, and deleting it then
  removes its containers.
- **An offline runner is caught up on connect.** The frame waits in memory per machine and is sent when a runner on that
  machine connects, before queued deploys, so a slug reused in the meantime is cleared before it is deployed again. This is
  the same shape as the deploy queue, and shares its ceiling: a server restart forgets what was waiting, and those
  containers stay until removed on the machine. The confirm names the offline case; it does not promise past a restart.
- **Personal runners refuse the frame** like every other machine job, and the runner validates the slug against the slug
  alphabet because it reaches Docker's argument list.

A Cloudflare tunnel delete now removes its connections first (`DELETE .../connections`) and then the tunnel, so a connector
that outlived its stack no longer blocks it.

Rejected: persisting the pending teardowns in a table, which costs a migration and a sweeper for a case the deploy queue
already accepts losing; running the teardown as a deploy job, which would borrow the runner's one-job slot and report a
result for a deploy record that does not exist.

Decided: 2026-10-16
