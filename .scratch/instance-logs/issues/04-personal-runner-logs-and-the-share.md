# 04: How a personal runner keeps and serves its own logs, and the share's Read logs

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A personal runner keeps its logs on its computer and never ships them. They
are read live through the runner, the way a container's are
(`logs_request`, `logs_chunk` and `logs_end`, ADR 0091). The computer's
owner can always read them. Anyone else needs the computer's share to allow
"Read logs", which also unlocks the computer's containers' logs, now owner
only (`ownComputer` in `internal/pairing/containers.go`). How does it work?

- Where the runner's logs live on each OS the personal runner ships for: the
  journal on Linux, a file under launchd on macOS, the Windows service host.
  Does the runner read its own journal, or should it also write a bounded
  file of its own that it can serve on every OS the same way?
- The runner message that asks for them: a new target on `logs_request`, or
  a message of its own. Tail and follow, as for containers.
- The "Read logs" action on a computer's share (#627): its name in the
  share's rule, off by default, and how `dockerComputer` and the new runner
  read check it instead of ownership alone.
- The MCP surface: extend `computer_list` (beside `container_logs`) rather
  than add a tool. The HTTP route and the `/ws` stream, beside the
  container-log ones.
- Masking. A computer's container logs reach their owner unmasked ("the
  output is their own"). Once someone else can read them, are env values
  masked for them as stack logs are?
- What a server runner's live logs mean once ticket 02 ships them: whether
  the same live read is offered there too, or the Logs area is its only road.

Technical: answered in this ticket, not grilled with the owner.

## Answer

- **The runner keeps its own file on every OS.** It doesn't read the OS's own log. Today a
  personal runner's output lands in a different place per OS: the journal on Linux, a file under
  `Paths.Logs` on macOS, and `service.log` in its unit directory on Windows. Reading the Linux
  journal would also turn on the person's journal access. Instead, in personal mode the runner's
  logger fans out to one more handler: JSON lines in `runner.log` in its own directory, mode 0600,
  rotated at 5 MiB with one previous file kept. The OS-specific output stays as it is.
- **Tail reads the file; follow subscribes to the logger.** A tail reads the last lines of
  `runner.log`, so the lines from before a crash and restart are still there. A follow listens to
  the logger itself, which hands each new record to its followers in process. Nothing polls the
  file.
- **The frame: `logs_request` with `target: "runner"`.** With that target the container is not
  required and must be empty. `logs_chunk` and `logs_end` come back unchanged, with each record
  rendered as one line (`time level message key=value…`) and passed through `redact.Tokens`. The
  limits are the container ones of ADR 0091: a 1000-line tail, the follow rate cap, and 16 streams.
  The server sends it only to personal runners. A machine runner's logs are read from the Logs area
  (ticket 03), one road each.
- **The share action: `computers:read_logs` ("Read logs").** It is added to `ComputerActions`, off
  by default, and like the Runs it implies See (`seeWithRun`, and See going takes it with it in
  `computerRule`). It covers the runner's logs, the computer's container list, and each container's
  logs. `dockerComputer` and the new runner-log read check "the owner, or a share that allows
  `computers:read_logs`" in place of `ownComputer`. A sharee without it gets the same not-found
  answer as a computer without Docker, so a refusal doesn't say what's there.
- **Masking for sharees.** The owner still reads their own containers unmasked. For anyone else
  the server sets `mask: true` on the request. The runner then reads the container's env through
  `docker inspect` and masks each value of six characters or more in the output, as stack logs
  do with the stack's env. Runner logs need no env mask; redaction covers them.
- **Surfaces.** In MCP, `computer_list` with `id` takes `runner_logs` (`lines`, as for
  `container_logs`), so no new tool is needed. For HTTP and `/ws`, the runner's stream sits beside
  the container-log routes in `pairing/handler.go` and `wire_computer_streams.go`, under
  `.../runner/logs`. The permissions panel for a computer shows the new action as "Read logs:
  this runner's logs and its Docker containers' logs".
