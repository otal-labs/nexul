# 04: How a personal runner keeps and serves its own logs, and the share's Read logs

Type: grilling
Status: open
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
